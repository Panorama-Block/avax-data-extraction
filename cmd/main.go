package main

import (
	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
)

func main() {
	cfg := config.LoadConfig()
	client := api.NewClient(cfg.APIUrl, cfg.APIKey)
	producer := kafka.NewProducer(cfg.KafkaBroker, cfg.KafkaTopic)

	extractor.StartPipeline(client, producer)
}