package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/event"
	"github.com/Panorama-Block/avax/internal/types"
)

// Base provides common functionality for all services
type Base struct {
	api          *api.API
	eventManager *event.Manager
	pollInterval time.Duration
	workerCount  int
	workerWg     sync.WaitGroup
	stopChan     chan struct{}
	context      context.Context
	cancelFunc   context.CancelFunc
	name         string
	isRunning    bool
	runningMutex sync.Mutex
}

// ServiceOption is a function that configures a Service
type ServiceOption func(*Base)

// NewBase creates a new base service
func NewBase(api *api.API, eventManager *event.Manager, name string, options ...ServiceOption) *Base {
	ctx, cancel := context.WithCancel(context.Background())

	base := &Base{
		api:          api,
		eventManager: eventManager,
		pollInterval: 30 * time.Second,
		workerCount:  5,
		stopChan:     make(chan struct{}),
		context:      ctx,
		cancelFunc:   cancel,
		name:         name,
	}

	// Apply options
	for _, option := range options {
		option(base)
	}

	return base
}

// WithPollInterval sets the poll interval
func WithPollInterval(interval time.Duration) ServiceOption {
	return func(b *Base) {
		b.pollInterval = interval
	}
}

// WithWorkerCount sets the worker count
func WithWorkerCount(count int) ServiceOption {
	return func(b *Base) {
		b.workerCount = count
	}
}

// Start starts the service
func (b *Base) Start() error {
	b.runningMutex.Lock()
	defer b.runningMutex.Unlock()

	if b.isRunning {
		log.Printf("[%s] Service already running", b.name)
		return nil
	}

	log.Printf("[%s] Starting base service", b.name)
	b.isRunning = true
	return nil
}

// Stop stops the service
func (b *Base) Stop() {
	b.runningMutex.Lock()
	defer b.runningMutex.Unlock()

	if !b.isRunning {
		return
	}

	b.cancelFunc()
	close(b.stopChan)
	b.workerWg.Wait()
	b.isRunning = false
}

// IsRunning returns whether the service is running
func (b *Base) IsRunning() bool {
	b.runningMutex.Lock()
	defer b.runningMutex.Unlock()
	return b.isRunning
}

// PublishEvent publishes an event
func (b *Base) PublishEvent(eventType string, data interface{}) error {
	return b.eventManager.PublishEvent(types.Event{
		Type: eventType,
		Data: data,
	})
}

// RunWorker runs a worker with the specified ID
func (b *Base) RunWorker(id int, workerFunc func(context.Context, int)) {
	log.Printf("[%s] Starting worker %d", b.name, id)
	b.workerWg.Add(1)
	go func() {
		defer b.workerWg.Done()
		workerFunc(b.context, id)
	}()
}

// GetContext returns the service context
func (b *Base) GetContext() context.Context {
	return b.context
}

// GetAPI returns the API client
func (b *Base) GetAPI() *api.API {
	return b.api
}

// GetEventManager returns the event manager
func (b *Base) GetEventManager() *event.Manager {
	return b.eventManager
}

// GetWorkerCount returns the worker count
func (b *Base) GetWorkerCount() int {
	return b.workerCount
}

// GetPollInterval returns the poll interval
func (b *Base) GetPollInterval() time.Duration {
	return b.pollInterval
}

// GetName returns the service name
func (b *Base) GetName() string {
	return b.name
}
