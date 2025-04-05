package event

import (
	"context"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/types"
)

// BatchProcessor processes events in batches for improved performance
type BatchProcessor struct {
	maxBatchSize    int
	batchTimeout    time.Duration
	eventChan       chan types.Event
	processBatch    func([]types.Event) error
	currentBatch    []types.Event
	mutex           sync.Mutex
	workerCount     int
	processingWg    sync.WaitGroup
	shutdown        chan struct{}
	eventBufferSize int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(
	maxBatchSize int,
	batchTimeout time.Duration,
	eventBufferSize int,
	workerCount int,
	processBatch func([]types.Event) error,
) *BatchProcessor {
	return &BatchProcessor{
		maxBatchSize:    maxBatchSize,
		batchTimeout:    batchTimeout,
		eventChan:       make(chan types.Event, eventBufferSize),
		processBatch:    processBatch,
		currentBatch:    make([]types.Event, 0, maxBatchSize),
		workerCount:     workerCount,
		shutdown:        make(chan struct{}),
		eventBufferSize: eventBufferSize,
	}
}

// AddEvent adds an event to the batch processor
func (b *BatchProcessor) AddEvent(event types.Event) error {
	select {
	case b.eventChan <- event:
		return nil
	default:
		return ErrBatchProcessorFull
	}
}

// Start begins processing batches
func (b *BatchProcessor) Start(ctx context.Context) {
	for i := 0; i < b.workerCount; i++ {
		b.processingWg.Add(1)
		go b.processingWorker(ctx, i)
	}

	// Batch timer worker
	go b.batchTimerWorker(ctx)
}

// Stop stops the batch processor
func (b *BatchProcessor) Stop() {
	close(b.shutdown)
	b.processingWg.Wait()
}

func (b *BatchProcessor) processingWorker(ctx context.Context, id int) {
	defer b.processingWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.shutdown:
			return
		case event := <-b.eventChan:
			b.addToBatch(event)
		}
	}
}

func (b *BatchProcessor) batchTimerWorker(ctx context.Context) {
	ticker := time.NewTicker(b.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.shutdown:
			return
		case <-ticker.C:
			b.flushBatchIfNeeded(true)
		}
	}
}

func (b *BatchProcessor) addToBatch(event types.Event) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.currentBatch = append(b.currentBatch, event)
	if len(b.currentBatch) >= b.maxBatchSize {
		b.flushBatchLocked()
	}
}

func (b *BatchProcessor) flushBatchIfNeeded(force bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.currentBatch) > 0 && (force || len(b.currentBatch) >= b.maxBatchSize) {
		b.flushBatchLocked()
	}
}

func (b *BatchProcessor) flushBatchLocked() {
	if len(b.currentBatch) == 0 {
		return
	}

	// Make a copy of the current batch
	batch := make([]types.Event, len(b.currentBatch))
	copy(batch, b.currentBatch)
	b.currentBatch = b.currentBatch[:0]

	// Process the batch in a separate goroutine
	go func(eventBatch []types.Event) {
		_ = b.processBatch(eventBatch)
	}(batch)
}

// ErrBatchProcessorFull is returned when the batch processor channel is full
var ErrBatchProcessorFull = &BatchProcessorError{message: "batch processor is full"}

// BatchProcessorError represents an error in the batch processor
type BatchProcessorError struct {
	message string
}

func (e *BatchProcessorError) Error() string {
	return e.message
} 