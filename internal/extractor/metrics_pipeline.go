package extractor

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"
    
    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
)

type MetricsPipeline struct {
    metricsAPI     *api.MetricsAPI
    kafkaProducer  *kafka.Producer
    client         *api.Client 
    
    network        string 
    chains         []string      
    interval       time.Duration 
    
    stop           chan struct{}
    mutex          sync.Mutex
    running        bool
    
    lastRunTime    time.Time
}

func NewMetricsPipeline(
    metricsAPI *api.MetricsAPI,
    kafkaProducer *kafka.Producer,
    network string,
    interval time.Duration,
) *MetricsPipeline {
    return &MetricsPipeline{
        metricsAPI:    metricsAPI,
        kafkaProducer: kafkaProducer,
        network:       network,
        interval:      interval,
        stop:          make(chan struct{}),
    }
}

func (m *MetricsPipeline) SetChains(chains []string) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.chains = chains
}

func (m *MetricsPipeline) Start(ctx context.Context) error {
    m.mutex.Lock()
    if m.running {
        m.mutex.Unlock()
        return fmt.Errorf("pipeline de métricas já está em execução")
    }
    m.running = true
    m.lastRunTime = time.Time{}
    m.mutex.Unlock()
    
    log.Printf("Iniciando pipeline de métricas para rede: %s", m.network)
    
    if err := m.collectMetrics(ctx); err != nil {
        log.Printf("Aviso: Coleta inicial de métricas falhou: %v", err)
    }
    
    go m.runPeriodic(ctx)
    
    return nil
}

func (m *MetricsPipeline) Status() map[string]interface{} {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    lastRunStr := "Nunca"
    if !m.lastRunTime.IsZero() {
        lastRunStr = m.lastRunTime.Format(time.RFC3339)
    }
    
    nextRunStr := "Não agendado"
    if m.running && !m.lastRunTime.IsZero() {
        nextRun := m.lastRunTime.Add(m.interval)
        nextRunStr = nextRun.Format(time.RFC3339)
    }
    
    return map[string]interface{}{
        "running":      m.running,
        "network":      m.network,
        "chainCount":   len(m.chains),
        "interval":     m.interval.String(),
        "lastRunTime":  lastRunStr,
        "nextRunTime":  nextRunStr,
    }
}

func (m *MetricsPipeline) Stop() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if !m.running {
        return
    }
    
    close(m.stop)
    m.running = false
    log.Printf("Interrompendo pipeline de métricas")
}

func (m *MetricsPipeline) runPeriodic(ctx context.Context) {
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-m.stop:
            return
        case <-ticker.C:
            if err := m.collectMetrics(ctx); err != nil {
                log.Printf("Erro ao coletar métricas: %v", err)
            }
        }
    }
}

func (m *MetricsPipeline) collectMetrics(ctx context.Context) error {
    start := time.Now()
    log.Printf("Iniciando coleta de métricas")
    
    m.mutex.Lock()
    m.lastRunTime = start
    m.mutex.Unlock()
    
    if err := m.collectStakingMetrics(ctx); err != nil {
        log.Printf("Erro ao coletar métricas de staking: %v", err)
    }
    if err := m.collectTeleporterMetrics(ctx); err != nil {
        log.Printf("Erro ao coletar métricas de teleporter: %v", err)
    }
    if len(m.chains) > 0 {
        for _, chainID := range m.chains {
            if err := m.collectChainSpecificMetrics(ctx, chainID); err != nil {
                log.Printf("Erro ao coletar métricas para cadeia %s: %v", chainID, err)
            }
        }
    }
    
    log.Printf("Coleta de métricas concluída em %s", time.Since(start))
    return nil
}

func (m *MetricsPipeline) collectStakingMetrics(ctx context.Context) error {
    stakingMetrics, err := m.client.GetStakingMetrics(m.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar métricas de staking: %w", err)
    }
    
    metricsData := map[string]interface{}{
        "type":           "staking",
        "validatorCount": stakingMetrics.ValidatorCount,
        "delegatorCount": stakingMetrics.DelegatorCount,
        "totalStaked":    stakingMetrics.TotalStaked,
        "timestamp":      time.Now().Unix(),
        "network":        m.network,
    }

    data, err := json.Marshal(metricsData)
    if err != nil {
        return fmt.Errorf("erro ao serializar métricas de staking: %w", err)
    }
    
    m.kafkaProducer.PublishMetrics(data)
    
    return nil
}

func (m *MetricsPipeline) collectTeleporterMetrics(ctx context.Context) error {
    teleporterMetrics := []string{
        "activeTeleporterBridges",
        "teleporterTransactions",
        "totalBridgedBtc",
    }
    
    for _, metric := range teleporterMetrics {
        value, err := m.client.GetTeleporterMetric("43114", metric)
        if err != nil {
            log.Printf("Erro ao buscar métrica teleporter %s: %v", metric, err)
            continue
        }
        metricsData := map[string]interface{}{
            "type":      "teleporter",
            "metric":    metric,
            "value":     value,
            "timestamp": time.Now().Unix(),
        }
        data, err := json.Marshal(metricsData)
        if err != nil {
            log.Printf("Erro ao serializar métrica teleporter %s: %v", metric, err)
            continue
        }
        
        m.kafkaProducer.PublishMetrics(data)
    }
    
    return nil
}

func (m *MetricsPipeline) collectChainSpecificMetrics(ctx context.Context, chainID string) error {
    chainMetrics := []api.EVMChainMetric{
        api.ActiveAddresses,
        api.TxCount,
        api.AvgTps,
        api.AvgGasPrice,
        api.FeesPaid,
    }
    
    endTime := time.Now().Unix()
    startTime := endTime - 86400 
    
    params := &api.MetricsParams{
        StartTimestamp: startTime,
        EndTimestamp:   endTime,
        TimeInterval:   api.TimeIntervalHour,
        PageSize:       24, 
    }

    for _, metric := range chainMetrics {
        metricsResp, err := m.metricsAPI.GetEVMChainMetrics(chainID, metric, params)
        if err != nil {
            log.Printf("Erro ao buscar métrica %s para cadeia %s: %v", metric, chainID, err)
            continue
        }
        for _, dataPoint := range metricsResp.Results {
            metricData := map[string]interface{}{
                "chainId":    chainID,
                "metric":     string(metric),
                "timestamp":  dataPoint.Timestamp,
                "value":      dataPoint.Value,
                "interval":   "hour",
                "collectTime": time.Now().Unix(),
            }

            data, err := json.Marshal(metricData)
            if err != nil {
                log.Printf("Erro ao serializar ponto de métrica: %v", err)
                continue
            }
            
            switch metric {
            case api.ActiveAddresses, api.TxCount:
                // Activity metrics go to the activity metrics topic
                m.kafkaProducer.PublishActivityMetrics(data)
            case api.AvgTps:
                // Performance metrics go to the performance metrics topic
                m.kafkaProducer.PublishPerformanceMetrics(data)
            case api.AvgGasPrice, api.FeesPaid:
                // Gas-related metrics go to the gas metrics topic
                m.kafkaProducer.PublishGasMetrics(data)
            default:
                // Default metrics topic for any other metrics
                m.kafkaProducer.PublishMetrics(data)
            }
        }
    }
    
    return nil
}
