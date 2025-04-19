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

// ChainPipeline manages data extraction at the chain level
type ChainPipeline struct {
    dataAPI       *api.DataAPI
    metricsAPI    *api.MetricsAPI
    kafkaProducer *kafka.Producer
    
    // Configuration
    network       string           // "mainnet", "fuji", etc.
    interval      time.Duration    // How frequently to fetch chain data
    
    // Control
    stop          chan struct{}
    mutex         sync.Mutex
    running       bool
}

// NewChainPipeline creates a new chain data pipeline
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

// Start initiates the chain data extraction process
func (c *ChainPipeline) Start(ctx context.Context) error {
    c.mutex.Lock()
    if c.running {
        c.mutex.Unlock()
        return fmt.Errorf("chain pipeline is already running")
    }
    c.running = true
    c.mutex.Unlock()
    
    log.Printf("Starting chain data pipeline for network: %s", c.network)
    
    // Performs the initial extraction
    if err := c.extractChainData(ctx); err != nil {
        log.Printf("Warning: Initial chain data extraction failed: %v", err)
        // Continue anyway, as this is just the initial execution
    }
    
    // Start periodic extraction
    go c.runPeriodic(ctx)
    
    return nil
}

// Stop interrupts the chain data extraction process
func (c *ChainPipeline) Stop() {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if !c.running {
        return
    }
    
    close(c.stop)
    c.running = false
    log.Printf("Stopping chain data pipeline")
}

// runPeriodic executes the chain data extraction process periodically
func (c *ChainPipeline) runPeriodic(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-c.stop:
            return
        case <-ticker.C:
            if err := c.extractChainData(ctx); err != nil {
                log.Printf("Error extracting chain data: %v", err)
                // Continue with the next interval
            }
        }
    }
}

// extractChainData extracts and processes chain data
func (c *ChainPipeline) extractChainData(ctx context.Context) error {
    // Fetch chains
    chains, err := c.dataAPI.GetChains()
    if err != nil {
        return fmt.Errorf("failed to fetch chains: %w", err)
    }
    
    log.Printf("Found %d chains for processing", len(chains))
    
    // Process each chain
    var wg sync.WaitGroup
    results := make(chan *types.Chain, len(chains))
    
    for _, chain := range chains {
        wg.Add(1)
        go func(ch types.Chain) {
            defer wg.Done()
            
            // Fetch detailed chain data
            chainData, err := c.dataAPI.GetChainByID(ch.ChainID)
            if err != nil {
                log.Printf("Error fetching chain details %s: %v", ch.ChainID, err)
                return
            }
            
            results <- chainData
        }(chain)
    }
    
    // Wait for all chains to be processed
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Publish chain data to Kafka
    for chainData := range results {
        c.kafkaProducer.PublishChain(chainData)
    }
    
    // Extract data from the primary network (validators, delegators, etc.)
    if err := c.extractPrimaryNetworkData(ctx); err != nil {
        log.Printf("Error extracting data from the primary network: %v", err)
        // Continue with other extractions
    }
    
    // Extract subnet data
    if err := c.extractSubnetData(ctx); err != nil {
        log.Printf("Error extracting subnet data: %v", err)
        // Continue with other extractions
    }
    
    return nil
}

// extractPrimaryNetworkData extracts data about the Primary Network
func (c *ChainPipeline) extractPrimaryNetworkData(ctx context.Context) error {
    // Extract validator data
    validators, err := c.dataAPI.GetValidators(c.network)
    if err != nil {
        return fmt.Errorf("failed to fetch validators: %w", err)
    }
    
    log.Printf("Processing %d validators", len(validators))
    
    // Publish validator data to Kafka
    for _, validator := range validators {
        c.kafkaProducer.PublishValidator(validator)
    }
    
    // Extract delegator data
    delegators, err := c.dataAPI.GetDelegators(c.network)
    if err != nil {
        return fmt.Errorf("failed to fetch delegators: %w", err)
    }
    
    log.Printf("Processing %d delegators", len(delegators))
    
    // Publish delegator data to Kafka
    for _, delegator := range delegators {
        c.kafkaProducer.PublishDelegator(delegator)
    }
    
    return nil
}

// extractSubnetData extracts data about Subnets
func (c *ChainPipeline) extractSubnetData(ctx context.Context) error {
    // Extract subnets
    subnets, err := c.dataAPI.GetSubnets(c.network)
    if err != nil {
        return fmt.Errorf("failed to fetch subnets: %w", err)
    }
    
    log.Printf("Processing %d subnets", len(subnets))
    
    // Publish subnet data to Kafka
    for _, subnet := range subnets {
        c.kafkaProducer.PublishSubnet(subnet)
    }
    
    // Extract blockchains
    blockchains, err := c.dataAPI.GetBlockchains(c.network)
    if err != nil {
        return fmt.Errorf("failed to fetch blockchains: %w", err)
    }
    
    log.Printf("Processing %d blockchains", len(blockchains))
    
    // Publish blockchain data to Kafka
    for _, blockchain := range blockchains {
        c.kafkaProducer.PublishBlockchain(blockchain)
    }
    
    return nil
}
