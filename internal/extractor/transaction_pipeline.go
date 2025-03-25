package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/kafka"
)

// WebhookEvent represents an event received from a webhook
type WebhookEvent struct {
	WebhookID  string                 `json:"webhookId"`
	EventType  api.EventType          `json:"eventType"`
	MessageID  string                 `json:"messageId"`
	Event      map[string]interface{} `json:"event"`
}

// TransactionPipeline handles processing of transactions from webhook events
type TransactionPipeline struct {
	dataAPI       *api.DataAPI
	kafkaProducer *kafka.Producer

	// Channels
	eventCh       chan WebhookEvent
	stop          chan struct{}

	// Configuration
	workerCount   int
	bufferSize    int
	
	// Statistics
	mutex         sync.Mutex
	processed     int64
	errors        int64
	running       bool
	startTime     time.Time
}

// NewTransactionPipeline creates a new transaction pipeline
func NewTransactionPipeline(
	dataAPI *api.DataAPI,
	kafkaProducer *kafka.Producer,
	workerCount int,
	bufferSize int,
) *TransactionPipeline {
	return &TransactionPipeline{
		dataAPI:       dataAPI,
		kafkaProducer: kafkaProducer,
		eventCh:       make(chan WebhookEvent, bufferSize),
		stop:          make(chan struct{}),
		workerCount:   workerCount,
		bufferSize:    bufferSize,
	}
}

// Start begins the transaction processing pipeline
func (t *TransactionPipeline) Start(ctx context.Context) error {
	t.mutex.Lock()
	if t.running {
		t.mutex.Unlock()
		return fmt.Errorf("transaction pipeline already running")
	}
	t.running = true
	t.startTime = time.Now()
	t.processed = 0
	t.errors = 0
	t.mutex.Unlock()

	log.Printf("Starting transaction processing pipeline with %d workers", t.workerCount)

	// Start worker goroutines
	for i := 0; i < t.workerCount; i++ {
		go t.worker(ctx, i)
	}

	return nil
}

// Stop halts the transaction processing pipeline
func (t *TransactionPipeline) Stop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.running {
		return
	}

	close(t.stop)
	t.running = false
	log.Printf("Stopping transaction processing pipeline")
}

// Submit adds a webhook event to the processing queue
func (t *TransactionPipeline) Submit(event WebhookEvent) error {
	if !t.running {
		return fmt.Errorf("transaction pipeline is not running")
	}

	// Try to add the event to the channel
	select {
	case t.eventCh <- event:
		return nil
	default:
		return fmt.Errorf("event channel is full, consider increasing buffer size")
	}
}

// worker processes webhook events
func (t *TransactionPipeline) worker(ctx context.Context, id int) {
	log.Printf("Transaction worker %d started", id)

	for {
		select {
		case <-t.stop:
			log.Printf("Transaction worker %d stopping", id)
			return
		case event := <-t.eventCh:
			if err := t.processWebhookEvent(ctx, event); err != nil {
				log.Printf("Error processing webhook event: %v", err)
				t.mutex.Lock()
				t.errors++
				t.mutex.Unlock()
			} else {
				t.mutex.Lock()
				t.processed++
				t.mutex.Unlock()
			}
		}
	}
}

// processWebhookEvent processes a single webhook event
func (t *TransactionPipeline) processWebhookEvent(ctx context.Context, event WebhookEvent) error {
	// Log event receipt
	log.Printf("Processing webhook event: %s (type: %s)", event.MessageID, event.EventType)

	// Process based on event type
	switch event.EventType {
	case api.EventTypeAddressActivity:
		return t.processAddressActivity(ctx, event)
	case api.EventTypeERC20Transfer:
		return t.processERC20Transfer(ctx, event)
	case api.EventTypeContractEvent:
		return t.processContractEvent(ctx, event)
	default:
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}
}

// processAddressActivity processes an address activity event
func (t *TransactionPipeline) processAddressActivity(ctx context.Context, event WebhookEvent) error {
	// Extract transaction details from the event
	txData, ok := event.Event["transaction"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("event does not contain transaction data")
	}

	// Extract chain ID
	chainID, ok := txData["chainId"].(string)
	if !ok {
		return fmt.Errorf("transaction data does not contain chainId")
	}

	// Extract transaction hash
	txHash, ok := txData["txHash"].(string)
	if !ok {
		return fmt.Errorf("transaction data does not contain txHash")
	}

	// For address activity events, we might want to fetch the full transaction details
	// from the API to get more information (like logs, internal transactions, etc.)
	txDetail, err := t.dataAPI.GetTransaction(chainID, txHash)
	if err != nil {
		return fmt.Errorf("failed to fetch transaction details: %w", err)
	}

	// Prepare data for Kafka
	transactionData := map[string]interface{}{
		"transaction":      txDetail.Transaction,
		"erc20Transfers":   txDetail.ERC20Transfers,
		"erc721Transfers":  txDetail.ERC721Transfers,
		"erc1155Transfers": txDetail.ERC1155Transfers,
		"internalTxs":      txDetail.InternalTransactions,
		"source":           "webhook",
		"webhookId":        event.WebhookID,
		"messageId":        event.MessageID,
		"eventType":        event.EventType,
		"processingTime":   time.Now().Unix(),
	}

	// Convert to JSON
	txJSON, err := json.Marshal(transactionData)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction data: %w", err)
	}

	// Publish to Kafka
	if err := t.kafkaProducer.PublishTransaction(chainID, txHash, txJSON); err != nil {
		return fmt.Errorf("failed to publish transaction to Kafka: %w", err)
	}

	// Process token transfers separately if they exist
	if len(txDetail.ERC20Transfers) > 0 {
		for _, transfer := range txDetail.ERC20Transfers {
			transferData := map[string]interface{}{
				"transfer":       transfer,
				"txHash":         txHash,
				"source":         "webhook",
				"processingTime": time.Now().Unix(),
			}

			transferJSON, err := json.Marshal(transferData)
			if err != nil {
				log.Printf("Error marshaling ERC20 transfer: %v", err)
				continue
			}

			if err := t.kafkaProducer.PublishERC20Transfer(chainID, transfer.ERC20Token.Address, transferJSON); err != nil {
				log.Printf("Error publishing ERC20 transfer: %v", err)
			}
		}
	}

	// Process NFT transfers
	if len(txDetail.ERC721Transfers) > 0 {
		for _, transfer := range txDetail.ERC721Transfers {
			transferData := map[string]interface{}{
				"transfer":       transfer,
				"txHash":         txHash,
				"source":         "webhook",
				"processingTime": time.Now().Unix(),
			}

			transferJSON, err := json.Marshal(transferData)
			if err != nil {
				log.Printf("Error marshaling ERC721 transfer: %v", err)
				continue
			}

			if err := t.kafkaProducer.PublishERC721Transfer(chainID, transfer.ERC721Token.Address, transferJSON); err != nil {
				log.Printf("Error publishing ERC721 transfer: %v", err)
			}
		}
	}

	// Check if any of the addresses involved are "whales" (large holders)
	// This would typically involve checking against a list of known whale addresses
	// or checking balance thresholds
	if err := t.checkForWhaleActivity(ctx, chainID, txDetail); err != nil {
		log.Printf("Error checking for whale activity: %v", err)
		// Continue processing even if whale check fails
	}

	return nil
}

// processERC20Transfer processes an ERC20 transfer event
func (t *TransactionPipeline) processERC20Transfer(ctx context.Context, event WebhookEvent) error {
	// Extract transfer details from the event
	transferData, ok := event.Event["transfer"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("event does not contain transfer data")
	}

	// Extract chain ID
	chainID, ok := transferData["chainId"].(string)
	if !ok {
		return fmt.Errorf("transfer data does not contain chainId")
	}

	// Extract token address
	tokenData, ok := transferData["erc20Token"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("transfer data does not contain erc20Token")
	}

	tokenAddress, ok := tokenData["address"].(string)
	if !ok {
		return fmt.Errorf("token data does not contain address")
	}

	// Prepare data for Kafka
	enrichedTransferData := map[string]interface{}{
		"transfer":       transferData,
		"source":         "webhook",
		"webhookId":      event.WebhookID,
		"messageId":      event.MessageID,
		"eventType":      event.EventType,
		"processingTime": time.Now().Unix(),
	}

	// Convert to JSON
	transferJSON, err := json.Marshal(enrichedTransferData)
	if err != nil {
		return fmt.Errorf("failed to marshal transfer data: %w", err)
	}

	// Publish to Kafka
	if err := t.kafkaProducer.PublishERC20Transfer(chainID, tokenAddress, transferJSON); err != nil {
		return fmt.Errorf("failed to publish ERC20 transfer to Kafka: %w", err)
	}

	return nil
}

// processContractEvent processes a contract event
func (t *TransactionPipeline) processContractEvent(ctx context.Context, event WebhookEvent) error {
	// Extract event details
	contractEvent, ok := event.Event["contractEvent"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("event does not contain contractEvent data")
	}

	// Extract chain ID
	chainID, ok := contractEvent["chainId"].(string)
	if !ok {
		return fmt.Errorf("contractEvent data does not contain chainId")
	}

	// Extract contract address
	contractAddress, ok := contractEvent["address"].(string)
	if !ok {
		return fmt.Errorf("contractEvent data does not contain contract address")
	}

	// Prepare data for Kafka
	contractEventData := map[string]interface{}{
		"contractEvent":  contractEvent,
		"source":         "webhook",
		"webhookId":      event.WebhookID,
		"messageId":      event.MessageID,
		"eventType":      event.EventType,
		"processingTime": time.Now().Unix(),
	}

	// Convert to JSON
	eventJSON, err := json.Marshal(contractEventData)
	if err != nil {
		return fmt.Errorf("failed to marshal contract event data: %w", err)
	}

	// Publish to Kafka
	if err := t.kafkaProducer.PublishContractEvent(chainID, contractAddress, eventJSON); err != nil {
		return fmt.Errorf("failed to publish contract event to Kafka: %w", err)
	}

	return nil
}

// checkForWhaleActivity checks if any addresses involved in a transaction are whales
func (t *TransactionPipeline) checkForWhaleActivity(ctx context.Context, chainID string, txDetail *api.TransactionDetail) error {
	// This is where you would implement your whale detection logic
	// For example, check if any of the addresses involved have a balance above a certain threshold
	// or are in a predetermined list of whale addresses

	// For demonstration purposes, let's assume we're tracking addresses with large AVAX holdings

	// Extract addresses from the transaction
	addresses := make([]string, 0)
	
	// Add sender
	if txDetail.Transaction.From.Address != "" {
		addresses = append(addresses, txDetail.Transaction.From.Address)
	}
	
	// Add recipient
	if txDetail.Transaction.To.Address != "" {
		addresses = append(addresses, txDetail.Transaction.To.Address)
	}
	
	// Check for large value transfers (e.g., > 1000 AVAX)
	// Convert from string to numeric value would be needed in a real implementation
	// This is just a placeholder for illustration
	valueAVAX := 0.0 // Would parse txDetail.Transaction.Value to AVAX
	
	if valueAVAX > 1000.0 {
		// This is a large transfer, flag as potential whale activity
		whaleAlert := map[string]interface{}{
			"txHash":         txDetail.Transaction.Hash,
			"chainId":        chainID,
			"from":           txDetail.Transaction.From.Address,
			"to":             txDetail.Transaction.To.Address,
			"value":          txDetail.Transaction.Value,
			"blockNumber":    txDetail.Transaction.BlockNumber,
			"blockTimestamp": txDetail.Transaction.BlockTimestamp,
			"alertType":      "large_transfer",
			"alertTime":      time.Now().Unix(),
		}
		
		whaleAlertJSON, err := json.Marshal(whaleAlert)
		if err != nil {
			return fmt.Errorf("failed to marshal whale alert: %w", err)
		}
		
		if err := t.kafkaProducer.PublishWhaleAlert(chainID, txDetail.Transaction.Hash, whaleAlertJSON); err != nil {
			return fmt.Errorf("failed to publish whale alert to Kafka: %w", err)
		}
	}
	
	return nil
}

// Status returns the current status of the pipeline
func (t *TransactionPipeline) Status() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	uptime := time.Since(t.startTime).String()
	if !t.running {
		uptime = "0s"
	}

	return map[string]interface{}{
		"running":        t.running,
		"workerCount":    t.workerCount,
		"bufferSize":     t.bufferSize,
		"currentBuffer":  len(t.eventCh),
		"processed":      t.processed,
		"errors":         t.errors,
		"uptime":         uptime,
		"startTime":      t.startTime.Format(time.RFC3339),
	}
}