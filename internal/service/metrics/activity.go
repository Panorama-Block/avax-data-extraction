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

// ActivityService collects user activity metrics
type ActivityService struct {
	*BaseService
}

// NewActivityService creates a new activity metrics service
func NewActivityService(
	api *api.API,
	eventManager *event.Manager,
	collectionInterval time.Duration,
	lookbackPeriod time.Duration,
	options ...service.ServiceOption,
) *ActivityService {
	baseService := NewBaseService(
		api,
		eventManager,
		"activity-metrics-service",
		collectionInterval,
		lookbackPeriod,
		options...,
	)
	
	return &ActivityService{
		BaseService: baseService,
	}
}

// Start starts the activity metrics service
func (s *ActivityService) Start() error {
	if err := s.BaseService.Start(); err != nil {
		return err
	}
	
	// Start metrics collection worker
	s.RunWorker(0, s.collectionWorker)
	
	return nil
}

// handleChainEvent handles chain events to add new chains for monitoring
func (s *ActivityService) handleChainEvent(evt types.Event) error {
	chainEvent, ok := evt.Data.(types.ChainEvent)
	if !ok {
		log.Printf("[%s] Invalid chain event data", s.GetName())
		return nil
	}
	
	// Add the chain for metrics collection
	s.AddChain(chainEvent.Chain.ChainID)
	
	return nil
}

// collectionWorker periodically collects activity metrics
func (s *ActivityService) collectionWorker(ctx context.Context, id int) {
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

// collectMetrics collects activity metrics for all chains
func (s *ActivityService) collectMetrics(ctx context.Context) {
	log.Printf("[%s] Collecting activity metrics", s.GetName())
	s.executeCollection()
}

// executeCollection collects activity metrics for all chains
func (s *ActivityService) executeCollection() error {
	if err := s.BaseService.executeCollection(); err != nil {
		return err
	}
	
	chains := s.GetChains()
	if len(chains) == 0 {
		log.Printf("[ActivityService] No chains configured for metrics collection")
		return nil
	}
	
	for _, chainID := range chains {
		metrics, err := s.collectChainActivityMetrics(chainID)
		if err != nil {
			log.Printf("[ActivityService] Error collecting metrics for chain %s: %v", chainID, err)
			continue
		}
		
		// Publish metrics to event manager
		s.publishMetrics(chainID, metrics)
	}
	
	return nil
}

// collectChainActivityMetrics collects activity metrics for a specific chain
func (s *ActivityService) collectChainActivityMetrics(chainID string) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.Add(-s.GetLookbackPeriod())
	
	log.Printf("[ActivityService] Collecting activity metrics for chain %s from %s to %s", 
		chainID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	
	// Sample metrics data - in real implementation, this would come from the API
	metrics := map[string]interface{}{
		"chainId":              chainID,
		"timestamp":            endTime.Unix(),
		"period":               s.GetLookbackPeriod().String(),
		"activeAddresses":      1250,
		"newAddressesCreated":  145,
		"transactionCount":     12500,
		"successfulTxCount":    12350,
		"failedTxCount":        150,
		"averageTxPerBlock":    85.5,
		"totalGasUsed":         "1250000000",
		"averageTxFee":         "0.00125",
	}
	
	return metrics, nil
}

// publishMetrics publishes activity metrics to the event manager
func (s *ActivityService) publishMetrics(chainID string, metrics map[string]interface{}) {
	event := types.Event{
		Type: types.EventActivityMetricsUpdated,
		Data: metrics,
	}
	
	if err := s.GetEventManager().PublishEvent(event); err != nil {
		log.Printf("[ActivityService] Error publishing activity metrics for chain %s: %v", chainID, err)
	} else {
		log.Printf("[ActivityService] Published activity metrics for chain %s", chainID)
	}
} 