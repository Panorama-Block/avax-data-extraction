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

// GasService collects gas-related metrics
type GasService struct {
	*BaseService
}

// NewGasService creates a new gas metrics service
func NewGasService(
	api *api.API,
	eventManager *event.Manager,
	collectionInterval time.Duration,
	lookbackPeriod time.Duration,
	options ...service.ServiceOption,
) *GasService {
	baseService := NewBaseService(
		api,
		eventManager,
		"gas-metrics-service",
		collectionInterval,
		lookbackPeriod,
		options...,
	)
	
	return &GasService{
		BaseService: baseService,
	}
}

// Start starts the gas metrics service
func (s *GasService) Start() error {
	if err := s.BaseService.Start(); err != nil {
		return err
	}
	
	// Start metrics collection worker
	s.RunWorker(0, s.collectionWorker)
	
	return nil
}

// handleChainEvent handles chain events to add new chains for monitoring
func (s *GasService) handleChainEvent(evt types.Event) error {
	chainEvent, ok := evt.Data.(types.ChainEvent)
	if !ok {
		log.Printf("[%s] Invalid chain event data", s.GetName())
		return nil
	}
	
	// Add the chain for metrics collection
	s.AddChain(chainEvent.Chain.ChainID)
	
	return nil
}

// collectionWorker periodically collects gas metrics
func (s *GasService) collectionWorker(ctx context.Context, id int) {
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

// collectMetrics collects gas metrics for all chains
func (s *GasService) collectMetrics(ctx context.Context) {
	log.Printf("[%s] Collecting gas metrics", s.GetName())
	s.executeCollection()
}

// executeCollection collects gas metrics for all chains
func (s *GasService) executeCollection() error {
	if err := s.BaseService.executeCollection(); err != nil {
		return err
	}
	
	chains := s.GetChains()
	if len(chains) == 0 {
		log.Printf("[GasService] No chains configured for metrics collection")
		return nil
	}
	
	for _, chainID := range chains {
		metrics, err := s.collectChainGasMetrics(chainID)
		if err != nil {
			log.Printf("[GasService] Error collecting metrics for chain %s: %v", chainID, err)
			continue
		}
		
		// Publish metrics to event manager
		s.publishMetrics(chainID, metrics)
	}
	
	return nil
}

// collectChainGasMetrics collects gas metrics for a specific chain
func (s *GasService) collectChainGasMetrics(chainID string) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.Add(-s.lookbackPeriod)
	
	log.Printf("[GasService] Collecting gas metrics for chain %s from %s to %s", 
		chainID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	
	// Sample metrics data - in real implementation, this would come from the API
	metrics := map[string]interface{}{
		"chainId":             chainID,
		"timestamp":           endTime.Unix(),
		"period":              s.lookbackPeriod.String(),
		"averageGasPrice":     "25000000000",  // 25 gwei
		"medianGasPrice":      "20000000000",  // 20 gwei
		"minimumGasPrice":     "15000000000",  // 15 gwei
		"maximumGasPrice":     "85000000000",  // 85 gwei
		"totalGasUsed":        "12500000000",
		"averageGasLimit":     "8000000",
		"averageGasUsed":      "5500000",
		"gasUtilizationRate":  68.75,  // percentage
	}
	
	return metrics, nil
}

// publishMetrics publishes gas metrics to the event manager
func (s *GasService) publishMetrics(chainID string, metrics map[string]interface{}) {
	event := types.Event{
		Type: types.EventGasMetricsUpdated,
		Data: metrics,
	}
	
	if err := s.GetEventManager().PublishEvent(event); err != nil {
		log.Printf("[GasService] Error publishing gas metrics for chain %s: %v", chainID, err)
	} else {
		log.Printf("[GasService] Published gas metrics for chain %s", chainID)
	}
} 