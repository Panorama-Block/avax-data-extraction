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
	s.runMetricsCollection(ctx, s.collectChainActivityMetrics)
}

// collectChainActivityMetrics collects activity metrics for a specific chain
func (s *ActivityService) collectChainActivityMetrics(ctx context.Context, chainID string, startTime, endTime time.Time) {
	log.Printf("[%s] Collecting activity metrics for chain %s", s.GetName(), chainID)
	
	// In a real implementation, this would query the API for:
	// 1. Number of active addresses in the period
	// 2. Number of active senders in the period
	// 3. Transaction count in the period
	// 4. New addresses created in the period
	// 5. Unique contracts interacted with in the period
	
	// For example (pseudocode):
	// activeAddresses, err := s.GetAPI().GetActiveAddresses(chainID, startTime, endTime)
	// if err != nil { log.Printf("Error: %v", err); return }
	
	// For this example, we'll create dummy metrics
	metrics := types.ActivityMetrics{
		ChainID:         chainID,
		Timestamp:       time.Now(),
		ActiveAddresses: 1000, // Example value
		ActiveSenders:   500,  // Example value
		TxCount:         5000, // Example value
		NewAddresses:    200,  // Example value
		UniqueContracts: 50,   // Example value
	}
	
	// Publish metrics event
	event := types.ActivityMetricsEvent{
		Type:    types.EventActivityMetricsUpdated,
		Metrics: metrics,
	}
	
	err := s.PublishEvent(types.EventActivityMetricsUpdated, event)
	if err != nil {
		log.Printf("[%s] Error publishing activity metrics: %v", s.GetName(), err)
	}
} 