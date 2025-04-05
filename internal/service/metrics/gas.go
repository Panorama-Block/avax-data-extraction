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
	s.runMetricsCollection(ctx, s.collectChainGasMetrics)
}

// collectChainGasMetrics collects gas metrics for a specific chain
func (s *GasService) collectChainGasMetrics(ctx context.Context, chainID string, startTime, endTime time.Time) {
	log.Printf("[%s] Collecting gas metrics for chain %s", s.GetName(), chainID)
	
	// In a real implementation, this would query the API for:
	// 1. Total gas used in the period
	// 2. Average and max gas price
	// 3. Total fees paid in the period
	// 4. Average gas limit
	// 5. Gas efficiency (gasUsed/gasLimit ratio)
	
	// For this example, we'll create dummy metrics
	metrics := types.GasMetrics{
		ChainID:       chainID,
		Timestamp:     time.Now(),
		GasUsed:       5000000000,   // Example value
		AvgGasPrice:   50.0,         // Example value (gwei)
		MaxGasPrice:   200.0,        // Example value (gwei)
		FeesPaid:      25.5,         // Example value (native token)
		AvgGasLimit:   8000000,      // Example value
		GasEfficiency: 0.75,         // Example value (75% efficiency)
	}
	
	// Publish metrics event
	event := types.GasMetricsEvent{
		Type:    types.EventGasMetricsUpdated,
		Metrics: metrics,
	}
	
	err := s.PublishEvent(types.EventGasMetricsUpdated, event)
	if err != nil {
		log.Printf("[%s] Error publishing gas metrics: %v", s.GetName(), err)
	}
} 