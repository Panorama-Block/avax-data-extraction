package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	confluent "github.com/confluentinc/confluent-kafka-go/kafka"
	
	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/webhook"
	"github.com/Panorama-Block/avax/internal/websocket"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// Parse command line flags
	_ = flag.String("config", "", "Path to config file") // Ignoring the configPath for now
	flag.Parse()

	// Create Kafka configuration
	kafkaConfig := confluent.ConfigMap{}
	
	// Set Kafka configuration options
	if err := kafkaConfig.SetKey("bootstrap.servers", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); err != nil {
		log.Fatalf("Error setting Kafka bootstrap servers: %v", err)
	}
	
	// Create Kafka producer
	producer, err := kafka.NewProducer(
		os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		os.Getenv("KAFKA_TOPIC_PREFIX"),
		&kafkaConfig,
	)
	if err != nil {
		log.Fatalf("Error creating Kafka producer: %v", err)
	}
	defer producer.Close()
	
	// Create Kafka consumer
	consumerConfig := confluent.ConfigMap{}
	if err := consumerConfig.SetKey("bootstrap.servers", os.Getenv("KAFKA_BOOTSTRAP_SERVERS")); err != nil {
		log.Fatalf("Error setting Kafka bootstrap servers for consumer: %v", err)
	}
	
	if err := consumerConfig.SetKey("group.id", "avax-data-pipeline"); err != nil {
		log.Fatalf("Error setting Kafka group ID: %v", err)
	}
	
	consumer, err := kafka.NewConsumer(
		os.Getenv("KAFKA_BOOTSTRAP_SERVERS"),
		"avax-data-pipeline",
		&consumerConfig,
	)
	if err != nil {
		log.Fatalf("Error creating Kafka consumer: %v", err)
	}
	defer consumer.Close()
	
	// Create API clients
	client := api.NewClient(os.Getenv("AVACLOUD_API_URL"), os.Getenv("AVACLOUD_API_KEY"))
	dataAPI := api.NewDataAPI(client)
	metricsAPI := api.NewMetricsAPI(client)
	
	// Create the context that will be used for all operations
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Set up real-time extractors for specific chains
	if os.Getenv("ENABLE_REALTIME_EXTRACTOR") == "true" {
		// C-Chain extractor
		cChainExtractor := extractor.NewRealTimeExtractor(
			dataAPI,
			producer,
			os.Getenv("CCHAIN_WS_ENDPOINT"), // e.g. wss://api.avacloud.io/v1/chains/43114/ws
			"43114", // C-Chain ID
			"C-Chain",
		)
		
		if err := cChainExtractor.Start(ctx); err != nil {
			log.Fatalf("Error starting C-Chain extractor: %v", err)
		}
		defer cChainExtractor.Stop()
		
		// Add more chain extractors as needed
	}
	
	// Set up chain data pipeline
	if os.Getenv("ENABLE_CHAIN_PIPELINE") == "true" {
		chainPipeline := extractor.NewChainPipeline(
			dataAPI,
			metricsAPI,
			producer,
			"mainnet", // or "fuji" for testnet
			30*time.Minute, // Update interval
		)
		
		if err := chainPipeline.Start(ctx); err != nil {
			log.Fatalf("Error starting chain pipeline: %v", err)
		}
		defer chainPipeline.Stop()
	}
	
	// Set up transaction pipeline
	if os.Getenv("ENABLE_TRANSACTION_PIPELINE") == "true" {
		workerCount, _ := strconv.Atoi(os.Getenv("TRANSACTION_PIPELINE_WORKERS"))
		if workerCount <= 0 {
			workerCount = 5 // Default worker count
		}
		
		bufferSize, _ := strconv.Atoi(os.Getenv("TRANSACTION_PIPELINE_BUFFER"))
		if bufferSize <= 0 {
			bufferSize = 1000 // Default buffer size
		}
		
		transactionPipeline := extractor.NewTransactionPipeline(
			dataAPI,
			producer,
			workerCount,
			bufferSize,
		)
		
		if err := transactionPipeline.Start(ctx); err != nil {
			log.Fatalf("Error starting transaction pipeline: %v", err)
		}
		defer transactionPipeline.Stop()
		
		// Set up webhook server to receive events
		if os.Getenv("ENABLE_WEBHOOK_SERVER") == "true" {
			webhookPort, _ := strconv.Atoi(os.Getenv("WEBHOOK_PORT"))
			if webhookPort <= 0 {
				webhookPort = 8080 // Default webhook port
			}
			
			webhookServer := webhook.NewServer(
				transactionPipeline,
				os.Getenv("WEBHOOK_SHARED_SECRET"),
				webhookPort,
			)
			
			if err := webhookServer.Start(); err != nil {
				log.Fatalf("Error starting webhook server: %v", err)
			}
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				webhookServer.Stop(shutdownCtx)
			}()
		}
	}
	
	// Set up metrics pipeline
	if os.Getenv("ENABLE_METRICS_PIPELINE") == "true" {
		// Get chains to monitor
		chains := []string{
			"43114", // C-Chain
			// Add more chains as needed
		}
		
		metricsPipeline := extractor.NewMetricsPipeline(
			metricsAPI,
			producer,
			"mainnet", // or "fuji" for testnet
			1*time.Hour, // Update interval
		)
		
		metricsPipeline.SetChains(chains)
		
		if err := metricsPipeline.Start(ctx); err != nil {
			log.Fatalf("Error starting metrics pipeline: %v", err)
		}
		defer metricsPipeline.Stop()
	}
	
	// Set up cron job manager
	if os.Getenv("ENABLE_CRON_JOBS") == "true" {
		cronJobManager := extractor.NewCronJobManager(
			dataAPI,
			metricsAPI,
			producer,
			"mainnet", // or "fuji" for testnet
		)
		
		// Add default jobs
		cronJobManager.CreateDefaultJobs()
		
		if err := cronJobManager.Start(ctx); err != nil {
			log.Fatalf("Error starting cron job manager: %v", err)
		}
		defer cronJobManager.Stop()
	}
	
	// Set up WebSocket server for frontend
	if os.Getenv("ENABLE_WEBSOCKET_SERVER") == "true" {
		wsPort, _ := strconv.Atoi(os.Getenv("WEBSOCKET_PORT"))
		if wsPort <= 0 {
			wsPort = 8081 // Default WebSocket port
		}
		
		wsServer := websocket.NewServer(wsPort, consumer)
		
		if err := wsServer.Start(); err != nil {
			log.Fatalf("Error starting WebSocket server: %v", err)
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			wsServer.Stop(shutdownCtx)
		}()
	}
	
	// Set up health check HTTP server
	httpPort, _ := strconv.Atoi(os.Getenv("HTTP_PORT"))
	if httpPort <= 0 {
		httpPort = 8082 // Default HTTP port
	}
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", httpPort),
	}
	
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()
	
	// Configure graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	
	// Create a deadline context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	// Shutdown the HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	
	// Cancel the main context to signal all components to stop
	cancel()
	
	log.Println("Shutdown complete")
}
