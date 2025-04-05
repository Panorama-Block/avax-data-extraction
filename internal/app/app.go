package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/event"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/service"
	"github.com/Panorama-Block/avax/internal/service/block"
	"github.com/Panorama-Block/avax/internal/service/chain"
	"github.com/Panorama-Block/avax/internal/service/metrics"
	"github.com/Panorama-Block/avax/internal/types"
	"github.com/Panorama-Block/avax/internal/websocket"
)

// MockWebhookServer is a placeholder for the actual webhook server
type MockWebhookServer struct {
	eventChan chan types.WebhookEvent
	port      string
	running   bool
}

// NewMockWebhookServer creates a new mock webhook server
func NewMockWebhookServer(eventChan chan types.WebhookEvent, port string) *MockWebhookServer {
	return &MockWebhookServer{
		eventChan: eventChan,
		port:      port,
		running:   false,
	}
}

// Start starts the mock webhook server
func (s *MockWebhookServer) Start() error {
	s.running = true
	log.Printf("Mock Webhook server started on port %s", s.port)
	return nil
}

// Stop stops the mock webhook server
func (s *MockWebhookServer) Stop() {
	s.running = false
	log.Println("Mock Webhook server stopped")
}

// App represents the main application
type App struct {
	config        *config.Config
	api           *api.API
	eventManager  *event.Manager
	kafkaProducer *kafka.EventProducer
	services      []Service
	wsServer      *websocket.Server
	whServer      *MockWebhookServer
	context       context.Context
	cancelFunc    context.CancelFunc
	running       bool
	runningMutex  sync.Mutex
	mockMode      bool
}

// Service interface for all services
type Service interface {
	Start() error
	Stop()
	IsRunning() bool
	GetName() string
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize API
	avalancheAPI := api.NewAPI(cfg.APIUrl, cfg.APIKey)
	
	// Initialize Event Manager
	eventManager := event.NewManager(cfg.Workers, 10000)
	
	app := &App{
		config:       cfg,
		api:          avalancheAPI,
		eventManager: eventManager,
		services:     make([]Service, 0),
		context:      ctx,
		cancelFunc:   cancel,
		mockMode:     true,  // Enable mock mode for demonstration
	}
	
	return app
}

// mockEventProcessor is a simple processor that prints events instead of sending to Kafka
func mockEventProcessor(events []types.Event) error {
	for _, evt := range events {
		jsonData, err := json.MarshalIndent(evt, "", "  ")
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}
		fmt.Printf("\n=== Event Type: %s ===\n%s\n", evt.Type, string(jsonData))
	}
	return nil
}

// SetupServices initializes all services
func (a *App) SetupServices() error {
	if a.mockMode {
		log.Println("Running in MOCK MODE - No Kafka connections will be made")
		
		// In mock mode, we register our mock processor for all event types
		a.setupMockEventProcessors()
	} else {
		// Initialize Kafka producer
		baseProducer := kafka.NewProducer(a.config)
		a.kafkaProducer = kafka.NewEventProducer(baseProducer)
		
		// Set up Kafka topic mappings
		a.setupTopicMappings()
		
		// Register batch processor for each event type
		a.eventManager.RegisterBatchProcessor(
			types.EventBlockCreated, 
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventBlockUpdated, 
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventChainCreated, 
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventChainUpdated, 
			a.kafkaProducer.ProcessEvents,
		)
		
		// Register batch processors for metrics events
		a.eventManager.RegisterBatchProcessor(
			types.EventActivityMetricsUpdated,
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventPerformanceMetricsUpdated,
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventGasMetricsUpdated,
			a.kafkaProducer.ProcessEvents,
		)
		a.eventManager.RegisterBatchProcessor(
			types.EventCumulativeMetricsUpdated,
			a.kafkaProducer.ProcessEvents,
		)
		
		// Initialize WebSocket server if port is configured
		if a.config.WebSocketPort != "" {
			consumer, err := kafka.NewConsumer(a.config.KafkaBroker, "websocket-group", nil)
			if err != nil {
				return err
			}
			
			a.wsServer = websocket.NewServer(a.config.WebSocketPort, consumer)
		}
	}
	
	// Initialize Webhook server (no Kafka dependencies)
	eventChan := make(chan types.WebhookEvent, 1000)
	a.whServer = NewMockWebhookServer(eventChan, a.config.WebhookPort)
	
	// Add core services
	chainService := chain.NewService(
		a.api, 
		a.eventManager, 
		service.WithPollInterval(30*time.Second),
		service.WithWorkerCount(1),
	)
	a.services = append(a.services, chainService)
	
	blockService := block.NewService(
		a.api, 
		a.eventManager, 
		service.WithPollInterval(15*time.Second),
		service.WithWorkerCount(5),
	)
	blockService.SetChains([]string{"43114"}) // Avalanche C-Chain
	a.services = append(a.services, blockService)
	
	// Add metrics services
	activityMetricsService := metrics.NewActivityService(
		a.api,
		a.eventManager,
		5*time.Minute,  // Collection interval
		30*time.Minute, // Lookback period
		service.WithWorkerCount(2),
	)
	activityMetricsService.SetChains([]string{"43114"}) // Avalanche C-Chain
	a.services = append(a.services, activityMetricsService)
	
	performanceMetricsService := metrics.NewPerformanceService(
		a.api,
		a.eventManager,
		1*time.Minute,  // Collection interval (more frequent for real-time performance monitoring)
		15*time.Minute, // Lookback period
		service.WithWorkerCount(2),
	)
	performanceMetricsService.SetChains([]string{"43114"}) // Avalanche C-Chain
	a.services = append(a.services, performanceMetricsService)
	
	gasMetricsService := metrics.NewGasService(
		a.api,
		a.eventManager,
		5*time.Minute,  // Collection interval
		30*time.Minute, // Lookback period
		service.WithWorkerCount(2),
	)
	gasMetricsService.SetChains([]string{"43114"}) // Avalanche C-Chain
	a.services = append(a.services, gasMetricsService)
	
	cumulativeMetricsService := metrics.NewCumulativeService(
		a.api,
		a.eventManager,
		15*time.Minute, // Collection interval (less frequent as it doesn't change as rapidly)
		service.WithWorkerCount(1),
	)
	cumulativeMetricsService.SetChains([]string{"43114"}) // Avalanche C-Chain
	a.services = append(a.services, cumulativeMetricsService)
	
	return nil
}

// setupMockEventProcessors registers mock processors for all event types
func (a *App) setupMockEventProcessors() {
	// Block events
	a.eventManager.RegisterBatchProcessor(types.EventBlockCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventBlockUpdated, mockEventProcessor)
	
	// Chain events
	a.eventManager.RegisterBatchProcessor(types.EventChainCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventChainUpdated, mockEventProcessor)
	
	// Transaction events
	a.eventManager.RegisterBatchProcessor(types.EventTransactionCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventTransactionUpdated, mockEventProcessor)
	
	// Transfer events
	a.eventManager.RegisterBatchProcessor(types.EventERC20Transfer, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventERC721Transfer, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventERC1155Transfer, mockEventProcessor)
	
	// Log events
	a.eventManager.RegisterBatchProcessor(types.EventLogCreated, mockEventProcessor)
	
	// Validator events
	a.eventManager.RegisterBatchProcessor(types.EventValidatorCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventValidatorUpdated, mockEventProcessor)
	
	// Delegator events
	a.eventManager.RegisterBatchProcessor(types.EventDelegatorCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventDelegatorUpdated, mockEventProcessor)
	
	// Subnet events
	a.eventManager.RegisterBatchProcessor(types.EventSubnetCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventSubnetUpdated, mockEventProcessor)
	
	// Blockchain events
	a.eventManager.RegisterBatchProcessor(types.EventBlockchainCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventBlockchainUpdated, mockEventProcessor)
	
	// Teleporter events
	a.eventManager.RegisterBatchProcessor(types.EventTeleporterTxCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventTeleporterTxUpdated, mockEventProcessor)
	
	// Token events
	a.eventManager.RegisterBatchProcessor(types.EventTokenCreated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventTokenUpdated, mockEventProcessor)
	
	// Metrics events
	a.eventManager.RegisterBatchProcessor(types.EventMetricsUpdated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventActivityMetricsUpdated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventPerformanceMetricsUpdated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventGasMetricsUpdated, mockEventProcessor)
	a.eventManager.RegisterBatchProcessor(types.EventCumulativeMetricsUpdated, mockEventProcessor)
}

// setupTopicMappings sets up mappings between event types and Kafka topics
func (a *App) setupTopicMappings() {
	a.kafkaProducer.RegisterTopicMapping(types.EventBlockCreated, a.config.KafkaTopicBlocks)
	a.kafkaProducer.RegisterTopicMapping(types.EventBlockUpdated, a.config.KafkaTopicBlocks)
	a.kafkaProducer.RegisterTopicMapping(types.EventChainCreated, a.config.KafkaTopicChains)
	a.kafkaProducer.RegisterTopicMapping(types.EventChainUpdated, a.config.KafkaTopicChains)
	a.kafkaProducer.RegisterTopicMapping(types.EventTransactionCreated, a.config.KafkaTopicTransactions)
	a.kafkaProducer.RegisterTopicMapping(types.EventTransactionUpdated, a.config.KafkaTopicTransactions)
	a.kafkaProducer.RegisterTopicMapping(types.EventERC20Transfer, a.config.KafkaTopicERC20)
	a.kafkaProducer.RegisterTopicMapping(types.EventERC721Transfer, a.config.KafkaTopicERC721)
	a.kafkaProducer.RegisterTopicMapping(types.EventERC1155Transfer, a.config.KafkaTopicERC1155)
	a.kafkaProducer.RegisterTopicMapping(types.EventLogCreated, a.config.KafkaTopicLogs)
	a.kafkaProducer.RegisterTopicMapping(types.EventValidatorCreated, a.config.KafkaTopicValidators)
	a.kafkaProducer.RegisterTopicMapping(types.EventValidatorUpdated, a.config.KafkaTopicValidators)
	a.kafkaProducer.RegisterTopicMapping(types.EventDelegatorCreated, a.config.KafkaTopicDelegators)
	a.kafkaProducer.RegisterTopicMapping(types.EventDelegatorUpdated, a.config.KafkaTopicDelegators)
	a.kafkaProducer.RegisterTopicMapping(types.EventSubnetCreated, a.config.KafkaTopicSubnets)
	a.kafkaProducer.RegisterTopicMapping(types.EventSubnetUpdated, a.config.KafkaTopicSubnets)
	a.kafkaProducer.RegisterTopicMapping(types.EventBlockchainCreated, a.config.KafkaTopicBlockchains)
	a.kafkaProducer.RegisterTopicMapping(types.EventBlockchainUpdated, a.config.KafkaTopicBlockchains)
	a.kafkaProducer.RegisterTopicMapping(types.EventTeleporterTxCreated, a.config.KafkaTopicBridges)
	a.kafkaProducer.RegisterTopicMapping(types.EventTeleporterTxUpdated, a.config.KafkaTopicBridges)
	a.kafkaProducer.RegisterTopicMapping(types.EventTokenCreated, a.config.KafkaTopicMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventTokenUpdated, a.config.KafkaTopicMetrics)
	
	// Map the new metrics event types to the metrics Kafka topic
	a.kafkaProducer.RegisterTopicMapping(types.EventMetricsUpdated, a.config.KafkaTopicMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventActivityMetricsUpdated, a.config.KafkaTopicMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventPerformanceMetricsUpdated, a.config.KafkaTopicMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventGasMetricsUpdated, a.config.KafkaTopicMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventCumulativeMetricsUpdated, a.config.KafkaTopicMetrics)
}

// Start starts the application
func (a *App) Start() error {
	a.runningMutex.Lock()
	defer a.runningMutex.Unlock()
	
	if a.running {
		return nil
	}
	
	log.Println("Starting application in mock mode...")
	log.Printf("Config: API URL=%s, WebhookPort=%s, WebSocketPort=%s", 
		a.config.APIUrl, a.config.WebhookPort, a.config.WebSocketPort)
	
	// Start event manager
	if err := a.eventManager.Start(a.context); err != nil {
		return err
	}
	log.Println("Event manager started successfully")
	
	// Start WebSocket server if configured and not in mock mode
	if a.wsServer != nil && !a.mockMode {
		go func() {
			if err := a.wsServer.Start(); err != nil {
				log.Printf("Error starting WebSocket server: %v", err)
			}
		}()
	}
	
	// Start Webhook server
	go func() {
		if err := a.whServer.Start(); err != nil {
			log.Printf("Error starting Webhook server: %v", err)
		}
	}()
	
	// Start all services
	log.Printf("Starting %d services...", len(a.services))
	for i, svc := range a.services {
		log.Printf("Starting service %d/%d: %s", i+1, len(a.services), svc.GetName())
		if err := svc.Start(); err != nil {
			log.Printf("Error starting service %s: %v", svc.GetName(), err)
		} else {
			log.Printf("Service %s started successfully", svc.GetName())
		}
	}
	
	a.running = true
	log.Println("Application started successfully")
	
	// Log a message every 5 seconds for interactive testing
	if a.mockMode {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			
			i := 0
			for {
				select {
				case <-a.context.Done():
					return
				case <-ticker.C:
					i++
					log.Printf("[MOCK] Application running for %d seconds...", i*5)
					
					// After 30 seconds, simulate a chain event to trigger metrics collection
					if i == 6 {
						log.Println("[MOCK] Simulating chain event...")
						chainEvent := types.ChainEvent{
							Type: types.EventChainCreated,
							Chain: types.Chain{
								ChainID: "43114",
								ChainName: "Avalanche C-Chain",
								VmName: "EVM",
							},
						}
						a.eventManager.PublishEvent(types.Event{
							Type: types.EventChainCreated,
							Data: chainEvent,
						})
					}
				}
			}
		}()
	}
	
	return nil
}

// Stop stops the application
func (a *App) Stop() {
	a.runningMutex.Lock()
	defer a.runningMutex.Unlock()
	
	if !a.running {
		return
	}
	
	log.Println("Stopping application...")
	
	// Stop all services
	for _, svc := range a.services {
		if svc.IsRunning() {
			svc.Stop()
		}
	}
	
	// Stop WebSocket server if configured
	if a.wsServer != nil {
		a.wsServer.Stop(a.context)
	}
	
	// Stop Webhook server
	if a.whServer != nil {
		a.whServer.Stop()
	}
	
	// Cancel context to stop all operations
	a.cancelFunc()
	
	a.running = false
	log.Println("Application stopped successfully")
} 