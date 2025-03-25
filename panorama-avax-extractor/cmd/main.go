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
	// Carregar configuração
	cfg := config.LoadConfig()
	
	// Criar cliente API
	client := api.NewClient(cfg.APIUrl, cfg.APIKey)
	
	// Inicializar APIs específicas
	dataAPI := api.NewDataAPI(client)
	metricsAPI := api.NewMetricsAPI(client)
	
	// Criar produtor Kafka
	producer := kafka.NewProducer(cfg)
	
	// Criar contexto que pode ser cancelado
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Iniciar servidor WebSocket (opcional, se configurado)
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
	
	// Canal para eventos do webhook
	eventChan := make(chan types.WebhookEvent, 100)
	
	// Iniciar servidor webhook
	wh := webhook.NewWebhookServer(eventChan, cfg.WebhookPort)
	go wh.Start()
	
	// Iniciar pipeline de processamento de transações
	transactionPipeline := extractor.NewTransactionPipeline(dataAPI, producer, cfg.Workers, 100)
	if err := transactionPipeline.Start(ctx); err != nil {
		log.Fatalf("Erro ao iniciar pipeline de transações: %v", err)
	}
	
	// Iniciar worker para processar eventos do webhook
	go func() {
		for evt := range eventChan {
			if err := transactionPipeline.Submit(evt); err != nil {
				log.Printf("Erro ao enviar evento para pipeline: %v", err)
			}
		}
	}()
	
	// Iniciar extração em tempo real de blocos via WebSocket
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
	
	// Iniciar pipeline de métricas
	metricsPipeline := extractor.NewMetricsPipeline(metricsAPI, producer, "mainnet", 30*time.Second)
	metricsPipeline.SetChains([]string{"43114"}) // Avalanche C-Chain
	if err := metricsPipeline.Start(ctx); err != nil {
		log.Printf("Erro ao iniciar pipeline de métricas: %v", err)
	}
	
	// Capturar sinais de interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Aguardar sinal de término
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
