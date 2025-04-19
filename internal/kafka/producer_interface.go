package kafka

import (
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// KafkaProducer defines the interface for Kafka producers
type KafkaProducer interface {
	PublishTransactions(txChan <-chan *types.Transaction)
	PublishERC20Transfers(erc20Chan <-chan types.ERC20Transfer)
	PublishERC721Transfers(erc721Chan <-chan types.ERC721Transfer)
	PublishERC1155Transfers(erc1155Chan <-chan types.ERC1155Transfer)
	PublishInternalTransactions(internalTxChan <-chan types.InternalTransaction)
	PublishLogs(logChan <-chan types.Log)
	PublishChain(ch *types.Chain)
	PublishBlock(block types.Block)
	PublishSingleTx(tx *types.Transaction)
	PublishMetrics(data []byte)
	PublishSubnet(s types.Subnet)
	PublishBlockchain(bc types.Blockchain)
	PublishValidator(val types.Validator)
	PublishDelegator(del types.Delegator)
	PublishBridgeTx(tx types.TeleporterTx)
	PublishActivityMetrics(data []byte)
	PublishPerformanceMetrics(data []byte)
	PublishGasMetrics(data []byte)
	PublishCumulativeMetrics(data []byte)
	Close()
	sendMessage(topic string, data []byte) error
}

// MockProducer implements the KafkaProducer interface for testing/mock mode
type MockProducer struct {
	TopicChains          string
	TopicBlocks          string
	TopicTransactions    string
	TopicLogs            string
	TopicERC20           string
	TopicERC721          string
	TopicERC1155         string
	TopicInternalTx      string
	TopicMetrics         string
	TopicActivityMetrics string
	TopicPerformanceMetrics string
	TopicGasMetrics      string
	TopicCumulativeMetrics string
	TopicSubnets         string
	TopicBlockchains     string
	TopicValidators      string
	TopicDelegators      string
	TopicBridges         string
}

// NewMockProducer creates a new mock producer
func NewMockProducer() *MockProducer {
	return &MockProducer{
		TopicChains:           "mock-chains",
		TopicBlocks:           "mock-blocks",
		TopicTransactions:     "mock-transactions",
		TopicLogs:             "mock-logs",
		TopicERC20:            "mock-erc20",
		TopicERC721:           "mock-erc721",
		TopicERC1155:          "mock-erc1155",
		TopicInternalTx:       "mock-internal-tx",
		TopicMetrics:          "mock-metrics",
		TopicActivityMetrics:  "mock-activity-metrics",
		TopicPerformanceMetrics: "mock-performance-metrics",
		TopicGasMetrics:       "mock-gas-metrics",
		TopicCumulativeMetrics: "mock-cumulative-metrics",
		TopicSubnets:          "mock-subnets",
		TopicBlockchains:      "mock-blockchains",
		TopicValidators:       "mock-validators",
		TopicDelegators:       "mock-delegators",
		TopicBridges:          "mock-bridges",
	}
}

// Mock implementation of all the KafkaProducer interface methods
func (p *MockProducer) PublishTransactions(txChan <-chan *types.Transaction) {
	for range txChan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishERC20Transfers(erc20Chan <-chan types.ERC20Transfer) {
	for range erc20Chan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishERC721Transfers(erc721Chan <-chan types.ERC721Transfer) {
	for range erc721Chan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishERC1155Transfers(erc1155Chan <-chan types.ERC1155Transfer) {
	for range erc1155Chan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishInternalTransactions(internalTxChan <-chan types.InternalTransaction) {
	for range internalTxChan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishLogs(logChan <-chan types.Log) {
	for range logChan {
		// Just consume the channel without doing anything
	}
}

func (p *MockProducer) PublishChain(ch *types.Chain) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishBlock(block types.Block) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishSingleTx(tx *types.Transaction) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishMetrics(data []byte) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishSubnet(s types.Subnet) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishBlockchain(bc types.Blockchain) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishValidator(val types.Validator) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishDelegator(del types.Delegator) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishBridgeTx(tx types.TeleporterTx) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishActivityMetrics(data []byte) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishPerformanceMetrics(data []byte) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishGasMetrics(data []byte) {
	// Mock implementation does nothing
}

func (p *MockProducer) PublishCumulativeMetrics(data []byte) {
	// Mock implementation does nothing
}

func (p *MockProducer) Close() {
	// Mock implementation does nothing
}

// sendMessage mocks sending a message to Kafka
func (p *MockProducer) sendMessage(topic string, data []byte) error {
	if topic == "" {
		log.Printf("[MockKafka] Tópico vazio – mensagem descartada")
		return nil
	}
	
	log.Printf("[MockKafka] Simulando publicação no tópico %s: %s", topic, data)
	return nil
} 