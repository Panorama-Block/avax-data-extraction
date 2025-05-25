package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/avacloud"
	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/event"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/service"
	"github.com/Panorama-Block/avax/internal/service/block"
	"github.com/Panorama-Block/avax/internal/service/chain"
	"github.com/Panorama-Block/avax/internal/service/metrics"
	"github.com/Panorama-Block/avax/internal/types"
	"github.com/Panorama-Block/avax/internal/webhook"
	"github.com/Panorama-Block/avax/internal/websocket"
)

// WebhookServer interface for handling webhook events
type WebhookServer interface {
	Start() error
	Stop()
	SetTransactionPipeline(pipeline *extractor.TransactionPipeline)
}

// App represents the main application
type App struct {
	config        *config.Config
	api           *api.API
	eventManager  *event.Manager
	kafkaProducer *kafka.EventProducer
	wsServer      *websocket.Server
	whServer      WebhookServer
	services      []Service
	context       context.Context
	cancelFunc    context.CancelFunc
	running       bool
	runningMutex  sync.Mutex
	webhookID     string
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

	// Use the rate-limited API client with configuration
	apiClient := api.NewAPIWithConfig(cfg)

	eventManager := event.NewManager(cfg.Workers, 10000)

	return &App{
		config:       cfg,
		api:          apiClient,
		eventManager: eventManager,
		services:     []Service{},
		context:      ctx,
		cancelFunc:   cancel,
	}
}

// SetMockMode is now a no-op as mock mode has been removed
func (a *App) SetMockMode(enabled bool) {
	// No-op, mock mode removed
	log.Println("Mock mode has been removed, using real Kafka producer")
}

// SetupServices initializes all services
func (a *App) SetupServices() error {
	// Initialize Kafka producer
	baseProducer := kafka.NewProducer(a.config)
	log.Println("Using Kafka producer")

	a.kafkaProducer = kafka.NewEventProducer(baseProducer)

	// Set up Kafka topic mappings
	a.setupTopicMappings()

	// Create transaction pipeline for processing webhook events
	dataAPI := api.NewDataAPI(a.api.Client)
	txPipeline := extractor.NewTransactionPipeline(
		dataAPI,
		baseProducer,
		a.config.Workers,
		a.config.Workers*100, // Buffer size proportional to workers
	)

	// Start the pipeline and create service adapter
	if err := txPipeline.Start(a.context); err != nil {
		return fmt.Errorf("falha ao iniciar transaction pipeline: %w", err)
	}
	txPipelineService := extractor.NewTransactionPipelineAdapter(a.context, txPipeline)
	a.services = append(a.services, txPipelineService)

	// Initialize Webhook server
	eventChan := make(chan types.WebhookEvent, 1000)
	webhookServer := webhook.NewWebhookServer(eventChan, a.config.WebhookPort, a.config.WebhookSecret)
	a.whServer = webhookServer

	// Connect webhook to transaction pipeline
	a.whServer.SetTransactionPipeline(txPipeline)

	// Create AvaCloud webhook
	webhookID, err := avacloud.EnsureCChainWebhook(a.config)
	if err != nil {
		log.Printf("AVISO: Falha ao criar webhook na AvaCloud: %v", err)
	} else {
		a.webhookID = webhookID
		log.Printf("Webhook C-Chain configurado na AvaCloud: %s", webhookID)
	}

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
	a.eventManager.RegisterBatchProcessor(
		types.EventTransactionCreated,
		a.kafkaProducer.ProcessEvents,
	)
	a.eventManager.RegisterBatchProcessor(
		types.EventTransactionUpdated,
		a.kafkaProducer.ProcessEvents,
	)
	a.eventManager.RegisterBatchProcessor(
		types.EventERC20Transfer,
		a.kafkaProducer.ProcessEvents,
	)
	a.eventManager.RegisterBatchProcessor(
		types.EventERC721Transfer,
		a.kafkaProducer.ProcessEvents,
	)
	a.eventManager.RegisterBatchProcessor(
		types.EventERC1155Transfer,
		a.kafkaProducer.ProcessEvents,
	)
	a.eventManager.RegisterBatchProcessor(
		types.EventLogCreated,
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

	// Add RealTime extractor if enabled
	if a.config.EnableRealtime && a.config.ChainWsUrl != "" {
		realTimeExtractor := extractor.NewRealTimeExtractor(
			dataAPI,
			baseProducer,
			a.config.ChainWsUrl,
			"43114", // Avalanche C-Chain
			"Avalanche C-Chain",
		)

		// Start extractor and create adapter
		if err := realTimeExtractor.Start(a.context); err != nil {
			log.Printf("AVISO: Falha ao iniciar extrator em tempo real: %v", err)
		} else {
			rtExtractorService := extractor.NewRealTimeExtractorAdapter(a.context, realTimeExtractor)
			a.services = append(a.services, rtExtractorService)
		}

		// Start periodic jobs for other data
		cronJobManager := extractor.NewCronJobManager(a.api.Client, baseProducer, "mainnet")
		if err := cronJobManager.Start(); err != nil {
			log.Printf("AVISO: Falha ao iniciar gerenciador de tarefas cron: %v", err)
		} else {
			a.services = append(a.services, cronJobManager)
		}
	}

	// Initialize WebSocket server if port is configured
	if a.config.WebSocketPort != "" {
		consumer, err := kafka.NewConsumer(a.config.KafkaBroker, "websocket-group", nil)
		if err != nil {
			return err
		}

		a.wsServer = websocket.NewServer(a.config.WebSocketPort, consumer)
	}

	// Add core services
	chainService := chain.NewService(
		a.api,
		a.eventManager,
		service.WithPollInterval(30*time.Second),
		service.WithWorkerCount(1),
	)
	a.services = append(a.services, chainService)
	log.Printf("Chain service created and added to services list")

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

// setupTopicMappings sets up mappings between event types and Kafka topics
func (a *App) setupTopicMappings() {
	log.Printf("Setting up Kafka topic mappings")
	log.Printf("Mapping chain events to topic: %s", a.config.KafkaTopicChains)

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

	// Map general metrics events to the default metrics topic
	a.kafkaProducer.RegisterTopicMapping(types.EventMetricsUpdated, a.config.KafkaTopicMetrics)

	// Map specific metrics event types to their dedicated Kafka topics
	a.kafkaProducer.RegisterTopicMapping(types.EventActivityMetricsUpdated, a.config.KafkaTopicActivityMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventPerformanceMetricsUpdated, a.config.KafkaTopicPerformanceMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventGasMetricsUpdated, a.config.KafkaTopicGasMetrics)
	a.kafkaProducer.RegisterTopicMapping(types.EventCumulativeMetricsUpdated, a.config.KafkaTopicCumulativeMetrics)
}

// Start starts the application
func (a *App) Start() error {
	a.runningMutex.Lock()
	defer a.runningMutex.Unlock()

	if a.running {
		return nil
	}

	log.Println("Iniciando aplicação...")
	log.Printf("Configuração: API URL=%s, WebhookPort=%s, WebSocketPort=%s",
		a.config.APIUrl, a.config.WebhookPort, a.config.WebSocketPort)

	// Start event manager
	if err := a.eventManager.Start(a.context); err != nil {
		return fmt.Errorf("falha ao iniciar gerenciador de eventos: %w", err)
	}
	log.Println("Gerenciador de eventos iniciado com sucesso")

	// Start Webhook server - this is critical
	log.Println("Iniciando servidor de webhook...")
	if err := a.whServer.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar servidor webhook: %w", err)
	}
	log.Println("Servidor webhook iniciado com sucesso")

	// Start WebSocket server if configured
	if a.wsServer != nil {
		log.Println("Iniciando servidor WebSocket...")
		go func() {
			if err := a.wsServer.Start(); err != nil {
				log.Printf("Erro ao iniciar servidor WebSocket: %v", err)
			} else {
				log.Println("Servidor WebSocket iniciado com sucesso")
			}
		}()
	}

	// Start all services
	log.Printf("Iniciando %d serviços...", len(a.services))
	for i, svc := range a.services {
		log.Printf("Iniciando serviço %d/%d: %s", i+1, len(a.services), svc.GetName())
		if err := svc.Start(); err != nil {
			log.Printf("Erro ao iniciar serviço %s: %v", svc.GetName(), err)
		} else {
			log.Printf("Serviço %s iniciado com sucesso", svc.GetName())
		}
	}

	a.running = true
	log.Println("Aplicação iniciada com sucesso. Escutando por webhooks na porta", a.config.WebhookPort)

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
