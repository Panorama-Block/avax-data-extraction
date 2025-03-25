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

// ChainPipeline handles extraction of chain-level data
type ChainPipeline struct {
	dataAPI       *api.DataAPI
	metrics       *api.MetricsAPI
	kafkaProducer *kafka.Producer
	
	// Configuration 
	network       string           // "mainnet", "fuji", etc.
	chains        []string         // List of chainIDs to monitor
	interval      time.Duration    // How often to fetch chain data
	
	// Control
	stop          chan struct{}
	mutex         sync.Mutex
	running       bool
}

// NewChainPipeline creates a new chain data pipeline
func NewChainPipeline(
	dataAPI *api.DataAPI,
	metrics *api.MetricsAPI,
	kafkaProducer *kafka.Producer,
	network string,
	interval time.Duration,
) *ChainPipeline {
	return &ChainPipeline{
		dataAPI:       dataAPI,
		metrics:       metrics, 
		kafkaProducer: kafkaProducer,
		network:       network,
		interval:      interval,
		stop:          make(chan struct{}),
	}
}

// Start begins the chain data extraction process
func (c *ChainPipeline) Start(ctx context.Context) error {
	c.mutex.Lock()
	if c.running {
		c.mutex.Unlock()
		return fmt.Errorf("chain pipeline already running")
	}
	c.running = true
	c.mutex.Unlock()
	
	log.Printf("Starting chain data pipeline for network: %s", c.network)
	
	// First fetch the list of chains to monitor
	if err := c.fetchChains(ctx); err != nil {
		return fmt.Errorf("failed to fetch chains: %w", err)
	}
	
	// Run initial extraction
	if err := c.extractChainData(ctx); err != nil {
		log.Printf("Warning: Initial chain data extraction failed: %v", err)
		// Continue anyway as this is just the initial run
	}
	
	// Start the periodic extraction
	go c.runPeriodic(ctx)
	
	return nil
}

// Stop halts the chain data extraction process
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

// fetchChains fetches and stores the list of chains to monitor
func (c *ChainPipeline) fetchChains(ctx context.Context) error {
	// Get the list of chains from the API
	chainsResp, err := c.dataAPI.GetChains(c.network, &api.PaginationParams{
		PageSize:  100, // Adjust as needed
		SortOrder: "asc",
	})
	if err != nil {
		return fmt.Errorf("failed to get chains: %w", err)
	}
	
	chains := make([]string, 0, len(chainsResp.Chains))
	for _, chain := range chainsResp.Chains {
		chains = append(chains, chain.ChainID)
		
		// Publish chain information to Kafka
		chainData := map[string]interface{}{
			"chainId":        chain.ChainID,
			"name":           chain.Name,
			"network":        chain.Network,
			"isTestnet":      chain.IsTestnet,
			"isSubnet":       chain.IsSubnet,
			"isL1":           chain.IsL1,
			"subnetId":       chain.SubnetID,
			"nativeCurrency": chain.NativeCurrency,
			"rpcUrl":         chain.RpcURL,
			"explorerUrl":    chain.ExplorerURL,
			"updateTime":     time.Now().Unix(),
		}
		
		chainDataJSON, err := json.Marshal(chainData)
		if err != nil {
			log.Printf("Error marshaling chain data: %v", err)
			continue
		}
		
		if err := c.kafkaProducer.PublishChainInfo(chain.ChainID, chainDataJSON); err != nil {
			log.Printf("Error publishing chain info to Kafka: %v", err)
			// Continue anyway
		}
	}
	
	c.chains = chains
	log.Printf("Found %d chains in network %s", len(chains), c.network)
	
	return nil
}

// runPeriodic runs the chain data extraction process periodically
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
	// Extract Primary Network data
	if err := c.extractPrimaryNetworkData(ctx); err != nil {
		log.Printf("Error extracting Primary Network data: %v", err)
		// Continue with other extractions
	}
	
	// Extract Subnet data
	if err := c.extractSubnetData(ctx); err != nil {
		log.Printf("Error extracting Subnet data: %v", err)
		// Continue with other extractions
	}
	
	// Extract chain-specific metrics for each chain
	for _, chainID := range c.chains {
		if err := c.extractChainMetrics(ctx, chainID); err != nil {
			log.Printf("Error extracting metrics for chain %s: %v", chainID, err)
			// Continue with the next chain
		}
	}
	
	return nil
}

// extractPrimaryNetworkData extracts data about the Primary Network
func (c *ChainPipeline) extractPrimaryNetworkData(ctx context.Context) error {
	// Extract validator data
	validatorsResp, err := c.dataAPI.GetValidators(c.network, nil, "", &api.PaginationParams{
		PageSize: 1000, // Adjust as needed
	})
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}
	
	log.Printf("Processing %d validators", len(validatorsResp.Validators))
	
	// Publish validator data to Kafka
	for _, validator := range validatorsResp.Validators {
		validatorData := map[string]interface{}{
			"nodeId":            validator.NodeID,
			"startTime":         validator.StartTime,
			"endTime":           validator.EndTime,
			"stakeAmount":       validator.StakeAmount,
			"potentialReward":   validator.PotentialReward,
			"delegationFee":     validator.DelegationFee,
			"uptime":            validator.Uptime,
			"connected":         validator.Connected,
			"stakeOwnerAddress": validator.StakeOwnerAddress,
			"network":           c.network,
			"updateTime":        time.Now().Unix(),
		}
		
		if len(validator.SubnetIDs) > 0 {
			validatorData["subnetIds"] = validator.SubnetIDs
		}
		
		validatorDataJSON, err := json.Marshal(validatorData)
		if err != nil {
			log.Printf("Error marshaling validator data: %v", err)
			continue
		}
		
		if err := c.kafkaProducer.PublishValidator(c.network, validator.NodeID, validatorDataJSON); err != nil {
			log.Printf("Error publishing validator data to Kafka: %v", err)
			// Continue with the next validator
		}
	}
	
	// Extract delegator data
	delegatorsResp, err := c.dataAPI.GetDelegators(c.network, &api.PaginationParams{
		PageSize: 1000, // Adjust as needed
	})
	if err != nil {
		return fmt.Errorf("failed to get delegators: %w", err)
	}
	
	log.Printf("Processing %d delegators", len(delegatorsResp.Delegators))
	
	// Publish delegator data to Kafka
	for _, delegator := range delegatorsResp.Delegators {
		delegatorData := map[string]interface{}{
			"nodeId":              delegator.NodeID,
			"startTime":           delegator.StartTime,
			"endTime":             delegator.EndTime,
			"stakeAmount":         delegator.StakeAmount,
			"potentialReward":     delegator.PotentialReward,
			"rewardOwnerAddress":  delegator.RewardOwnerAddress,
			"stakeOwnerAddress":   delegator.StakeOwnerAddress,
			"network":             c.network,
			"updateTime":          time.Now().Unix(),
		}
		
		delegatorDataJSON, err := json.Marshal(delegatorData)
		if err != nil {
			log.Printf("Error marshaling delegator data: %v", err)
			continue
		}
		
		if err := c.kafkaProducer.PublishDelegator(c.network, delegator.StakeOwnerAddress, delegatorDataJSON); err != nil {
			log.Printf("Error publishing delegator data to Kafka: %v", err)
			// Continue with the next delegator
		}
	}
	
	return nil
}

// extractSubnetData extracts data about Subnets
func (c *ChainPipeline) extractSubnetData(ctx context.Context) error {
	// Extract subnets
	subnetsResp, err := c.dataAPI.GetSubnets(c.network, &api.PaginationParams{
		PageSize: 100, // Adjust as needed
	})
	if err != nil {
		return fmt.Errorf("failed to get subnets: %w", err)
	}
	
	log.Printf("Processing %d subnets", len(subnetsResp.Subnets))
	
	// Publish subnet data to Kafka
	for _, subnet := range subnetsResp.Subnets {
		subnetData := map[string]interface{}{
			"id":          subnet.ID,
			"controlKeys": subnet.ControlKeys,
			"threshold":   subnet.Threshold,
			"isL1":        subnet.IsL1,
			"network":     c.network,
			"updateTime":  time.Now().Unix(),
		}
		
		subnetDataJSON, err := json.Marshal(subnetData)
		if err != nil {
			log.Printf("Error marshaling subnet data: %v", err)
			continue
		}
		
		if err := c.kafkaProducer.PublishSubnet(c.network, subnet.ID, subnetDataJSON); err != nil {
			log.Printf("Error publishing subnet data to Kafka: %v", err)
			// Continue with the next subnet
		}
	}
	
	// Extract blockchains
	blockchainsResp, err := c.dataAPI.GetBlockchains(c.network, &api.PaginationParams{
		PageSize: 100, // Adjust as needed
	})
	if err != nil {
		return fmt.Errorf("failed to get blockchains: %w", err)
	}
	
	log.Printf("Processing %d blockchains", len(blockchainsResp.Blockchains))
	
	// Publish blockchain data to Kafka
	for _, blockchain := range blockchainsResp.Blockchains {
		blockchainData := map[string]interface{}{
			"id":         blockchain.ID,
			"name":       blockchain.Name,
			"subnetId":   blockchain.SubnetID,
			"vmName":     blockchain.VMName,
			"network":    c.network,
			"updateTime": time.Now().Unix(),
		}
		
		blockchainDataJSON, err := json.Marshal(blockchainData)
		if err != nil {
			log.Printf("Error marshaling blockchain data: %v", err)
			continue
		}
		
		if err := c.kafkaProducer.PublishBlockchain(c.network, blockchain.ID, blockchainDataJSON); err != nil {
			log.Printf("Error publishing blockchain data to Kafka: %v", err)
			// Continue with the next blockchain
		}
	}
	
	return nil
}

// extractChainMetrics extracts metrics for a specific chain
func (c *ChainPipeline) extractChainMetrics(ctx context.Context, chainID string) error {
	// Define the time range for metrics (last 24 hours)
	endTime := time.Now().Unix()
	startTime := endTime - 86400 // 24 hours in seconds
	
	// List of metrics to extract
	metrics := []api.EVMChainMetric{
		api.ActiveAddresses,
		api.TxCount,
		api.GasUsed,
		api.AvgTps,
		api.MaxTps,
		api.AvgGasPrice,
		api.FeesPaid,
	}
	
	// Extract each metric
	for _, metric := range metrics {
		metricsResp, err := c.metrics.GetEVMChainMetrics(chainID, metric, &api.MetricsParams{
			StartTimestamp: startTime,
			EndTimestamp:   endTime,
			TimeInterval:   api.TimeIntervalHour,
			PageSize:       24, // 24 hours
		})
		if err != nil {
			log.Printf("Error getting metric %s for chain %s: %v", metric, chainID, err)
			continue
		}
		
		// Process and publish each data point
		for _, dataPoint := range metricsResp.Results {
			metricData := map[string]interface{}{
				"chainId":    chainID,
				"metric":     string(metric),
				"timestamp":  dataPoint.Timestamp,
				"value":      dataPoint.Value,
				"updateTime": time.Now().Unix(),
			}
			
			metricDataJSON, err := json.Marshal(metricData)
			if err != nil {
				log.Printf("Error marshaling metric data: %v", err)
				continue
			}
			
			if err := c.kafkaProducer.PublishMetric(chainID, string(metric), metricDataJSON); err != nil {
				log.Printf("Error publishing metric data to Kafka: %v", err)
				// Continue with the next data point
			}
		}
	}
	
	return nil
}

// Status returns the current status of the pipeline
func (c *ChainPipeline) Status() map[string]interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	return map[string]interface{}{
		"running":      c.running,
		"network":      c.network,
		"chainCount":   len(c.chains),
		"chains":       c.chains,
		"interval":     c.interval.String(),
	}
}