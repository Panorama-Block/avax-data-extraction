package extractor

import (
	"log"
	"sync"

	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/types"
)

// StartPipeline inicializa o pipeline para processar eventos do webhook
func StartTxPipeline(producer *kafka.Producer, eventChan chan types.WebhookEvent, numWorkers int) {
	// Canal para enviar eventos ao Kafka
	txChan := make(chan *types.Transaction, numWorkers)
	erc20Chan := make(chan types.ERC20Transfer, numWorkers)
	logChan := make(chan types.Log, numWorkers)

	// Etapa 1: Processar os eventos recebidos
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processWebhookEvents(eventChan, txChan, erc20Chan, logChan, &wg)
	}

	// Etapa 2: Publicar no Kafka
	go publishTransactions(producer, txChan)
	go publishERC20Transfers(producer, erc20Chan)
	go publishLogs(producer, logChan)

	// Aguarda finalização dos workers
	wg.Wait()
	close(txChan)
	close(erc20Chan)
	close(logChan)
}

// processWebhookEvents processa os eventos do webhook e envia para os canais corretos
func processWebhookEvents(eventChan <-chan types.WebhookEvent, txChan chan<- *types.Transaction, erc20Chan chan<- types.ERC20Transfer, logChan chan<- types.Log, wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range eventChan {
		tx := event.Event.Transaction
		log.Printf("Processando transação %s", tx.TxHash)

		// Enviar transação para o Kafka
		txChan <- &tx

		// Enviar transferências ERC20
		for _, transfer := range tx.ERC20Transfers {
			erc20Chan <- transfer
		}

		// Enviar logs de eventos
		for _, logData := range event.Event.Logs {
			logChan <- logData
		}
	}
}

// publishTransactions publica transações completas no Kafka
func publishTransactions(producer *kafka.Producer, txChan <-chan *types.Transaction) {
	for tx := range txChan {
		producer.SendTransaction(tx)
	}
}

// publishERC20Transfers publica transferências ERC20 no Kafka
func publishERC20Transfers(producer *kafka.Producer, erc20Chan <-chan types.ERC20Transfer) {
	for transfer := range erc20Chan {
		producer.SendERC20Transfer(transfer)
	}
}

// publishLogs publica logs de eventos no Kafka
func publishLogs(producer *kafka.Producer, logChan <-chan types.Log) {
	for logData := range logChan {
		producer.SendLog(logData)
	}
}
