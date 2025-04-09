package block

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

// Service fetches and processes blocks
type Service struct {
	*service.Base
	lastSyncTime     time.Time
	lastSyncMutex    sync.Mutex
	chains           []string
	chainMutex       sync.RWMutex
	processedBlocks  map[string]map[string]time.Time // chainID -> blockNumber -> time
	blocksMutex      sync.RWMutex
	pageSize         int
}

// NewService creates a new block service
func NewService(api *api.API, eventManager *event.Manager, options ...service.ServiceOption) *Service {
	baseService := service.NewBase(api, eventManager, "block-service", options...)
	
	return &Service{
		Base:            baseService,
		chains:          []string{},
		processedBlocks: make(map[string]map[string]time.Time),
		pageSize:        100,
	}
}

// Start starts the block service
func (s *Service) Start() error {
	if err := s.Base.Start(); err != nil {
		return err
	}
	
	// Subscribe to chain events
	s.GetEventManager().Subscribe(types.EventChainCreated, s.handleChainEvent)
	s.GetEventManager().Subscribe(types.EventChainUpdated, s.handleChainEvent)
	
	// Start block sync workers
	for i := 0; i < s.GetWorkerCount(); i++ {
		s.RunWorker(i, s.syncWorker)
	}
	
	log.Printf("[%s] Started successfully", s.GetName())
	return nil
}

// SetChains sets the chains to monitor
func (s *Service) SetChains(chains []string) {
	s.chainMutex.Lock()
	defer s.chainMutex.Unlock()
	s.chains = chains
	
	// Initialize processed blocks map for each chain
	s.blocksMutex.Lock()
	defer s.blocksMutex.Unlock()
	for _, chainID := range chains {
		if _, exists := s.processedBlocks[chainID]; !exists {
			s.processedBlocks[chainID] = make(map[string]time.Time)
		}
	}
}

// GetChains gets the chains being monitored
func (s *Service) GetChains() []string {
	s.chainMutex.RLock()
	defer s.chainMutex.RUnlock()
	result := make([]string, len(s.chains))
	copy(result, s.chains)
	return result
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

func (s *Service) handleChainEvent(event types.Event) error {
	chainEvent, ok := event.Data.(types.ChainEvent)
	if !ok {
		log.Printf("[%s] Invalid chain event data", s.GetName())
		return nil
	}
	
	chainID := chainEvent.Chain.ChainID
	
	// Add chain to our list if it's not already there
	s.chainMutex.Lock()
	found := false
	for _, id := range s.chains {
		if id == chainID {
			found = true
			break
		}
	}
	
	if !found {
		s.chains = append(s.chains, chainID)
		
		// Initialize processed blocks map for this chain
		s.blocksMutex.Lock()
		if _, exists := s.processedBlocks[chainID]; !exists {
			s.processedBlocks[chainID] = make(map[string]time.Time)
		}
		s.blocksMutex.Unlock()
	}
	s.chainMutex.Unlock()
	
	return nil
}

func (s *Service) syncWorker(ctx context.Context, id int) {
	ticker := time.NewTicker(s.GetPollInterval())
	defer ticker.Stop()
	
	// Run immediately on start
	s.syncBlocks(id)
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Worker %d stopping", s.GetName(), id)
			return
		case <-ticker.C:
			s.syncBlocks(id)
		}
	}
}

func (s *Service) syncBlocks(workerID int) {
	s.SetLastSyncTime(time.Now())
	
	chains := s.GetChains()
	if len(chains) == 0 {
		return
	}
	
	// Simple worker distribution - each worker takes a subset of chains
	for i, chainID := range chains {
		if i % s.GetWorkerCount() == workerID {
			s.syncChainBlocks(chainID)
		}
	}
}

func (s *Service) syncChainBlocks(chainID string) {
	log.Printf("[%s] Syncing blocks for chain %s", s.GetName(), chainID)
	
	var pageToken string
	for {
		blocks, nextPageToken, err := s.GetAPI().Blocks.GetBlocks(chainID, s.pageSize, pageToken)
		if err != nil {
			log.Printf("[%s] Error fetching blocks for chain %s: %v", s.GetName(), chainID, err)
			return
		}
		
		for _, block := range blocks {
			s.processBlock(block)
		}
		
		if nextPageToken == "" {
			break
		}
		pageToken = nextPageToken
	}
}

func (s *Service) processBlock(block types.Block) {
	s.blocksMutex.RLock()
	blockMap, exists := s.processedBlocks[block.ChainID]
	s.blocksMutex.RUnlock()
	
	if !exists {
		s.blocksMutex.Lock()
		s.processedBlocks[block.ChainID] = make(map[string]time.Time)
		blockMap = s.processedBlocks[block.ChainID]
		s.blocksMutex.Unlock()
	}
	
	// Skip blocks we've processed recently
	s.blocksMutex.RLock()
	lastProcessed, blockExists := blockMap[block.BlockNumber]
	s.blocksMutex.RUnlock()
	
	if blockExists && time.Since(lastProcessed) < s.GetPollInterval() {
		return
	}
	
	// Publish event for block
	err := s.PublishEvent(types.EventBlockCreated, types.BlockEvent{
		Type:  types.EventBlockCreated,
		Block: block,
	})
	
	if err != nil {
		log.Printf("[%s] Error publishing block event: %v", s.GetName(), err)
		return
	}
	
	s.blocksMutex.Lock()
	s.processedBlocks[block.ChainID][block.BlockNumber] = time.Now()
	s.blocksMutex.Unlock()
} 