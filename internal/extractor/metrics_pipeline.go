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

// MetricsPipeline handles collection and processing of metrics
type MetricsPipeline struct {
	metricsAPI     *api.MetricsAPI
	kafkaProducer  *kafka.Producer
	
	// Configuration
	network        string           // "mainnet", "fuji", etc.
	chains         []string         // List of chainIDs to monitor
	interval       time.Duration    // How often to fetch metrics
	
	// Control
	stop           chan struct{}
	mutex          sync.Mutex
	running        bool
	
	// State
	lastRunTime    time.Time
}

// NewMetricsPipeline creates a new metrics pipeline
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

// SetChains sets the chains to monitor
func (m *MetricsPipeline) SetChains(chains []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.chains = chains
}

// Start begins the metrics collection process
func (m *MetricsPipeline) Start(ctx context.Context) error {
	m.mutex.Lock()
	if m.running {
		m.mutex.Unlock()
		return fmt.Errorf("metrics pipeline already running")
	}
	m.running = true
	m.lastRunTime = time.Time{}
	m.mutex.Unlock()
	
	log.Printf("Starting metrics pipeline for network: %s", m.network)
	
	// Run initial collection
	if err := m.collectMetrics(ctx); err != nil {
		log.Printf("Warning: Initial metrics collection failed: %v", err)
		// Continue anyway as this is just the initial run
	}
	
	// Start the periodic collection
	go m.runPeriodic(ctx)
	
	return nil
}

// Status returns the current status of the pipeline
func (m *MetricsPipeline) Status() map[string]interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	lastRunStr := "Never"
	if !m.lastRunTime.IsZero() {
		lastRunStr = m.lastRunTime.Format(time.RFC3339)
	}
	
	nextRunStr := "Not scheduled"
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

// Stop halts the metrics collection process
func (m *MetricsPipeline) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return
	}
	
	close(m.stop)
	m.running = false
	log.Printf("Stopping metrics pipeline")
}

// runPeriodic runs the metrics collection process periodically
func (m *MetricsPipeline) runPeriodic(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			if err := m.collectMetrics(ctx); err != nil {
				log.Printf("Error collecting metrics: %v", err)
				// Continue with the next interval
			}
		}
	}
}

// collectMetrics collects and processes metrics
func (m *MetricsPipeline) collectMetrics(ctx context.Context) error {
	start := time.Now()
	log.Printf("Starting metrics collection")
	
	// Update lastRunTime
	m.mutex.Lock()
	m.lastRunTime = start
	m.mutex.Unlock()
	
	// Collect chain metrics
	if len(m.chains) > 0 {
		if err := m.collectChainMetrics(ctx); err != nil {
			log.Printf("Error collecting chain metrics: %v", err)
			// Continue with other metrics
		}
	} else {
		log.Printf("No chains configured for metrics collection")
	}
	
	return nil
}

// collectStakingMetrics collects metrics related to staking
func (m *MetricsPipeline) collectStakingMetrics(ctx context.Context) error {
	log.Printf("Collecting staking metrics for network: %s", m.network)
	
	// Define the time range for metrics
	endTime := time.Now().Unix()
	startTime := endTime - 86400 // Last 24 hours
	
	// List of staking metrics to collect
	metrics := []api.StakingMetric{
		api.ValidatorCount,
		api.DelegatorCount,
		api.ValidatorWeight,
		api.DelegatorWeight,
	}
	
	// Collect metrics for the primary network
	for _, metric := range metrics {
		metricsResp, err := m.metricsAPI.GetStakingMetrics(m.network, metric, "", &api.MetricsParams{
			StartTimestamp: startTime,
			EndTimestamp:   endTime,
			TimeInterval:   api.TimeIntervalHour,
			PageSize:       24, // 24 data points (hourly for a day)
		})
		if err != nil {
			log.Printf("Error collecting staking metric %s: %v", metric, err)
			continue
		}
		
		// Process and publish each data point
		for _, dataPoint := range metricsResp.Results {
			metricData := map[string]interface{}{
				"network":    m.network,
				"metric":     string(metric),
				"timestamp":  dataPoint.Timestamp,
				"value":      dataPoint.Value,
				"interval":   string(api.TimeIntervalHour),
				"collectTime": time.Now().Unix(),
			}
			
			metricJSON, err := json.Marshal(metricData)
			if err != nil {
				log.Printf("Error marshaling staking metric data: %v", err)
				continue
			}
			
			if err := m.kafkaProducer.PublishStakingMetric(m.network, string(metric), metricJSON); err != nil {
				log.Printf("Error publishing staking metric to Kafka: %v", err)
				continue
			}
		}
	}
	
	// Collect also for specific subnets if needed
	// This would be similar to the above but with a subnet ID specified
	
	return nil
}

// collectTeleporterMetrics collects metrics related to Teleporter (BTC bridge)
func (m *MetricsPipeline) collectTeleporterMetrics(ctx context.Context) error {
	log.Printf("Collecting Teleporter metrics")
	
	// Define the time range for metrics
	endTime := time.Now().Unix()
	startTime := endTime - 86400 // Last 24 hours
	
	// List of teleporter metrics to collect
	metrics := []string{
		"bridgeVolume",
		"bridgeTransactions",
		"avgBridgeTime",
		"activeBridgeUsers",
	}
	
	// Collect each metric
	for _, metric := range metrics {
		metricsResp, err := m.metricsAPI.GetTeleporterMetrics(metric, &api.MetricsParams{
			StartTimestamp: startTime,
			EndTimestamp:   endTime,
			TimeInterval:   api.TimeIntervalHour,
			PageSize:       24, // 24 data points (hourly for a day)
		})
		if err != nil {
			log.Printf("Error collecting teleporter metric %s: %v", metric, err)
			continue
		}
		
		// Process and publish each data point
		for _, dataPoint := range metricsResp.Results {
			metricData := map[string]interface{}{
				"metric":     metric,
				"timestamp":  dataPoint.Timestamp,
				"value":      dataPoint.Value,
				"interval":   string(api.TimeIntervalHour),
				"collectTime": time.Now().Unix(),
			}
			
			metricJSON, err := json.Marshal(metricData)
			if err != nil {
				log.Printf("Error marshaling teleporter metric data: %v", err)
				continue
			}
			
			if err := m.kafkaProducer.PublishTeleporterMetric(metric, metricJSON); err != nil {
				log.Printf("Error publishing teleporter metric to Kafka: %v", err)
				continue
			}
		}
	}
	
	return nil
}

// collectWealthDistributionMetrics collects wealth distribution metrics
func (m *MetricsPipeline) collectWealthDistributionMetrics(ctx context.Context) error {
	log.Printf("Collecting wealth distribution metrics")
	
	// Define balance thresholds to track
	balanceThresholds := []string{
		"0.1",  // At least 0.1 AVAX
		"1",    // At least 1 AVAX
		"10",   // At least 10 AVAX
		"100",  // At least 100 AVAX
		"1000", // At least 1000 AVAX
		"10000", // At least 10000 AVAX
	}
	
	// Process each chain
	for _, chainID := range m.chains {
		for _, threshold := range balanceThresholds {
			// Fetch addresses by balance
			params := &api.MetricsParams{
				PageSize: 1,
			}
			
			resp, err := m.metricsAPI.GetAddressesByBalance(chainID, threshold, params)
			if err != nil {
				log.Printf("Error fetching addresses by balance for chain %s, threshold %s: %v", 
					chainID, threshold, err)
				continue
			}
			
			// Process the data point (should be just one)
			if len(resp.Results) > 0 {
				result := resp.Results[0]
				
				wealthData := map[string]interface{}{
					"chainId":        chainID,
					"balanceThreshold": threshold,
					"addressCount":   result.Count,
					"timestamp":      result.Timestamp,
					"collectTime":    time.Now().Unix(),
				}
				
				wealthDataJSON, err := json.Marshal(wealthData)
				if err != nil {
					log.Printf("Error marshaling wealth distribution data: %v", err)
					continue
				}
				
				if err := m.kafkaProducer.PublishWealthDistribution(chainID, threshold, wealthDataJSON); err != nil {
					log.Printf("Error publishing wealth distribution to Kafka: %v", err)
					continue
				}
			}
		}
	}
	
	// Also collect BTC.b holders if needed
	btcBalanceThresholds := []string{
		"0.1",  // At least 0.1 BTC.b
		"1",    // At least 1 BTC.b
		"10",   // At least 10 BTC.b
	}
	
	for _, threshold := range btcBalanceThresholds {
		params := &api.MetricsParams{
			PageSize: 100, // Get top 100 holders
		}
		
		resp, err := m.metricsAPI.GetAddressesByBtcBalance(threshold, params)
		if err != nil {
			log.Printf("Error fetching BTC.b holders with threshold %s: %v", threshold, err)
			continue
		}
		
		// Process each BTC.b holder
		holderData := map[string]interface{}{
			"threshold":   threshold,
			"holderCount": len(resp.Results),
			"holders":     resp.Results,
			"collectTime": time.Now().Unix(),
		}
		
		holderDataJSON, err := json.Marshal(holderData)
		if err != nil {
			log.Printf("Error marshaling BTC.b holders data: %v", err)
			continue
		}
		
		if err := m.kafkaProducer.PublishBtcHolders(threshold, holderDataJSON); err != nil {
			log.Printf("Error publishing BTC.b holders to Kafka: %v", err)
			continue
		}
	}
	
	return nil
}


// collectChainMetrics collects metrics for each chain
func (m *MetricsPipeline) collectChainMetrics(ctx context.Context) error {
	// Define the time range for metrics
	endTime := time.Now().Unix()
	startTime := endTime - 86400 // Last 24 hours
	
	// Metrics to collect for each chain
	metrics := []api.EVMChainMetric{
		api.ActiveAddresses,
		api.ActiveSenders,
		api.TxCount,
		api.GasUsed,
		api.AvgTps,
		api.MaxTps,
		api.AvgGasPrice,
		api.MaxGasPrice,
		api.FeesPaid,
	}
	
	// Also collect some cumulative metrics (one data point)
	cumulativeMetrics := []api.EVMChainMetric{
		api.CumulativeAddresses,
		api.CumulativeTxCount,
		api.CumulativeContracts,
		api.CumulativeDeployers,
	}
	
	// Process each chain
	for _, chainID := range m.chains {
		// Collect standard metrics with hourly granularity
		for _, metric := range metrics {
			if err := m.collectChainMetric(ctx, chainID, metric, startTime, endTime, api.TimeIntervalHour); err != nil {
				log.Printf("Error collecting chain metric %s for chain %s: %v", metric, chainID, err)
				continue
			}
		}
		
		// Collect cumulative metrics (these only need daily granularity)
		for _, metric := range cumulativeMetrics {
			if err := m.collectChainMetric(ctx, chainID, metric, startTime, endTime, api.TimeIntervalDay); err != nil {
				log.Printf("Error collecting cumulative metric %s for chain %s: %v", metric, chainID, err)
				continue
			}
		}
	}
	
	return nil
}

// collectChainMetric collects a specific metric for a chain
func (m *MetricsPipeline) collectChainMetric(
	ctx context.Context,
	chainID string,
	metric api.EVMChainMetric,
	startTime, endTime int64,
	interval api.TimeInterval,
) error {
	// Fetch the metric data
	metricsResp, err := m.metricsAPI.GetEVMChainMetrics(chainID, metric, &api.MetricsParams{
		StartTimestamp: startTime,
		EndTimestamp:   endTime,
		TimeInterval:   interval,
		PageSize:       100, // Reasonable page size
	})
	if err != nil {
		return fmt.Errorf("failed to fetch metric %s for chain %s: %w", metric, chainID, err)
	}
	
	// Process each data point
	for _, dataPoint := range metricsResp.Results {
		// Prepare Kafka message
		metricData := map[string]interface{}{
			"chainId":    chainID,
			"metric":     string(metric),
			"timestamp":  dataPoint.Timestamp,
			"value":      dataPoint.Value,
			"interval":   string(interval),
			"collectTime": time.Now().Unix(),
		}
		
		// Convert to JSON
		metricJSON, err := json.Marshal(metricData)
		if err != nil {
			log.Printf("Error marshaling metric data: %v", err)
			continue
		}
		
		// Publish to Kafka
		if err := m.kafkaProducer.PublishMetric(chainID, string(metric), metricJSON); err != nil {
			log.Printf("Error publishing metric to Kafka: %v", err)
			continue
		}
	}
	
	// Handle pagination if necessary
	for metricsResp.NextPageToken != "" {
		metricsResp, err = m.metricsAPI.GetEVMChainMetrics(chainID, metric, &api.MetricsParams{
			StartTimestamp: startTime,
			EndTimestamp:   endTime,
			TimeInterval:   interval,
			PageSize:       100,
			PageToken:      metricsResp.NextPageToken,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch next page of metric %s for chain %s: %w", metric, chainID, err)
		}
		
		// Process each data point in the new page
		for _, dataPoint := range metricsResp.Results {
			metricData := map[string]interface{}{
				"chainId":    chainID,
				"metric":     string(metric),
				"timestamp":  dataPoint.Timestamp,
				"value":      dataPoint.Value,
				"interval":   string(interval),
				"collectTime": time.Now().Unix(),
			}
			
			metricJSON, err := json.Marshal(metricData)
			if err != nil {
				log.Printf("Error marshaling metric data: %v", err)
				continue
			}
			
			if err := m.kafkaProducer.PublishMetric(chainID, string(metric), metricJSON); err != nil {
				log.Printf("Error publishing metric to Kafka: %v", err)
				continue
			}
		}
	}
	return nil
}