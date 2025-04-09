package extractor

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
    "github.com/Panorama-Block/avax/internal/types"
)

// ChainPipeline gerencia a extração de dados no nível de cadeia
type ChainPipeline struct {
    dataAPI       *api.DataAPI
    metricsAPI    *api.MetricsAPI
    kafkaProducer *kafka.Producer
    
    // Configuração 
    network       string           // "mainnet", "fuji", etc.
    interval      time.Duration    // Com que frequência buscar dados da cadeia
    
    // Controle
    stop          chan struct{}
    mutex         sync.Mutex
    running       bool
}

// NewChainPipeline cria um novo pipeline de dados de cadeia
func NewChainPipeline(
    dataAPI *api.DataAPI,
    metricsAPI *api.MetricsAPI,
    kafkaProducer *kafka.Producer,
    network string,
    interval time.Duration,
) *ChainPipeline {
    return &ChainPipeline{
        dataAPI:       dataAPI,
        metricsAPI:    metricsAPI, 
        kafkaProducer: kafkaProducer,
        network:       network,
        interval:      interval,
        stop:          make(chan struct{}),
    }
}

// Start inicia o processo de extração de dados de cadeia
func (c *ChainPipeline) Start(ctx context.Context) error {
    c.mutex.Lock()
    if c.running {
        c.mutex.Unlock()
        return fmt.Errorf("pipeline de cadeia já está em execução")
    }
    c.running = true
    c.mutex.Unlock()
    
    log.Printf("Iniciando pipeline de dados de cadeia para rede: %s", c.network)
    
    // Executa a extração inicial
    if err := c.extractChainData(ctx); err != nil {
        log.Printf("Aviso: Extração inicial de dados de cadeia falhou: %v", err)
        // Continua mesmo assim, pois é apenas a execução inicial
    }
    
    // Inicia a extração periódica
    go c.runPeriodic(ctx)
    
    return nil
}

// Stop interrompe o processo de extração de dados de cadeia
func (c *ChainPipeline) Stop() {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if !c.running {
        return
    }
    
    close(c.stop)
    c.running = false
    log.Printf("Interrompendo pipeline de dados de cadeia")
}

// runPeriodic executa o processo de extração de dados de cadeia periodicamente
func (c *ChainPipeline) runPeriodic(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-c.stop:
            return
        case <-ticker.C:
            if err := c.extractChainData(ctx); err != nil {
                log.Printf("Erro ao extrair dados de cadeia: %v", err)
                // Continua com o próximo intervalo
            }
        }
    }
}

// extractChainData extrai e processa dados de cadeia
func (c *ChainPipeline) extractChainData(ctx context.Context) error {
    // Buscar cadeias
    chains, err := c.dataAPI.GetChains()
    if err != nil {
        return fmt.Errorf("falha ao buscar cadeias: %w", err)
    }
    
    log.Printf("Encontradas %d cadeias para processamento", len(chains))
    
    // Processa cada cadeia
    var wg sync.WaitGroup
    results := make(chan *types.Chain, len(chains))
    
    for _, chain := range chains {
        wg.Add(1)
        go func(ch types.Chain) {
            defer wg.Done()
            
            // Busca dados detalhados da cadeia
            chainData, err := c.dataAPI.GetChainByID(ch.ChainID)
            if err != nil {
                log.Printf("Erro ao buscar detalhes da cadeia %s: %v", ch.ChainID, err)
                return
            }
            
            results <- chainData
        }(chain)
    }
    
    // Aguarda o processamento de todas as cadeias
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Publica os dados das cadeias no Kafka
    for chainData := range results {
        c.kafkaProducer.PublishChain(chainData)
    }
    
    // Extrai dados da rede primária (validadores, delegadores, etc.)
    if err := c.extractPrimaryNetworkData(ctx); err != nil {
        log.Printf("Erro ao extrair dados da rede primária: %v", err)
        // Continua com outras extrações
    }
    
    // Extrai dados de subnets
    if err := c.extractSubnetData(ctx); err != nil {
        log.Printf("Erro ao extrair dados de subnet: %v", err)
        // Continua com outras extrações
    }
    
    return nil
}

// extractPrimaryNetworkData extrai dados sobre a Rede Primária
func (c *ChainPipeline) extractPrimaryNetworkData(ctx context.Context) error {
    // Extrai dados de validadores
    validators, err := c.dataAPI.GetValidators(c.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar validadores: %w", err)
    }
    
    log.Printf("Processando %d validadores", len(validators))
    
    // Publica dados de validadores no Kafka
    for _, validator := range validators {
        c.kafkaProducer.PublishValidator(validator)
    }
    
    // Extrai dados de delegadores
    delegators, err := c.dataAPI.GetDelegators(c.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar delegadores: %w", err)
    }
    
    log.Printf("Processando %d delegadores", len(delegators))
    
    // Publica dados de delegadores no Kafka
    for _, delegator := range delegators {
        c.kafkaProducer.PublishDelegator(delegator)
    }
    
    return nil
}

// extractSubnetData extrai dados sobre Subnets
func (c *ChainPipeline) extractSubnetData(ctx context.Context) error {
    // Extrai subnets
    subnets, err := c.dataAPI.GetSubnets(c.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar subnets: %w", err)
    }
    
    log.Printf("Processando %d subnets", len(subnets))
    
    // Publica dados de subnet no Kafka
    for _, subnet := range subnets {
        c.kafkaProducer.PublishSubnet(subnet)
    }
    
    // Extrai blockchains
    blockchains, err := c.dataAPI.GetBlockchains(c.network)
    if err != nil {
        return fmt.Errorf("falha ao buscar blockchains: %w", err)
    }
    
    log.Printf("Processando %d blockchains", len(blockchains))
    
    // Publica dados de blockchain no Kafka
    for _, blockchain := range blockchains {
        c.kafkaProducer.PublishBlockchain(blockchain)
    }
    
    return nil
}
