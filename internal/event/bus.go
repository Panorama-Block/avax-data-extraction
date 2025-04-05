package event

import (
	"context"
	"sync"

	"github.com/Panorama-Block/avax/internal/types"
)

// Subscriber is a function that processes events
type Subscriber func(event types.Event) error

// Bus manages event publishing and subscription
type Bus struct {
	subscribers     map[string][]Subscriber
	mutex           sync.RWMutex
	workerCount     int
	eventChan       chan types.Event
	shutdown        chan struct{}
	batchSize       int
	eventBufferSize int
}

// NewBus creates a new event bus with the specified configuration
func NewBus(workerCount, batchSize, eventBufferSize int) *Bus {
	return &Bus{
		subscribers:     make(map[string][]Subscriber),
		workerCount:     workerCount,
		eventChan:       make(chan types.Event, eventBufferSize),
		shutdown:        make(chan struct{}),
		batchSize:       batchSize,
		eventBufferSize: eventBufferSize,
	}
}

// Subscribe registers a subscriber for a specific event type
func (b *Bus) Subscribe(eventType string, subscriber Subscriber) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.subscribers[eventType]; !exists {
		b.subscribers[eventType] = make([]Subscriber, 0)
	}
	b.subscribers[eventType] = append(b.subscribers[eventType], subscriber)
}

// Publish sends an event to all subscribers of the event type
func (b *Bus) Publish(event types.Event) error {
	select {
	case b.eventChan <- event:
		return nil
	default:
		return ErrEventBusFull
	}
}

// PublishBlocking sends an event to the event bus, blocking if the bus is full
func (b *Bus) PublishBlocking(event types.Event) {
	b.eventChan <- event
}

// Start begins processing events
func (b *Bus) Start(ctx context.Context) error {
	for i := 0; i < b.workerCount; i++ {
		go b.worker(ctx, i)
	}
	return nil
}

// Stop stops processing events
func (b *Bus) Stop() {
	close(b.shutdown)
}

func (b *Bus) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.shutdown:
			return
		case event := <-b.eventChan:
			b.processEvent(event)
		}
	}
}

func (b *Bus) processEvent(event types.Event) {
	b.mutex.RLock()
	subscribers, exists := b.subscribers[event.Type]
	b.mutex.RUnlock()

	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		// Error handling can be improved as needed
		_ = subscriber(event)
	}
}

// ErrEventBusFull is returned when the event bus channel is full
var ErrEventBusFull = &EventBusError{message: "event bus is full"}

// EventBusError represents an error in the event bus
type EventBusError struct {
	message string
}

func (e *EventBusError) Error() string {
	return e.message
} 