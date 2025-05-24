package chain

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

// Service fetches and processes chains
type Service struct {
	*service.Base
	lastSyncTime    time.Time
	lastSyncMutex   sync.Mutex
	processedChains map[string]time.Time
}

// NewService creates a new chain service
func NewService(api *api.API, eventManager *event.Manager, options ...service.ServiceOption) *Service {
	baseService := service.NewBase(api, eventManager, "chain-service", options...)

	return &Service{
		Base:            baseService,
		processedChains: make(map[string]time.Time),
	}
}

// Start starts the chain service
func (s *Service) Start() error {
	if err := s.Base.Start(); err != nil {
		return err
	}

	// Start chain sync worker
	s.RunWorker(0, s.syncWorker)

	log.Printf("[%s] Started successfully", s.GetName())
	return nil
}

// SetLastSyncTime sets the last sync time
func (s *Service) SetLastSyncTime(t time.Time) {
	s.lastSyncMutex.Lock()
	defer s.lastSyncMutex.Unlock()
	s.lastSyncTime = t
}

// GetLastSyncTime gets the last sync time
func (s *Service) GetLastSyncTime() time.Time {
	s.lastSyncMutex.Lock()
	defer s.lastSyncMutex.Unlock()
	return s.lastSyncTime
}

func (s *Service) syncWorker(ctx context.Context, id int) {
	ticker := time.NewTicker(s.GetPollInterval())
	defer ticker.Stop()

	// Run immediately on start
	s.syncChains()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Worker %d stopping", s.GetName(), id)
			return
		case <-ticker.C:
			s.syncChains()
		}
	}
}

func (s *Service) syncChains() {
	log.Printf("[%s] Syncing chains", s.GetName())
	s.SetLastSyncTime(time.Now())

	chains, err := s.GetAPI().Chains.GetChains()
	if err != nil {
		log.Printf("[%s] Error fetching chains: %v", s.GetName(), err)
		return
	}

	for _, chain := range chains {
		s.processChain(chain)
	}
}

func (s *Service) processChain(chain types.Chain) {
	// Skip chains we've processed recently
	lastProcessed, exists := s.processedChains[chain.ChainID]
	if exists && time.Since(lastProcessed) < s.GetPollInterval() {
		return
	}

	// Publish event for chain
	err := s.PublishEvent(types.EventChainUpdated, types.ChainEvent{
		Chain: chain,
	})

	if err != nil {
		log.Printf("[%s] Error publishing chain event: %v", s.GetName(), err)
		return
	}

	s.processedChains[chain.ChainID] = time.Now()
}
