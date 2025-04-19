package extractor

import (
	"context"
)

// ServiceAdapter adapts components to match the Service interface
type ServiceAdapter struct {
	name      string
	startFunc func() error
	stopFunc  func()
	isRunning func() bool
}

// NewServiceAdapter creates a new service adapter
func NewServiceAdapter(name string, startFunc func() error, stopFunc func(), isRunning func() bool) *ServiceAdapter {
	return &ServiceAdapter{
		name:      name,
		startFunc: startFunc,
		stopFunc:  stopFunc,
		isRunning: isRunning,
	}
}

// Start starts the service
func (s *ServiceAdapter) Start() error {
	return s.startFunc()
}

// Stop stops the service
func (s *ServiceAdapter) Stop() {
	s.stopFunc()
}

// IsRunning returns the running status of the service
func (s *ServiceAdapter) IsRunning() bool {
	return s.isRunning()
}

// GetName returns the name of the service
func (s *ServiceAdapter) GetName() string {
	return s.name
}

// NewRealTimeExtractorAdapter creates a service adapter for RealTimeExtractor
func NewRealTimeExtractorAdapter(ctx context.Context, extractor *RealTimeExtractor) *ServiceAdapter {
	return NewServiceAdapter(
		extractor.GetName(),
		func() error {
			return extractor.Start(ctx)
		},
		extractor.Stop,
		extractor.IsRunning,
	)
}

// NewTransactionPipelineAdapter creates a service adapter for TransactionPipeline
func NewTransactionPipelineAdapter(ctx context.Context, pipeline *TransactionPipeline) *ServiceAdapter {
	return NewServiceAdapter(
		pipeline.GetName(),
		func() error {
			return pipeline.Start(ctx)
		},
		pipeline.Stop,
		pipeline.IsRunning,
	)
} 