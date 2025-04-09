package extractor

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
)

// StartCronJobs inicia trabalhos cron para extração de dados
func StartCronJobs(client *api.Client, producer *kafka.Producer, network string) {
    // Criar APIs a partir do cliente base
    dataAPI := api.NewDataAPI(client)
    
    go func() {
        for {
            err := fetchSubnetsAndBlockchains(dataAPI, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro subnets: %v", err)
            }

            err = fetchValidators(dataAPI, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro validators: %v", err)
            }

            err = fetchDelegators(dataAPI, producer, network)
            if err != nil {
                log.Printf("[CRON] Erro delegators: %v", err)
            }

            _ = fetchTeleporterTxs(dataAPI, producer)

            time.Sleep(10 * time.Minute)
        }
    }()
}

func fetchSubnetsAndBlockchains(dataAPI *api.DataAPI, producer *kafka.Producer, network string) error {
    subnets, err := dataAPI.GetSubnets(network)
    if err != nil {
        return err
    }
    for _, s := range subnets {
        producer.PublishSubnet(s)
        det, err2 := dataAPI.GetSubnetByID(network, s.SubnetID)
        if err2 == nil && det != nil {
            producer.PublishSubnet(*det)
            for _, bc := range det.Blockchains {
                producer.PublishBlockchain(bc)
            }
        }
    }
    bcs, err := dataAPI.GetBlockchains(network)
    if err == nil {
        for _, bc := range bcs {
            producer.PublishBlockchain(bc)
        }
    }
    return nil
}

func fetchValidators(dataAPI *api.DataAPI, producer *kafka.Producer, network string) error {
    vals, err := dataAPI.GetValidators(network)
    if err != nil {
        return err
    }
    for _, v := range vals {
        producer.PublishValidator(v)
    }
    return nil
}

func fetchDelegators(dataAPI *api.DataAPI, producer *kafka.Producer, network string) error {
    delegs, err := dataAPI.GetDelegators(network)
    if err != nil {
        return err
    }
    for _, d := range delegs {
        producer.PublishDelegator(d)
    }
    return nil
}

func fetchTeleporterTxs(dataAPI *api.DataAPI, producer *kafka.Producer) error {
    txs, err := dataAPI.GetTeleporterTxs("43114", "99999")
    if err != nil {
        log.Printf("[CRON] Erro teleporter TXs: %v", err)
        return err
    }
    for _, t := range txs {
        producer.PublishBridgeTx(t)
    }
    return nil
}

// CronJobManager gerencia tarefas agendadas para extração de dados
type CronJobManager struct {
    dataAPI       *api.DataAPI
    metricsAPI    *api.MetricsAPI
    kafkaProducer *kafka.Producer
    
    // Configuração
    network       string
    
    // Controle
    stop          chan struct{}
    wg            sync.WaitGroup
    mutex         sync.Mutex
    running       bool
    
    // Trabalhos
    jobs          []*CronJob
}

// CronJob representa uma tarefa agendada
type CronJob struct {
    Name          string
    Interval      time.Duration
    LastRunTime   time.Time
    IsRunning     bool
    Task          func(context.Context) error
}

// NewCronJobManager cria um novo gerenciador de trabalhos cron
func NewCronJobManager(
    dataAPI *api.DataAPI,
    metricsAPI *api.MetricsAPI,
    kafkaProducer *kafka.Producer,
    network string,
) *CronJobManager {
    return &CronJobManager{
        dataAPI:       dataAPI,
        metricsAPI:    metricsAPI,
        kafkaProducer: kafkaProducer,
        network:       network,
        stop:          make(chan struct{}),
    }
}

// AddJob adiciona um novo trabalho ao gerenciador
func (c *CronJobManager) AddJob(name string, interval time.Duration, task func(context.Context) error) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    c.jobs = append(c.jobs, &CronJob{
        Name:     name,
        Interval: interval,
        Task:     task,
    })
    
    log.Printf("Adicionado trabalho cron: %s (intervalo: %s)", name, interval)
}

// Start inicia o gerenciador de trabalhos cron
func (c *CronJobManager) Start(ctx context.Context) error {
    c.mutex.Lock()
    if c.running {
        c.mutex.Unlock()
        return fmt.Errorf("gerenciador de trabalhos cron já está em execução")
    }
    c.running = true
    c.mutex.Unlock()
    
    log.Printf("Iniciando gerenciador de trabalhos cron com %d trabalhos", len(c.jobs))
    
    // Inicia cada trabalho em sua própria goroutine
    for i := range c.jobs {
        c.wg.Add(1)
        go c.runJob(ctx, c.jobs[i])
    }
    
    return nil
}

// Stop interrompe o gerenciador de trabalhos cron
func (c *CronJobManager) Stop() {
    c.mutex.Lock()
    if !c.running {
        c.mutex.Unlock()
        return
    }
    c.running = false
    close(c.stop)
    c.mutex.Unlock()
    
    log.Printf("Interrompendo gerenciador de trabalhos cron, aguardando conclusão...")
    c.wg.Wait()
    log.Printf("Gerenciador de trabalhos cron parado")
}

// runJob executa um trabalho em um cronograma
func (c *CronJobManager) runJob(ctx context.Context, job *CronJob) {
    defer c.wg.Done()
    
    log.Printf("Iniciando trabalho: %s", job.Name)
    
    // Executa o trabalho imediatamente na inicialização
    if err := c.executeJob(ctx, job); err != nil {
        log.Printf("Erro ao executar trabalho %s: %v", job.Name, err)
    }
    
    // Cria ticker para execução regular
    ticker := time.NewTicker(job.Interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-c.stop:
            log.Printf("Interrompendo trabalho: %s", job.Name)
            return
        case <-ticker.C:
            if err := c.executeJob(ctx, job); err != nil {
                log.Printf("Erro ao executar trabalho %s: %v", job.Name, err)
                // Continua com a próxima execução agendada
            }
        }
    }
}

// executeJob executa um único trabalho
func (c *CronJobManager) executeJob(ctx context.Context, job *CronJob) error {
    // Verifica se o trabalho já está em execução
    if job.IsRunning {
        log.Printf("Trabalho %s já está em execução, pulando esta execução", job.Name)
        return nil
    }
    
    // Marca o trabalho como em execução
    job.IsRunning = true
    defer func() {
        job.IsRunning = false
        job.LastRunTime = time.Now()
    }()
    
    log.Printf("Executando trabalho: %s", job.Name)
    
    // Executa o trabalho
    start := time.Now()
    err := job.Task(ctx)
    elapsed := time.Since(start)
    
    if err != nil {
        log.Printf("Trabalho %s falhou após %s: %v", job.Name, elapsed, err)
        return err
    }
    
    log.Printf("Trabalho %s concluído com sucesso em %s", job.Name, elapsed)
    return nil
}

// CreateDefaultJobs cria os trabalhos padrão para o pipeline de dados Avalanche
func (c *CronJobManager) CreateDefaultJobs() {
    // Adiciona trabalho para coletar validadores (a cada 10 minutos)
    c.AddJob("ValidatorCollector", 10*time.Minute, func(ctx context.Context) error {
        validators, err := c.dataAPI.GetValidators(c.network)
        if err != nil {
            return fmt.Errorf("falha ao buscar validadores: %w", err)
        }
        
        log.Printf("Processando %d validadores", len(validators))
        
        for _, validator := range validators {
            c.kafkaProducer.PublishValidator(validator)
        }
        
        return nil
    })
    
    // Adiciona trabalho para coletar delegadores (a cada 30 minutos)
    c.AddJob("DelegatorCollector", 30*time.Minute, func(ctx context.Context) error {
        delegators, err := c.dataAPI.GetDelegators(c.network)
        if err != nil {
            return fmt.Errorf("falha ao buscar delegadores: %w", err)
        }
        
        log.Printf("Processando %d delegadores", len(delegators))
        
        for _, delegator := range delegators {
            c.kafkaProducer.PublishDelegator(delegator)
        }
        
        return nil
    })
    
    // Adiciona trabalho para coletar subnets (a cada hora)
    c.AddJob("SubnetCollector", time.Hour, func(ctx context.Context) error {
        subnets, err := c.dataAPI.GetSubnets(c.network)
        if err != nil {
            return fmt.Errorf("falha ao buscar subnets: %w", err)
        }
        
        log.Printf("Processando %d subnets", len(subnets))
        
        for _, subnet := range subnets {
            c.kafkaProducer.PublishSubnet(subnet)
        }
        
        // Também coleta blockchains
        blockchains, err := c.dataAPI.GetBlockchains(c.network)
        if err != nil {
            return fmt.Errorf("falha ao buscar blockchains: %w", err)
        }
        
        log.Printf("Processando %d blockchains", len(blockchains))
        
        for _, blockchain := range blockchains {
            c.kafkaProducer.PublishBlockchain(blockchain)
        }
        
        return nil
    })
    
    // Adiciona trabalho para coletar transações de teleporter (a cada 15 minutos)
    c.AddJob("TeleporterCollector", 15*time.Minute, func(ctx context.Context) error {
        txs, err := c.dataAPI.GetTeleporterTxs("43114", "99999")
        if err != nil {
            return fmt.Errorf("falha ao buscar transações teleporter: %w", err)
        }
        
        log.Printf("Processando %d transações teleporter", len(txs))
        
        for _, tx := range txs {
            c.kafkaProducer.PublishBridgeTx(tx)
        }
        
        return nil
    })
}

// Status retorna o status atual do gerenciador de trabalhos cron
func (c *CronJobManager) Status() map[string]interface{} {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    jobStatuses := make([]map[string]interface{}, len(c.jobs))
    for i, job := range c.jobs {
        lastRunStr := "Nunca"
        if !job.LastRunTime.IsZero() {
            lastRunStr = job.LastRunTime.Format(time.RFC3339)
        }
        
        nextRunStr := "Não agendado"
        if c.running && !job.LastRunTime.IsZero() {
            nextRun := job.LastRunTime.Add(job.Interval)
            nextRunStr = nextRun.Format(time.RFC3339)
        }
        
        jobStatuses[i] = map[string]interface{}{
            "name":        job.Name,
            "interval":    job.Interval.String(),
            "lastRunTime": lastRunStr,
            "nextRunTime": nextRunStr,
            "isRunning":   job.IsRunning,
        }
    }
    
    return map[string]interface{}{
        "running":  c.running,
        "jobCount": len(c.jobs),
        "jobs":     jobStatuses,
    }
}
