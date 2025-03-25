package extractor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"encoding/json"
	"os"
	
	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/kafka"
)

// CronJobManager manages scheduled tasks for data extraction
type CronJobManager struct {
	dataAPI       *api.DataAPI
	metricsAPI    *api.MetricsAPI
	kafkaProducer *kafka.Producer
	
	// Configuration
	network       string
	
	// Control
	stop          chan struct{}
	wg            sync.WaitGroup
	mutex         sync.Mutex
	running       bool
	
	// Jobs
	jobs          []*CronJob
}

// CronJob represents a scheduled task
type CronJob struct {
	Name          string
	Interval      time.Duration
	LastRunTime   time.Time
	IsRunning     bool
	Task          func(context.Context) error
}

// NewCronJobManager creates a new cron job manager
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

// AddJob adds a new job to the manager
func (c *CronJobManager) AddJob(name string, interval time.Duration, task func(context.Context) error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.jobs = append(c.jobs, &CronJob{
		Name:     name,
		Interval: interval,
		Task:     task,
	})
	
	log.Printf("Added cron job: %s (interval: %s)", name, interval)
}

// Start begins the cron job manager
func (c *CronJobManager) Start(ctx context.Context) error {
	c.mutex.Lock()
	if c.running {
		c.mutex.Unlock()
		return fmt.Errorf("cron job manager already running")
	}
	c.running = true
	c.mutex.Unlock()
	
	log.Printf("Starting cron job manager with %d jobs", len(c.jobs))
	
	// Start each job in its own goroutine
	for i := range c.jobs {
		c.wg.Add(1)
		go c.runJob(ctx, c.jobs[i])
	}
	
	return nil
}

// Stop halts the cron job manager
func (c *CronJobManager) Stop() {
	c.mutex.Lock()
	if !c.running {
		c.mutex.Unlock()
		return
	}
	c.running = false
	close(c.stop)
	c.mutex.Unlock()
	
	log.Printf("Stopping cron job manager, waiting for jobs to complete...")
	c.wg.Wait()
	log.Printf("Cron job manager stopped")
}

// runJob runs a job on a schedule
func (c *CronJobManager) runJob(ctx context.Context, job *CronJob) {
	defer c.wg.Done()
	
	log.Printf("Starting job: %s", job.Name)
	
	// Run the job immediately on startup
	if err := c.executeJob(ctx, job); err != nil {
		log.Printf("Error running job %s: %v", job.Name, err)
	}
	
	// Create ticker for regular execution
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.stop:
			log.Printf("Stopping job: %s", job.Name)
			return
		case <-ticker.C:
			if err := c.executeJob(ctx, job); err != nil {
				log.Printf("Error running job %s: %v", job.Name, err)
				// Continue with the next scheduled execution
			}
		}
	}
}

// executeJob executes a single job
func (c *CronJobManager) executeJob(ctx context.Context, job *CronJob) error {
	// Check if the job is already running
	if job.IsRunning {
		log.Printf("Job %s is already running, skipping this execution", job.Name)
		return nil
	}
	
	// Mark the job as running
	job.IsRunning = true
	defer func() {
		job.IsRunning = false
		job.LastRunTime = time.Now()
	}()
	
	log.Printf("Executing job: %s", job.Name)
	
	// Execute the job
	start := time.Now()
	err := job.Task(ctx)
	elapsed := time.Since(start)
	
	if err != nil {
		log.Printf("Job %s failed after %s: %v", job.Name, elapsed, err)
		return err
	}
	
	log.Printf("Job %s completed successfully in %s", job.Name, elapsed)
	return nil
}

// Status returns the current status of the cron job manager
func (c *CronJobManager) Status() map[string]interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	jobStatuses := make([]map[string]interface{}, len(c.jobs))
	for i, job := range c.jobs {
		lastRunStr := "Never"
		if !job.LastRunTime.IsZero() {
			lastRunStr = job.LastRunTime.Format(time.RFC3339)
		}
		
		nextRunStr := "Not scheduled"
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

// CreateDefaultJobs creates the default jobs for the Avalanche data pipeline
func (c *CronJobManager) CreateDefaultJobs() {
	// Add job to collect validators (every 10 minutes)
	c.AddJob("ValidatorCollector", 10*time.Minute, func(ctx context.Context) error {
		return c.collectValidators(ctx)
	})
	
	// Add job to collect delegators (every 30 minutes)
	c.AddJob("DelegatorCollector", 30*time.Minute, func(ctx context.Context) error {
		return c.collectDelegators(ctx)
	})
	
	// Add job to collect subnets (every hour)
	c.AddJob("SubnetCollector", time.Hour, func(ctx context.Context) error {
		return c.collectSubnets(ctx)
	})
	
	// Add job to collect pending rewards (every hour)
	c.AddJob("PendingRewardCollector", time.Hour, func(ctx context.Context) error {
		return c.collectPendingRewards(ctx)
	})
	
	// Add job to collect X-Chain assets (every 12 hours)
	c.AddJob("XChainAssetCollector", 12*time.Hour, func(ctx context.Context) error {
		return c.collectXChainAssets(ctx)
	})
	
	// Add job to collect wealth distribution (daily)
	c.AddJob("WealthDistributionCollector", 24*time.Hour, func(ctx context.Context) error {
		return c.collectWealthDistribution(ctx)
	})
	
	// Add job to monitor and clean up completed webhooks (daily)
	c.AddJob("WebhookMaintenance", 24*time.Hour, func(ctx context.Context) error {
		return c.webhookMaintenance(ctx)
	})
}

// collectValidators collects validator information
func (c *CronJobManager) collectValidators(ctx context.Context) error {
	log.Printf("Collecting validators for network: %s", c.network)
	
	// Get validators
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
			"source":            "cronjob",
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
	
	return nil
}

// collectDelegators collects delegator information
func (c *CronJobManager) collectDelegators(ctx context.Context) error {
	log.Printf("Collecting delegators for network: %s", c.network)
	
	// Get delegators
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
			"source":              "cronjob",
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

// collectSubnets collects subnet information
func (c *CronJobManager) collectSubnets(ctx context.Context) error {
	log.Printf("Collecting subnets for network: %s", c.network)
	
	// Get subnets
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
			"source":      "cronjob",
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
	
	// Also collect blockchains
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
			"source":     "cronjob",
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

// collectPendingRewards collects pending staking rewards
func (c *CronJobManager) collectPendingRewards(ctx context.Context) error {
	log.Printf("Collecting pending rewards for network: %s", c.network)
	
	// Because this could be many addresses, we'll get a sample first
	// A real implementation would track all addresses with active stake
	// Here we'll just get the top validators and delegators
	
	// Get top validators first
	validatorsResp, err := c.dataAPI.GetValidators(c.network, nil, "", &api.PaginationParams{
		PageSize: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}
	
	// Extract addresses from validators
	addresses := make([]string, 0, len(validatorsResp.Validators))
	for _, validator := range validatorsResp.Validators {
		if validator.StakeOwnerAddress != "" {
			addresses = append(addresses, validator.StakeOwnerAddress)
		}
	}
	
	// Get pending rewards
	pendingRewardsResp, err := c.dataAPI.GetPendingRewards(c.network, addresses, &api.PaginationParams{
		PageSize: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to get pending rewards: %w", err)
	}
	
	log.Printf("Processing %d pending rewards", len(pendingRewardsResp.PendingRewards))
	
	// Publish pending rewards data to Kafka
	for _, reward := range pendingRewardsResp.PendingRewards {
		rewardData := map[string]interface{}{
			"nodeId":       reward.NodeID,
			"amount":       reward.Amount,
			"endTime":      reward.EndTime,
			"stakeOwner":   reward.StakeOwner,
			"rewardOwner":  reward.RewardOwner,
			"network":      c.network,
			"updateTime":   time.Now().Unix(),
			"source":       "cronjob",
		}
		
		rewardDataJSON, err := json.Marshal(rewardData)
		if err != nil {
			log.Printf("Error marshaling pending reward data: %v", err)
			continue
		}
		
		// Use reward owner as the key, or stake owner if reward owner is empty
		rewardOwner := reward.RewardOwner
		if rewardOwner == "" {
			rewardOwner = reward.StakeOwner
		}
		
		if err := c.kafkaProducer.PublishPendingReward(c.network, rewardOwner, rewardDataJSON); err != nil {
			log.Printf("Error publishing pending reward data to Kafka: %v", err)
			// Continue with the next reward
		}
	}
	
	return nil
}

// collectXChainAssets collects asset information from the X-Chain
func (c *CronJobManager) collectXChainAssets(ctx context.Context) error {
	log.Printf("Collecting X-Chain assets for network: %s", c.network)
	
	// Get blockchains to find X-Chain ID
	blockchainsResp, err := c.dataAPI.GetBlockchains(c.network, &api.PaginationParams{
		PageSize: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get blockchains: %w", err)
	}
	
	// Find X-Chain ID
	var xChainID string
	for _, blockchain := range blockchainsResp.Blockchains {
		if blockchain.Name == "X-Chain" {
			xChainID = blockchain.ID
			break
		}
	}
	
	if xChainID == "" {
		return fmt.Errorf("could not find X-Chain ID")
	}
	
	// We would need to get a list of asset IDs first
	// This would typically be done by tracking assets from transactions
	// For now, let's just get the native asset (AVAX)
	avaxAssetID := "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z" // AVAX asset ID on mainnet
	
	// Get asset details
	asset, err := c.dataAPI.GetAsset(c.network, xChainID, avaxAssetID)
	if err != nil {
		return fmt.Errorf("failed to get asset details: %w", err)
	}
	
	// Publish asset data to Kafka
	assetData := map[string]interface{}{
		"id":           asset.ID,
		"name":         asset.Name,
		"symbol":       asset.Symbol,
		"denomination": asset.Denomination,
		"initialStates": asset.InitialStates,
		"network":      c.network,
		"chainId":      xChainID,
		"updateTime":   time.Now().Unix(),
		"source":       "cronjob",
	}
	
	assetDataJSON, err := json.Marshal(assetData)
	if err != nil {
		return fmt.Errorf("error marshaling asset data: %w", err)
	}
	
	if err := c.kafkaProducer.PublishAsset(c.network, xChainID, asset.ID, assetDataJSON); err != nil {
		return fmt.Errorf("error publishing asset data to Kafka: %w", err)
	}
	
	return nil
}

// collectWealthDistribution collects wealth distribution metrics
func (c *CronJobManager) collectWealthDistribution(ctx context.Context) error {
	log.Printf("Collecting wealth distribution metrics")
	
	// Collect wealth distribution metrics for each chain
	// This would be similar to the MetricsPipeline.collectWealthDistributionMetrics
	// But we'll keep it simple here
	
	// Define thresholds for AVAX holdings to track
	balanceThresholds := []string{
		"0.1",  // At least 0.1 AVAX
		"1",    // At least 1 AVAX
		"10",   // At least 10 AVAX
		"100",  // At least 100 AVAX
		"1000", // At least 1000 AVAX
		"10000", // At least 10000 AVAX
	}
	
	// Get chains to monitor
	chainsResp, err := c.dataAPI.GetChains(c.network, &api.PaginationParams{
		PageSize: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get chains: %w", err)
	}
	
	// For each chain and threshold, get addresses with balance above threshold
	for _, chain := range chainsResp.Chains {
		for _, threshold := range balanceThresholds {
			// Fetch addresses by balance
			params := &api.MetricsParams{
				PageSize: 1,
			}
			
			resp, err := c.metricsAPI.GetAddressesByBalance(chain.ChainID, threshold, params)
			if err != nil {
				log.Printf("Error fetching addresses by balance for chain %s, threshold %s: %v", 
					chain.ChainID, threshold, err)
				continue
			}
			
			// Process the data point (should be just one)
			if len(resp.Results) > 0 {
				result := resp.Results[0]
				
				wealthData := map[string]interface{}{
					"chainId":        chain.ChainID,
					"balanceThreshold": threshold,
					"addressCount":   result.Count,
					"timestamp":      result.Timestamp,
					"collectTime":    time.Now().Unix(),
					"source":         "cronjob",
				}
				
				wealthDataJSON, err := json.Marshal(wealthData)
				if err != nil {
					log.Printf("Error marshaling wealth distribution data: %v", err)
					continue
				}
				
				if err := c.kafkaProducer.PublishWealthDistribution(chain.ChainID, threshold, wealthDataJSON); err != nil {
					log.Printf("Error publishing wealth distribution to Kafka: %v", err)
					continue
				}
			}
		}
	}
	
	return nil
}

// webhookMaintenance performs maintenance on webhooks
func (c *CronJobManager) webhookMaintenance(ctx context.Context) error {
	log.Printf("Performing webhook maintenance")
	
	// Since we don't have access to a webhook API directly from DataAPI,
	// we'll need to create a new client using the same base URL and API key
	// that the DataAPI is using
	
	// For this to work, you should add a GetBaseURL and GetAPIKey methods to your DataAPI struct
	// or add a client field to CronJobManager
	
	// Option 1: Create a new webhook API client using environment variables
	baseURL := os.Getenv("AVACLOUD_API_URL")
	apiKey := os.Getenv("AVACLOUD_API_KEY")
	client := api.NewClient(baseURL, apiKey)
	webhookAPI := api.NewWebhookAPI(client)
	
	// Alternative option (requires adding client to CronJobManager):
	// webhookAPI := api.NewWebhookAPI(c.client)
	
	webhooksResp, err := webhookAPI.GetWebhooks(&api.PaginationParams{
		PageSize: 100,
	})
	if err != nil {
		return fmt.Errorf("failed to get webhooks: %w", err)
	}
	
	log.Printf("Found %d webhooks", len(webhooksResp.Webhooks))
	
	// Check each webhook
	for _, webhook := range webhooksResp.Webhooks {
		// If webhook is inactive, remove it
		if !webhook.Active {
			log.Printf("Deleting inactive webhook: %s", webhook.ID)
			if err := webhookAPI.DeleteWebhook(webhook.ID); err != nil {
				log.Printf("Error deleting webhook: %v", err)
				continue
			}
		}
		
		// For active webhooks, ensure we have enough addresses
		// This would depend on your business logic
		if webhook.EventType == "address_activity" && len(webhook.Addresses) < 10 {
			log.Printf("Webhook %s has only %d addresses, adding more...", webhook.ID, len(webhook.Addresses))
			
			// In a real implementation, you'd have logic to determine which addresses to add
			// For now, we'll just log
			log.Printf("Would add more addresses to webhook %s", webhook.ID)
		}
	}
	
	return nil
}