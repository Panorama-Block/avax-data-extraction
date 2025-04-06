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
	s.runMetricsCollection(ctx, s.collectChainCumulativeMetrics)
}

// collectChainCumulativeMetrics collects cumulative metrics for a specific chain
func (s *CumulativeService) collectChainCumulativeMetrics(ctx context.Context, chainID string, startTime, endTime time.Time) {
	log.Printf("[%s] Collecting cumulative metrics for chain %s", s.GetName(), chainID)
	
	// In a real implementation, this would query the API for:
	// 1. Total all-time transaction count
	// 2. Total unique addresses
	// 3. Total contracts deployed
	// 4. Total tokens created
	// 5. Total gas used
	// 6. Total fees collected
	
	// For this example, we'll create dummy metrics
	metrics := types.CumulativeMetrics{
		ChainID:             chainID,
		Timestamp:           time.Now(),
		TotalTxs:            10000000,       // Example value
		TotalAddresses:      500000,         // Example value
		TotalContracts:      5000,           // Example value
		TotalTokens:         1200,           // Example value
		TotalGasUsed:        5000000000000,  // Example value
		TotalFeesCollected:  25000.0,        // Example value
	}
	
	// Publish metrics event to cumulative metrics topic
	event := types.CumulativeMetricsEvent{
		Type:    types.EventCumulativeMetricsUpdated,
		Metrics: metrics,
	}
	
	err := s.PublishEvent(types.EventCumulativeMetricsUpdated, event)
	if err != nil {
		log.Printf("[%s] Error publishing cumulative metrics: %v", s.GetName(), err)
	}
} 