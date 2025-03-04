package main

import (
	"log"

	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/webhook"
	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/types"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	// Start API Client 
	client := api.NewClient(cfg.APIUrl, cfg.APIKey)

	// Start Kafka Producer
	producer := kafka.NewProducer(cfg.KafkaBroker, cfg.KafkaTopic)

	// Start Chain Pipeline 
	go extractor.StartChainPipeline(client, producer)

	// Channel to receive Webhook events
	eventChan := make(chan types.WebhookEvent, 100)

	// Start Webhook Server
	webhookServer := webhook.NewWebhookServer(eventChan, cfg.WebhookPort)
	go webhookServer.Start()

	// Start Transaction Pipeline
	extractor.StartTxPipeline(producer, eventChan, cfg.Workers)

	log.Println("End Application")
}
