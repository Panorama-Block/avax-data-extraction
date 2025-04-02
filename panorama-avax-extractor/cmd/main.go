package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"log"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/extractor"
	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/types"
	"github.com/Panorama-Block/avax/internal/webhook"
	"github.com/Panorama-Block/avax/internal/websocket"
)

func main() {
	cfg := config.LoadConfig()
	
	client := api.NewClient(cfg.APIUrl, cfg.APIKey)
	
	dataAPI := api.NewDataAPI(client)
	metricsAPI := api.NewMetricsAPI(client)
	
	producer := kafka.NewProducer(cfg)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if cfg.WebSocketPort != "" {
		consumer, err := kafka.NewConsumer(cfg.KafkaBroker, "websocket-group", nil)
		if err != nil {
			log.Fatalf("Erro ao criar consumer Kafka: %v", err)
		}
		
		wsServer := websocket.NewServer(cfg.WebSocketPort, consumer)
		go func() {
			if err := wsServer.Start(); err != nil {
				log.Printf("Erro ao iniciar servidor WebSocket: %v", err)
			}
		}()
		defer wsServer.Stop(ctx)
	}
	
	eventChan := make(chan types.WebhookEvent, 1000)
	wh := webhook.NewWebhookServer(eventChan, cfg.WebhookPort)
	go wh.Start()
	
	transactionPipeline := extractor.NewTransactionPipeline(dataAPI, producer, cfg.Workers, 1000)
	if err := transactionPipeline.Start(ctx); err != nil {
		log.Fatalf("Erro ao iniciar pipeline de transações: %v", err)
	}
	go func() {
		for evt := range eventChan {
			if err := transactionPipeline.Submit(evt); err != nil {
				log.Printf("Erro ao enviar evento para pipeline: %v", err)
			}
		}
	}()
	

	rtExtractor := extractor.NewRealTimeExtractor(dataAPI, producer, cfg.ChainWsUrl, "43114", "Avalanche C-Chain")
	if err := rtExtractor.Start(ctx); err != nil {
		log.Printf("Erro ao iniciar extrator em tempo real: %v", err)
	}
	
	// Iniciar pipeline de cadeias
	chainPipeline := extractor.NewChainPipeline(dataAPI, metricsAPI, producer, "mainnet", 30*time.Minute)
	if err := chainPipeline.Start(ctx); err != nil {
		log.Printf("Erro ao iniciar pipeline de cadeias: %v", err)
	}
	
	// Iniciar gerenciador de jobs cron
	cronManager := extractor.NewCronJobManager(dataAPI, metricsAPI, producer, "mainnet")
	cronManager.CreateDefaultJobs()
	if err := cronManager.Start(ctx); err != nil {
		log.Printf("Erro ao iniciar gerenciador de jobs cron: %v", err)
	}
	
	metricsPipeline := extractor.NewMetricsPipeline(metricsAPI, producer, "mainnet", 30*time.Second)
	metricsPipeline.SetChains([]string{"43114"}) 
	if err := metricsPipeline.Start(ctx); err != nil {
		log.Printf("Erro ao iniciar pipeline de métricas: %v", err)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Sinal de interrupção recebido, encerrando...")
	
	// Parar pipelines e serviços
	cancel()
	
	// Aguardar um tempo para que os processos finalizem graciosamente
	time.Sleep(2 * time.Second)
	
	// Fechar o produtor Kafka
	producer.Close()
	
	log.Println("Aplicação encerrada com sucesso")
}
