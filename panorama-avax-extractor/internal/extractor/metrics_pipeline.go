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

// MetricsPipeline gerencia a coleta e processamento de métricas
type MetricsPipeline struct {
    metricsAPI     *api.MetricsAPI
    kafkaProducer  *kafka.Producer
    client         *api.Client // Adicionando referência direta ao cliente para chamadas diretas
    
    // Configuração
    network        string        // "mainnet", "fuji", etc.
    chains         []string      // Lista de chainIDs para monitorar
    interval       time.Duration // Com que frequência buscar métricas
    
    // Controle
    stop           chan struct{}
    mutex          sync.Mutex
    running        bool
    
    // Estado
    lastRunTime    time.Time
}

// NewMetricsPipeline cria um novo pipeline de métricas
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

// SetChains define as cadeias a serem monitoradas
func (m *MetricsPipeline) SetChains(chains []string) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.chains = chains
}

// Start inicia o processo de coleta de métricas
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
    
    // Executa a coleta inicial
    if err := m.collectMetrics(ctx); err != nil {
        log.Printf("Aviso: Coleta inicial de métricas falhou: %v", err)
        // Continua mesmo assim, pois é apenas a execução inicial
    }
    
    // Inicia a coleta periódica
    go m.runPeriodic(ctx)
    
    return nil
}

// Status retorna o status atual do pipeline
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

// Stop interrompe o processo de coleta de métricas
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

// runPeriodic executa o processo de coleta de métricas periodicamente
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
                // Continua com o próximo intervalo
            }
        }
    }
}

// collectMetrics coleta e processa métricas
func (m *MetricsPipeline) collectMetrics(ctx context.Context) error {
    start := time.Now()
    log.Printf("Iniciando coleta de métricas")
    
    // Atualiza lastRunTime
    m.mutex.Lock()
    m.lastRunTime = start
    m.mutex.Unlock()
    
    // Coleta métricas de staking
    if err := m.collectStakingMetrics(ctx); err != nil {
        log.Printf("Erro ao coletar métricas de staking: %v", err)
        // Continua com outras métricas
    }
    
    // Coleta métricas de teleporter (ponte BTC)
    if err := m.collectTeleporterMetrics(ctx); err != nil {
        log.Printf("Erro ao coletar métricas de teleporter: %v", err)
        // Continua com outras métricas
    }
    
    // Coleta métricas de cadeias específicas
    if len(m.chains) > 0 {
        for _, chainID := range m.chains {
            if err := m.collectChainSpecificMetrics(ctx, chainID); err != nil {
                log.Printf("Erro ao coletar métricas para cadeia %s: %v", chainID, err)
                // Continua com a próxima cadeia
            }
        }
    }
    
    log.Printf("Coleta de métricas concluída em %s", time.Since(start))
    return nil
}

// collectStakingMetrics coleta métricas de staking
func (m *MetricsPipeline) collectStakingMetrics(ctx context.Context) error {
    // Usar o cliente da API que é acessível
    stakingMetrics, err := m.client.GetStakingMetrics(m.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar métricas de staking: %w", err)
    }
    
    // Prepara dados para o Kafka
    metricsData := map[string]interface{}{
        "type":           "staking",
        "validatorCount": stakingMetrics.ValidatorCount,
        "delegatorCount": stakingMetrics.DelegatorCount,
        "totalStaked":    stakingMetrics.TotalStaked,
        "timestamp":      time.Now().Unix(),
        "network":        m.network,
    }
    
    // Converte para JSON
    data, err := json.Marshal(metricsData)
    if err != nil {
        return fmt.Errorf("erro ao serializar métricas de staking: %w", err)
    }
    
    // Publica no Kafka
    m.kafkaProducer.PublishMetrics(data)
    
    return nil
}

// collectTeleporterMetrics coleta métricas relacionadas ao Teleporter
func (m *MetricsPipeline) collectTeleporterMetrics(ctx context.Context) error {
    // Lista de métricas do teleporter para coletar
    teleporterMetrics := []string{
        "activeTeleporterBridges",
        "teleporterTransactions",
        "totalBridgedBtc",
    }
    
    for _, metric := range teleporterMetrics {
        // Usar o cliente da API que é acessível
        value, err := m.client.GetTeleporterMetric("43114", metric)
        if err != nil {
            log.Printf("Erro ao buscar métrica teleporter %s: %v", metric, err)
            continue
        }
        
        // Prepara dados para o Kafka
        metricsData := map[string]interface{}{
            "type":      "teleporter",
            "metric":    metric,
            "value":     value,
            "timestamp": time.Now().Unix(),
        }
        
        // Converte para JSON
        data, err := json.Marshal(metricsData)
        if err != nil {
            log.Printf("Erro ao serializar métrica teleporter %s: %v", metric, err)
            continue
        }
        
        // Publica no Kafka
        m.kafkaProducer.PublishMetrics(data)
    }
    
    return nil
}

// collectChainSpecificMetrics coleta métricas específicas de uma cadeia
func (m *MetricsPipeline) collectChainSpecificMetrics(ctx context.Context, chainID string) error {
    // Lista de métricas de cadeia para coletar
    chainMetrics := []api.EVMChainMetric{
        api.ActiveAddresses,
        api.TxCount,
        api.AvgTps,
        api.AvgGasPrice,
        api.FeesPaid,
    }
    
    // Define o intervalo de tempo para métricas (últimas 24 horas)
    endTime := time.Now().Unix()
    startTime := endTime - 86400 // 24 horas em segundos
    
    // Parâmetros para coleta de métricas
    params := &api.MetricsParams{
        StartTimestamp: startTime,
        EndTimestamp:   endTime,
        TimeInterval:   api.TimeIntervalHour,
        PageSize:       24, // 24 pontos para intervalo de hora
    }
    
    // Coleta cada métrica
    for _, metric := range chainMetrics {
        metricsResp, err := m.metricsAPI.GetEVMChainMetrics(chainID, metric, params)
        if err != nil {
            log.Printf("Erro ao buscar métrica %s para cadeia %s: %v", metric, chainID, err)
            continue
        }
        
        // Processa cada ponto de dados
        for _, dataPoint := range metricsResp.Results {
            // Prepara dados para o Kafka
            metricData := map[string]interface{}{
                "chainId":    chainID,
                "metric":     string(metric),
                "timestamp":  dataPoint.Timestamp,
                "value":      dataPoint.Value,
                "interval":   "hour",
                "collectTime": time.Now().Unix(),
            }
            
            // Converte para JSON
            data, err := json.Marshal(metricData)
            if err != nil {
                log.Printf("Erro ao serializar ponto de métrica: %v", err)
                continue
            }
            
            // Publica no Kafka
            m.kafkaProducer.PublishMetrics(data)
        }
    }
    
    return nil
}
