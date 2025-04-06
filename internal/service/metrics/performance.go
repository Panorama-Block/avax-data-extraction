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
	s.runMetricsCollection(ctx, s.collectChainPerformanceMetrics)
}

// collectChainPerformanceMetrics collects performance metrics for a specific chain
func (s *PerformanceService) collectChainPerformanceMetrics(ctx context.Context, chainID string, startTime, endTime time.Time) {
	log.Printf("[%s] Collecting performance metrics for chain %s", s.GetName(), chainID)
	
	// In a real implementation, this would query the API for:
	// 1. Average and max TPS (transactions per second)
	// 2. Average and max GPS (gas per second)
	// 3. Average block time
	// 4. Average transaction latency

	// For example, to calculate TPS:
	// 1. Get total transaction count in the period
	// 2. Divide by the duration of the period in seconds
	
	// For this example, we'll create dummy metrics
	metrics := types.PerformanceMetrics{
		ChainID:    chainID,
		Timestamp:  time.Now(),
		AvgTPS:     25.5,  // Example value
		MaxTPS:     100.0, // Example value
		AvgGPS:     450000.0, // Example value (gas per second)
		MaxGPS:     1200000.0, // Example value
		BlockTime:  2.0,   // Example value (seconds)
		AvgLatency: 5.0,   // Example value (seconds)
	}
	
	// Publish metrics event to performance metrics topic
	event := types.PerformanceMetricsEvent{
		Type:    types.EventPerformanceMetricsUpdated,
		Metrics: metrics,
	}
	
	err := s.PublishEvent(types.EventPerformanceMetricsUpdated, event)
	if err != nil {
		log.Printf("[%s] Error publishing performance metrics: %v", s.GetName(), err)
	}
} 