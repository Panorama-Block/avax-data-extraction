package metrics

import (
	"context"
	"log"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/event"
	"github.com/Panorama-Block/avax/internal/service"
	"github.com/Panorama-Block/avax/internal/types"
)

// CumulativeService collects cumulative chain metrics
type CumulativeService struct {
	*BaseService
}

// NewCumulativeService creates a new cumulative metrics service
func NewCumulativeService(
	api *api.API,
	eventManager *event.Manager,
	collectionInterval time.Duration,
	options ...service.ServiceOption,
) *CumulativeService {
	// For cumulative metrics, we don't need a lookback period since 
	// we're looking at all-time metrics, but we'll set a small one
	// for consistency with other services
	lookbackPeriod := time.Hour
	
	baseService := NewBaseService(
		api,
		eventManager,
		"cumulative-metrics-service",
		collectionInterval,
		lookbackPeriod,
		options...,
	)
	
	return &CumulativeService{
		BaseService: baseService,
	}
}

// Start starts the cumulative metrics service
func (s *CumulativeService) Start() error {
	if err := s.BaseService.Start(); err != nil {
		return err
	}
	
	// Start metrics collection worker
	s.RunWorker(0, s.collectionWorker)
	
	return nil
}

// handleChainEvent handles chain events to add new chains for monitoring
func (s *CumulativeService) handleChainEvent(evt types.Event) error {
	chainEvent, ok := evt.Data.(types.ChainEvent)
	if !ok {
		log.Printf("[%s] Invalid chain event data", s.GetName())
		return nil
	}
	
	// Add the chain for metrics collection
	s.AddChain(chainEvent.Chain.ChainID)
	
	return nil
}

// collectionWorker periodically collects cumulative metrics
func (s *CumulativeService) collectionWorker(ctx context.Context, id int) {
	ticker := time.NewTicker(s.GetCollectionInterval())
	defer ticker.Stop()
	
	// Run immediately on start
	s.collectMetrics(ctx)
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Worker %d stopping", s.GetName(), id)
			return
		case <-ticker.C:
			s.collectMetrics(ctx)
		}
	}
}

// collectMetrics collects cumulative metrics for all chains
func (s *CumulativeService) collectMetrics(ctx context.Context) {
	log.Printf("[%s] Collecting cumulative metrics", s.GetName())
	s.executeCollection()
}

// executeCollection collects cumulative metrics for all chains
func (s *CumulativeService) executeCollection() error {
	if err := s.BaseService.executeCollection(); err != nil {
		return err
	}
	
	chains := s.GetChains()
	if len(chains) == 0 {
		log.Printf("[CumulativeService] No chains configured for metrics collection")
		return nil
	}
	
	for _, chainID := range chains {
		metrics, err := s.collectChainCumulativeMetrics(chainID)
		if err != nil {
			log.Printf("[CumulativeService] Error collecting metrics for chain %s: %v", chainID, err)
			continue
		}
		
		// Publish metrics to event manager
		s.publishMetrics(chainID, metrics)
	}
	
	return nil
}

// collectChainCumulativeMetrics collects cumulative metrics for a specific chain
func (s *CumulativeService) collectChainCumulativeMetrics(chainID string) (map[string]interface{}, error) {
	log.Printf("[CumulativeService] Collecting cumulative metrics for chain %s", chainID)
	
	// Sample metrics data - in real implementation, this would come from the API
	metrics := map[string]interface{}{
		"chainId":                     chainID,
		"timestamp":                   time.Now().Unix(),
		"totalTransactions":           45689720,
		"totalAccounts":               2356987,
		"totalActiveAccounts":         985642,
		"totalContractsDeployed":      125678,
		"activeContractsLastMonth":    45620,
		"totalGasUsed":                "1254789654123000",
		"totalBlocksProduced":         15784523,
		"averageBlockSize":            350750,
		"totalValueTransferred":       "125478965412300000000000", // in wei
		"totalValueTransferredUSD":    "245789654123.50",
	}
	
	return metrics, nil
}

// publishMetrics publishes cumulative metrics to the event manager
func (s *CumulativeService) publishMetrics(chainID string, metrics map[string]interface{}) {
	event := types.Event{
		Type: types.EventCumulativeMetricsUpdated,
		Data: metrics,
	}
	
	if err := s.GetEventManager().PublishEvent(event); err != nil {
		log.Printf("[CumulativeService] Error publishing cumulative metrics for chain %s: %v", chainID, err)
	} else {
		log.Printf("[CumulativeService] Published cumulative metrics for chain %s", chainID)
	}
} 