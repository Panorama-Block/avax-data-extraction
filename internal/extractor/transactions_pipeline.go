package extractor

import (
	"log"
	"sync"

	"github.com/Panorama-Block/avax/internal/kafka"
	"github.com/Panorama-Block/avax/internal/types"
)

// StartTxPipeline processes Webhook events and sends them to Kafka
func StartTxPipeline(producer *kafka.Producer, eventChan chan types.WebhookEvent, numWorkers int) {
	txChan := make(chan *types.Transaction, numWorkers)
	erc20Chan := make(chan types.ERC20Transfer, numWorkers)
	logChan := make(chan types.Log, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processWebhookEvents(eventChan, txChan, erc20Chan, logChan, &wg)
	}

	go producer.PublishTransactions(txChan)
	go producer.PublishERC20Transfers(erc20Chan)
	go producer.PublishLogs(logChan)

	wg.Wait()
	close(txChan)
	close(erc20Chan)
	close(logChan)
}

func processWebhookEvents(eventChan <-chan types.WebhookEvent, txChan chan<- *types.Transaction, erc20Chan chan<- types.ERC20Transfer, logChan chan<- types.Log, wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range eventChan {
		tx := event.Event.Transaction
		log.Printf("Processando transação %s", tx.TxHash)

		txChan <- &tx

		for _, transfer := range tx.ERC20Transfers {
			erc20Chan <- transfer
		}

		for _, logData := range event.Event.Logs {
			logChan <- logData
		}
	}
}
