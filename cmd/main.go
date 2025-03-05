// cmd/main.go
package main

import (
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/config"
    "github.com/Panorama-Block/avax/internal/extractor"
    "github.com/Panorama-Block/avax/internal/kafka"
    "github.com/Panorama-Block/avax/internal/types"
    "github.com/Panorama-Block/avax/internal/webhook"
)

func main() {
    cfg := config.LoadConfig()
    client := api.NewClient(cfg.APIUrl, cfg.APIKey)
    producer := kafka.NewProducer(cfg)

    // WebSocket (sem delay)
    go extractor.StartBlockWebSocket(client, producer)

    // Webhook
    eventChan := make(chan types.WebhookEvent, 100)
    wh := webhook.NewWebhookServer(eventChan, cfg.WebhookPort)
    go wh.Start()

    // Pipeline de tx do webhook
    go extractor.StartTxPipeline(client, producer, eventChan, cfg.Workers)

    // Chain pipeline
    go extractor.StartChainPipeline(client, producer)

    // Cronjobs
    go extractor.StartCronJobs(client, producer, "mainnet")

    // Metrics
    go extractor.StartMetricsPipeline(client, producer, 30*time.Second, cfg.TokenAddresses)

    select {}
}
