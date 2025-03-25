package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
	
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/Panorama-Block/avax/internal/api"
	"github.com/Panorama-Block/avax/internal/kafka"
)

// RealTimeExtractor handles real-time extraction of blocks and transactions
type RealTimeExtractor struct {
	dataAPI       *api.DataAPI
	kafkaProducer *kafka.Producer
	wsClient      *ethclient.Client
	wsEndpoint    string
	chainID       string
	chainName     string
	
	// Control channels
	stop          chan struct{}
	blocksCh      chan *types.Header
	errorCh       chan error
	
	// Configuration
	backfillMissedBlocks bool
	processTxs          bool
	processLogs         bool
	processTokenTransfers bool
	
	// Running state
	mutex              sync.Mutex
	lastProcessedBlock uint64
	running            bool
}

// NewRealTimeExtractor creates a new real-time extractor
func NewRealTimeExtractor(
	dataAPI *api.DataAPI,
	kafkaProducer *kafka.Producer,
	wsEndpoint string,
	chainID string,
	chainName string,
) *RealTimeExtractor {
	return &RealTimeExtractor{
		dataAPI:       dataAPI,
		kafkaProducer: kafkaProducer,
		wsEndpoint:    wsEndpoint,
		chainID:       chainID,
		chainName:     chainName,
		
		stop:          make(chan struct{}),
		blocksCh:      make(chan *types.Header, 100),
		errorCh:       make(chan error, 10),
		
		backfillMissedBlocks: true,
		processTxs:          true,
		processLogs:         true,
		processTokenTransfers: true,
	}
}

// Start begins the real-time extraction process
func (r *RealTimeExtractor) Start(ctx context.Context) error {
	r.mutex.Lock()
	if r.running {
		r.mutex.Unlock()
		return fmt.Errorf("extractor already running")
	}
	r.running = true
	r.mutex.Unlock()
	
	log.Printf("Starting real-time extractor for chain %s (%s)", r.chainName, r.chainID)
	
	// Create WebSocket connection to node
	var err error
	r.wsClient, err = ethclient.Dial(r.wsEndpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	
	// Get the current block number to determine where to start
	latestBlock, err := r.wsClient.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	
	log.Printf("Latest block number for %s: %d", r.chainName, latestBlock)
	r.lastProcessedBlock = latestBlock
	
	// Start subscription for new headers
	headers := make(chan *types.Header)
	sub, err := r.wsClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		return fmt.Errorf("failed to subscribe to new headers: %w", err)
	}
	
	// Start goroutines for processing
	go r.handleBlockHeaders(ctx, headers, sub)
	go r.processBlocks(ctx)
	
	return nil
}

// Stop halts the real-time extraction process
func (r *RealTimeExtractor) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if !r.running {
		return
	}
	
	close(r.stop)
	r.running = false
	log.Printf("Stopping real-time extractor for chain %s", r.chainName)
}

// handleBlockHeaders processes incoming block headers from the subscription
func (r *RealTimeExtractor) handleBlockHeaders(ctx context.Context, headers <-chan *types.Header, sub ethereum.Subscription) {
	defer sub.Unsubscribe()
	
	for {
		select {
		case <-r.stop:
			return
		case err := <-sub.Err():
			log.Printf("WebSocket subscription error for %s: %v", r.chainName, err)
			r.errorCh <- err
			// Try to reconnect after a short delay
			time.Sleep(5 * time.Second)
			headers := make(chan *types.Header)
			sub, err = r.wsClient.SubscribeNewHead(ctx, headers)
			if err != nil {
				log.Printf("Failed to resubscribe to headers for %s: %v", r.chainName, err)
				r.errorCh <- err
				return
			}
			go r.handleBlockHeaders(ctx, headers, sub)
			return
		case header := <-headers:
			select {
			case r.blocksCh <- header:
				log.Printf("Received new block header: %d", header.Number.Uint64())
			default:
				log.Printf("Block channel full, dropping block %d", header.Number.Uint64())
			}
		}
	}
}

// processBlocks handles the processing of incoming blocks
func (r *RealTimeExtractor) processBlocks(ctx context.Context) {
	for {
		select {
		case <-r.stop:
			return
		case header := <-r.blocksCh:
			blockNum := header.Number.Uint64()
			
			// Check if we need to backfill any missed blocks
			if r.backfillMissedBlocks && blockNum > r.lastProcessedBlock+1 {
				log.Printf("Detected missed blocks from %d to %d, backfilling...", r.lastProcessedBlock+1, blockNum-1)
				for i := r.lastProcessedBlock + 1; i < blockNum; i++ {
					if err := r.processBlockByNumber(ctx, i); err != nil {
						log.Printf("Error processing missed block %d: %v", i, err)
						// Continue with next block anyway
					}
				}
			}
			
			// Process the current block
			if err := r.processBlockByHash(ctx, header.Hash()); err != nil {
				log.Printf("Error processing block %d: %v", blockNum, err)
			} else {
				r.mutex.Lock()
				r.lastProcessedBlock = blockNum
				r.mutex.Unlock()
			}
		}
	}
}

// processBlockByNumber processes a specific block by number
func (r *RealTimeExtractor) processBlockByNumber(ctx context.Context, blockNum uint64) error {
	// First, get the block from the node
	block, err := r.wsClient.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNum, err)
	}
	
	return r.processBlock(ctx, block)
}

// processBlockByHash processes a specific block by hash
func (r *RealTimeExtractor) processBlockByHash(ctx context.Context, hash common.Hash) error {
	// First, get the block from the node
	block, err := r.wsClient.BlockByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("failed to get block by hash %s: %w", hash.Hex(), err)
	}
	
	return r.processBlock(ctx, block)
}

// processBlock processes a block and its transactions
func (r *RealTimeExtractor) processBlock(ctx context.Context, block *types.Block) error {
	blockNum := block.NumberU64()
	log.Printf("Processing block %d with %d transactions", blockNum, len(block.Transactions()))
	
	// Fetch block details from AvaCloud Data API for enriched information
	blockDetail, err := r.dataAPI.GetBlock(r.chainID, block.Number().String())
	if err != nil {
		// We'll continue with the direct node data even if the API call fails
		log.Printf("Warning: Failed to get block details from API: %v", err)
	}
	
	// Prepare Kafka message for block
	blockData := map[string]interface{}{
		"blockNumber":       blockNum,
		"blockHash":         block.Hash().Hex(),
		"parentHash":        block.ParentHash().Hex(),
		"timestamp":         block.Time(),
		"gasUsed":           block.GasUsed(),
		"gasLimit":          block.GasLimit(),
		"txCount":           len(block.Transactions()),
		"chainId":           r.chainID,
		"chainName":         r.chainName,
		"processingTime":    time.Now().Unix(),
	}
	
	// Enrich with AvaCloud data if available
	if blockDetail != nil {
		// Add any additional fields available from the API
		blockData["feesSpent"] = blockDetail.FeesSpent
		blockData["baseFee"] = blockDetail.BaseFee
		blockData["cumulativeTransactions"] = blockDetail.CumulativeTransactions
	}
	
	// Send block data to Kafka
	blockDataJSON, err := json.Marshal(blockData)
	if err != nil {
		return fmt.Errorf("failed to marshal block data: %w", err)
	}
	
	if err := r.kafkaProducer.PublishBlock(r.chainID, blockNum, blockDataJSON); err != nil {
		log.Printf("Failed to publish block data to Kafka: %v", err)
		// Continue processing anyway
	}
	
	// Process transactions if enabled
	if r.processTxs {
		for _, tx := range block.Transactions() {
			if err := r.processTransaction(ctx, tx, block); err != nil {
				log.Printf("Error processing transaction %s: %v", tx.Hash().Hex(), err)
				// Continue with next transaction
			}
		}
	}
	
	return nil
}

// processTransaction processes a single transaction
func (r *RealTimeExtractor) processTransaction(ctx context.Context, tx *types.Transaction, block *types.Block) error {
	txHash := tx.Hash().Hex()
	blockNum := block.NumberU64()
	
	// Fetch transaction details from AvaCloud Data API for enriched information
	txDetail, err := r.dataAPI.GetTransaction(r.chainID, txHash)
	if err != nil {
		// We'll continue with the direct node data even if the API call fails
		log.Printf("Warning: Failed to get transaction details from API: %v", err)
	}
	
	// Basic transaction data from the node
	// Instead of using tx.AsMessage, extract information directly
	var fromAddress string
	var toAddress string
	
	// Get sender address if available
	signer := types.NewEIP155Signer(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err == nil {
		fromAddress = from.Hex()
	}
	
	// Get recipient address if available
	if tx.To() != nil {
		toAddress = tx.To().Hex()
	}
	
	// Prepare Kafka message for transaction
	txData := map[string]interface{}{
		"hash":         txHash,
		"blockNumber":  blockNum,
		"blockHash":    block.Hash().Hex(),
		"timestamp":    block.Time(),
		"from":         fromAddress,
		"to":           toAddress,
		"value":        tx.Value().String(),
		"gasPrice":     tx.GasPrice().String(),
		"gas":          tx.Gas(),
		"nonce":        tx.Nonce(),
		"chainId":      r.chainID,
		"chainName":    r.chainName,
		"processingTime": time.Now().Unix(),
	}
	
	// Enrich with AvaCloud data if available
	if txDetail != nil {
		// Add gas used, status, and other enriched data
		txData["gasUsed"] = txDetail.Transaction.GasUsed
		txData["status"] = txDetail.Transaction.TxStatus
		
		// Add token transfers if available and enabled
		if r.processTokenTransfers {
			if len(txDetail.ERC20Transfers) > 0 {
				txData["erc20Transfers"] = txDetail.ERC20Transfers
			}
			if len(txDetail.ERC721Transfers) > 0 {
				txData["erc721Transfers"] = txDetail.ERC721Transfers
			}
			if len(txDetail.ERC1155Transfers) > 0 {
				txData["erc1155Transfers"] = txDetail.ERC1155Transfers
			}
		}
		
		// Add internal transactions if available and enabled
		if r.processLogs && len(txDetail.InternalTransactions) > 0 {
			txData["internalTransactions"] = txDetail.InternalTransactions
		}
	}
	
	// Send transaction data to Kafka
	txDataJSON, err := json.Marshal(txData)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction data: %w", err)
	}
	
	if err := r.kafkaProducer.PublishTransaction(r.chainID, txHash, txDataJSON); err != nil {
		return fmt.Errorf("failed to publish transaction data to Kafka: %w", err)
	}
	
	// If token transfers are enabled and we received enriched data
	// We might want to publish them separately
	if r.processTokenTransfers && txDetail != nil {
		// Publish ERC20 transfers to their own topic if available
		for _, transfer := range txDetail.ERC20Transfers {
			transferJSON, err := json.Marshal(transfer)
			if err != nil {
				log.Printf("Error marshaling ERC20 transfer: %v", err)
				continue
			}
			
			if err := r.kafkaProducer.PublishERC20Transfer(r.chainID, transfer.ERC20Token.Address, transferJSON); err != nil {
				log.Printf("Error publishing ERC20 transfer: %v", err)
			}
		}
		
		// Similar processing could be done for ERC721 and ERC1155 transfers
	}
	
	return nil
}

// Status returns the current status of the extractor
func (r *RealTimeExtractor) Status() map[string]interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	return map[string]interface{}{
		"chainId":            r.chainID,
		"chainName":          r.chainName,
		"running":            r.running,
		"lastProcessedBlock": r.lastProcessedBlock,
		"blockChannelLength": len(r.blocksCh),
		"errorChannelLength": len(r.errorCh),
	}
}