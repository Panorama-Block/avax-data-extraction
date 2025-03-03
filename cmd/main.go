package main

import (
	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/webhook"
	"github.com/Panorama-Block/avax/internal/types"
)

func main() {
	cfg := config.LoadConfig()
	// client := api.NewClient(cfg.APIUrl, cfg.APIKey)
	producer := kafka.NewProducer(cfg.KafkaBroker, cfg.KafkaTopic)

	// extractor.StartChainPipeline(client, producer)

	// Canal de eventos Webhook
	eventChan := make(chan types.WebhookEvent, 100)

	// Iniciar Webhook
	webhookServer := webhook.NewWebhookServer(eventChan, cfg.WebhookPort)
	go webhookServer.Start()
	extractor.StartTxPipeline(producer, eventChan, cfg.Workers)
}
