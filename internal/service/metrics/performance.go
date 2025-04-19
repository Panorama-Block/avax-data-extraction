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

// PerformanceService collects chain performance metrics
type PerformanceService struct {
	*BaseService
}

// NewPerformanceService creates a new performance metrics service
func NewPerformanceService(
	api *api.API,
	eventManager *event.Manager,
	collectionInterval time.Duration,
	lookbackPeriod time.Duration,
	options ...service.ServiceOption,
) *PerformanceService {
	baseService := NewBaseService(
		api,
		eventManager,
		"performance-metrics-service",
		collectionInterval,
		lookbackPeriod,
		options...,
	)
	
	return &PerformanceService{
		BaseService: baseService,
	}
}

// Start starts the performance metrics service
func (s *PerformanceService) Start() error {
	if err := s.BaseService.Start(); err != nil {
		return err
	}
	
	// Start metrics collection worker
	s.RunWorker(0, s.collectionWorker)
	
	return nil
}

// handleChainEvent handles chain events to add new chains for monitoring
func (s *PerformanceService) handleChainEvent(evt types.Event) error {
	chainEvent, ok := evt.Data.(types.ChainEvent)
	if !ok {
		log.Printf("[%s] Invalid chain event data", s.GetName())
		return nil
	}
	
	// Add the chain for metrics collection
	s.AddChain(chainEvent.Chain.ChainID)
	
	return nil
}

// collectionWorker periodically collects performance metrics
func (s *PerformanceService) collectionWorker(ctx context.Context, id int) {
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

// collectMetrics collects performance metrics for all chains
func (s *PerformanceService) collectMetrics(ctx context.Context) {
	log.Printf("[%s] Collecting performance metrics", s.GetName())
	s.executeCollection()
}

// executeCollection collects performance metrics for all chains
func (s *PerformanceService) executeCollection() error {
	if err := s.BaseService.executeCollection(); err != nil {
		return err
	}
	
	chains := s.GetChains()
	if len(chains) == 0 {
		log.Printf("[PerformanceService] No chains configured for metrics collection")
		return nil
	}
	
	for _, chainID := range chains {
		metrics, err := s.collectChainPerformanceMetrics(chainID)
		if err != nil {
			log.Printf("[PerformanceService] Error collecting metrics for chain %s: %v", chainID, err)
			continue
		}
		
		// Publish metrics to event manager
		s.publishMetrics(chainID, metrics)
	}
	
	return nil
}

// collectChainPerformanceMetrics collects performance metrics for a specific chain
func (s *PerformanceService) collectChainPerformanceMetrics(chainID string) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.Add(-s.lookbackPeriod)
	
	log.Printf("[PerformanceService] Collecting performance metrics for chain %s from %s to %s", 
		chainID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	
	// Sample metrics data - in real implementation, this would come from the API
	metrics := map[string]interface{}{
		"chainId":               chainID,
		"timestamp":             endTime.Unix(),
		"period":                s.lookbackPeriod.String(),
		"averageTps":            85.2,
		"peakTps":               350.5,
		"averageBlockTime":      2.15,
		"currentBlockTime":      2.05,
		"averageBlockSize":      245678,
		"currentMemPoolSize":    157,
		"averageConfirmationTime": 12.5,
	}
	
	return metrics, nil
}

// publishMetrics publishes performance metrics to the event manager
func (s *PerformanceService) publishMetrics(chainID string, metrics map[string]interface{}) {
	event := types.Event{
		Type: types.EventPerformanceMetricsUpdated,
		Data: metrics,
	}
	
	if err := s.GetEventManager().PublishEvent(event); err != nil {
		log.Printf("[PerformanceService] Error publishing performance metrics for chain %s: %v", chainID, err)
	} else {
		log.Printf("[PerformanceService] Published performance metrics for chain %s", chainID)
	}
} 