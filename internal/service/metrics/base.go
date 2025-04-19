package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/event"
	"github.com/Panorama-Block/avax/internal/service"
	"github.com/Panorama-Block/avax/internal/types"
)

// BaseService provides common functionality for metrics services
type BaseService struct {
	*service.Base
	lastCollectionTime time.Time
	lastCollectionMutex sync.Mutex
	chains              []string
	chainMutex          sync.RWMutex
	collectionInterval  time.Duration
	lookbackPeriod      time.Duration
	context             context.Context
}

// NewBaseService creates a new base metrics service
func NewBaseService(
	api *api.API,
	eventManager *event.Manager,
	name string,
	collectionInterval time.Duration,
	lookbackPeriod time.Duration,
	options ...service.ServiceOption,
) *BaseService {
	baseService := service.NewBase(api, eventManager, name, options...)
	
	return &BaseService{
		Base:              baseService,
		chains:            []string{},
		collectionInterval: collectionInterval,
		lookbackPeriod:    lookbackPeriod,
		context:           context.Background(),
	}
}

// SetChains sets the chains to collect metrics for
func (s *BaseService) SetChains(chains []string) {
	s.chainMutex.Lock()
	defer s.chainMutex.Unlock()
	s.chains = chains
}

// AddChain adds a chain to collect metrics for
func (s *BaseService) AddChain(chainID string) {
	s.chainMutex.Lock()
	defer s.chainMutex.Unlock()

	// Check if chain already exists
	for _, id := range s.chains {
		if id == chainID {
			return
		}
	}
	
	s.chains = append(s.chains, chainID)
}

// GetChains gets the chains being monitored
func (s *BaseService) GetChains() []string {
	s.chainMutex.RLock()
	defer s.chainMutex.RUnlock()
	result := make([]string, len(s.chains))
	copy(result, s.chains)
	return result
}

// SetLastCollectionTime sets the last metrics collection time
func (s *BaseService) SetLastCollectionTime(t time.Time) {
	s.lastCollectionMutex.Lock()
	defer s.lastCollectionMutex.Unlock()
	s.lastCollectionTime = t
}

// GetLastCollectionTime gets the last metrics collection time
func (s *BaseService) GetLastCollectionTime() time.Time {
	s.lastCollectionMutex.Lock()
	defer s.lastCollectionMutex.Unlock()
	return s.lastCollectionTime
}

// GetCollectionInterval returns the collection interval
func (s *BaseService) GetCollectionInterval() time.Duration {
	return s.collectionInterval
}

// GetLookbackPeriod returns the lookback period for metric calculation
func (s *BaseService) GetLookbackPeriod() time.Duration {
	return s.lookbackPeriod
}

// GetTimeRange returns start and end times for metrics collection
func (s *BaseService) GetTimeRange() (time.Time, time.Time) {
	endTime := time.Now()
	startTime := endTime.Add(-s.lookbackPeriod)
	return startTime, endTime
}

// Start starts the service
func (s *BaseService) Start() error {
	if err := s.Base.Start(); err != nil {
		return err
	}
	
	// Log the metrics topics being used
	log.Printf("[%s] Started with metrics collection interval: %s", s.GetName(), s.collectionInterval)
	
	// Execute once immediately
	if err := s.executeCollection(); err != nil {
		log.Printf("[%s] Initial metrics collection failed: %v", s.GetName(), err)
	}
	
	// Schedule periodic collection
	ticker := time.NewTicker(s.collectionInterval)
	go func() {
		for {
			select {
			case <-s.context.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := s.executeCollection(); err != nil {
					log.Printf("[%s] Metrics collection failed: %v", s.GetName(), err)
				}
			}
		}
	}()
	
	return nil
}

// executeCollection is a helper method to run the collection process
func (s *BaseService) executeCollection() error {
	s.SetLastCollectionTime(time.Now())
	log.Printf("[%s] Collecting metrics for chains: %v", s.GetName(), s.GetChains())
	return nil
}

// handleChainEvent handles chain events
func (s *BaseService) handleChainEvent(event types.Event) error {
	// This method would be implemented by the specific metrics service
	return nil
}

// runMetricsCollection runs a metrics collection cycle for all chains
func (s *BaseService) runMetricsCollection(ctx context.Context, workerFunc func(ctx context.Context, chainID string, startTime, endTime time.Time)) {
	startTime, endTime := s.GetTimeRange()
	s.SetLastCollectionTime(time.Now())
	
	chains := s.GetChains()
	if len(chains) == 0 {
		log.Printf("[%s] No chains configured for metrics collection", s.GetName())
		return
	}
	
	// Create a wait group to wait for all chains to be processed
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.GetWorkerCount())
	
	for _, chainID := range chains {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(cid string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			workerFunc(ctx, cid, startTime, endTime)
		}(chainID)
	}
	
	wg.Wait()
} 