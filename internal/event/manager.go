package event

import (
	"context"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/types"
)

// Manager coordinates event processing throughout the application
type Manager struct {
	bus           *Bus
	batchConfig   map[string]*BatchConfig
	processors    map[string]*BatchProcessor
	lock          sync.RWMutex
	defaultConfig *BatchConfig
}

// BatchConfig holds configuration for batch processing
type BatchConfig struct {
	MaxBatchSize    int
	BatchTimeout    time.Duration
	EventBufferSize int
	WorkerCount     int
}

// NewManager creates a new event manager
func NewManager(busWorkerCount, busEventBufferSize int) *Manager {
	defaultConfig := &BatchConfig{
		MaxBatchSize:    100,
		BatchTimeout:    5 * time.Second,
		EventBufferSize: 1000,
		WorkerCount:     5,
	}

	return &Manager{
		bus:           NewBus(busWorkerCount, 100, busEventBufferSize),
		batchConfig:   make(map[string]*BatchConfig),
		processors:    make(map[string]*BatchProcessor),
		defaultConfig: defaultConfig,
	}
}

// SetBatchConfig sets the batch configuration for a specific event type
func (m *Manager) SetBatchConfig(eventType string, config *BatchConfig) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.batchConfig[eventType] = config
}

// GetBatchConfig gets the batch configuration for a specific event type
func (m *Manager) GetBatchConfig(eventType string) *BatchConfig {
	m.lock.RLock()
	defer m.lock.RUnlock()
	
	config, exists := m.batchConfig[eventType]
	if !exists {
		return m.defaultConfig
	}
	return config
}

// RegisterBatchProcessor registers a batch processor for a specific event type
func (m *Manager) RegisterBatchProcessor(
	eventType string,
	processBatch func([]types.Event) error,
) {
	config := m.GetBatchConfig(eventType)
	
	processor := NewBatchProcessor(
		config.MaxBatchSize,
		config.BatchTimeout,
		config.EventBufferSize,
		config.WorkerCount,
		processBatch,
	)
	
	m.lock.Lock()
	m.processors[eventType] = processor
	m.lock.Unlock()
	
	// Subscribe to the event bus
	m.bus.Subscribe(eventType, func(event types.Event) error {
		return processor.AddEvent(event)
	})
}

// PublishEvent publishes an event to the event bus
func (m *Manager) PublishEvent(event types.Event) error {
	return m.bus.Publish(event)
}

// Start starts the event manager and all registered processors
func (m *Manager) Start(ctx context.Context) error {
	// Start the event bus
	if err := m.bus.Start(ctx); err != nil {
		return err
	}
	
	// Start all batch processors
	m.lock.RLock()
	defer m.lock.RUnlock()
	
	for _, processor := range m.processors {
		processor.Start(ctx)
	}
	
	return nil
}

// Stop stops the event manager and all registered processors
func (m *Manager) Stop() {
	m.lock.RLock()
	defer m.lock.RUnlock()
	
	for _, processor := range m.processors {
		processor.Stop()
	}
	
	m.bus.Stop()
}

// Subscribe directly subscribes to the event bus
func (m *Manager) Subscribe(eventType string, subscriber Subscriber) {
	m.bus.Subscribe(eventType, subscriber)
} 