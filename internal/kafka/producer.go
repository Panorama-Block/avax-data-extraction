package kafka

import (
	"encoding/json"
	"log"

	"github.com/Panorama-Block/avax/internal/config"
	"github.com/Panorama-Block/avax/internal/types"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

type Producer struct {
	Producer          *ckafka.Producer
	TopicChains       string
	TopicBlocks       string
	TopicTransactions string
	TopicLogs         string
	TopicERC20        string
	TopicERC721       string
	TopicERC1155      string
	TopicInternalTx   string
	TopicMetrics      string

	// New specific metrics topics
	TopicActivityMetrics    string
	TopicPerformanceMetrics string
	TopicGasMetrics         string
	TopicCumulativeMetrics  string

	TopicSubnets     string
	TopicBlockchains string
	TopicValidators  string
	TopicDelegators  string
	TopicBridges     string
}

func NewProducer(cfg *config.Config) *Producer {
	p, err := ckafka.NewProducer(&ckafka.ConfigMap{
		"bootstrap.servers": cfg.KafkaBroker,
	})
	if err != nil {
		log.Fatalf("Erro ao conectar Kafka: %v", err)
	}
	return &Producer{
		Producer:          p,
		TopicChains:       cfg.KafkaTopicChains,
		TopicBlocks:       cfg.KafkaTopicBlocks,
		TopicTransactions: cfg.KafkaTopicTransactions,
		TopicLogs:         cfg.KafkaTopicLogs,
		TopicERC20:        cfg.KafkaTopicERC20,
		TopicERC721:       cfg.KafkaTopicERC721,
		TopicERC1155:      cfg.KafkaTopicERC1155,
		TopicInternalTx:   cfg.KafkaTopicInternalTx,
		TopicMetrics:      cfg.KafkaTopicMetrics,

		// Initialize specific metrics topics
		TopicActivityMetrics:    cfg.KafkaTopicActivityMetrics,
		TopicPerformanceMetrics: cfg.KafkaTopicPerformanceMetrics,
		TopicGasMetrics:         cfg.KafkaTopicGasMetrics,
		TopicCumulativeMetrics:  cfg.KafkaTopicCumulativeMetrics,

		TopicSubnets:     cfg.KafkaTopicSubnets,
		TopicBlockchains: cfg.KafkaTopicBlockchains,
		TopicValidators:  cfg.KafkaTopicValidators,
		TopicDelegators:  cfg.KafkaTopicDelegators,
		TopicBridges:     cfg.KafkaTopicBridges,
	}
}

func (p *Producer) sendMessage(topic string, data []byte) error {
	if topic == "" {
		log.Printf("[Kafka] Tópico vazio – mensagem descartada (tipo=%T)", data)
		return nil
	}

	msg := &ckafka.Message{
		TopicPartition: ckafka.TopicPartition{Topic: &topic, Partition: ckafka.PartitionAny},
		Value:          data,
	}
	err := p.Producer.Produce(msg, nil)
	if err != nil {
		log.Printf("Erro ao enviar msg p/ tópico %s: %v", topic, err)
		return err
	} else {
		log.Printf("[Kafka] Publicado no tópico %s: %s", topic, data)
	}
	p.Producer.Flush(1000)
	return nil
}

func (p *Producer) PublishTransactions(txChan <-chan *types.Transaction) {
	for tx := range txChan {
		data, _ := json.Marshal(tx)
		p.sendMessage(p.TopicTransactions, data)
	}
}

func (p *Producer) PublishERC20Transfers(erc20Chan <-chan types.ERC20Transfer) {
	for transfer := range erc20Chan {
		data, _ := json.Marshal(transfer)
		p.sendMessage(p.TopicERC20, data)
	}
}

func (p *Producer) PublishERC721Transfers(erc721Chan <-chan types.ERC721Transfer) {
	for transfer := range erc721Chan {
		data, _ := json.Marshal(transfer)
		p.sendMessage(p.TopicERC721, data)
	}
}

func (p *Producer) PublishERC1155Transfers(erc1155Chan <-chan types.ERC1155Transfer) {
	for transfer := range erc1155Chan {
		data, _ := json.Marshal(transfer)
		p.sendMessage(p.TopicERC1155, data)
	}
}

func (p *Producer) PublishInternalTransactions(internalTxChan <-chan types.InternalTransaction) {
	for tx := range internalTxChan {
		data, _ := json.Marshal(tx)
		p.sendMessage(p.TopicInternalTx, data)
	}
}

func (p *Producer) PublishLogs(logChan <-chan types.Log) {
	for lg := range logChan {
		data, _ := json.Marshal(lg)
		p.sendMessage(p.TopicLogs, data)
	}
}

func (p *Producer) PublishChain(ch *types.Chain) {
	// Marshal only the Chain struct without the type
	data, _ := json.Marshal(ch)
	log.Printf("[Kafka] Publishing chain data to topic %s: %s", p.TopicChains, string(data))
	p.sendMessage(p.TopicChains, data)
}

func (p *Producer) PublishBlock(block types.Block) {
	// Marshal only the Block struct without the type
	data, _ := json.Marshal(block)
	p.sendMessage(p.TopicBlocks, data)
}

func (p *Producer) PublishSingleTx(tx *types.Transaction) {
	// Marshal only the Transaction struct without the type
	data, _ := json.Marshal(tx)
	p.sendMessage(p.TopicTransactions, data)
}

func (p *Producer) PublishMetrics(data []byte) {
	p.sendMessage(p.TopicMetrics, data)
}

func (p *Producer) PublishSubnet(s types.Subnet) {
	// Marshal only the Subnet struct without the type
	data, _ := json.Marshal(s)
	p.sendMessage(p.TopicSubnets, data)
}

func (p *Producer) PublishBlockchain(bc types.Blockchain) {
	// Marshal only the Blockchain struct without the type
	data, _ := json.Marshal(bc)
	p.sendMessage(p.TopicBlockchains, data)
}

func (p *Producer) PublishValidator(val types.Validator) {
	// Marshal only the Validator struct without the type
	data, _ := json.Marshal(val)
	p.sendMessage(p.TopicValidators, data)
}

func (p *Producer) PublishDelegator(del types.Delegator) {
	// Marshal only the Delegator struct without the type
	data, _ := json.Marshal(del)
	p.sendMessage(p.TopicDelegators, data)
}

func (p *Producer) PublishBridgeTx(tx types.TeleporterTx) {
	// Marshal only the TeleporterTx struct without the type
	data, _ := json.Marshal(tx)
	p.sendMessage(p.TopicBridges, data)
}

func (p *Producer) Close() {
	p.Producer.Close()
}

// New methods for specific metric types
func (p *Producer) PublishActivityMetrics(data []byte) {
	p.sendMessage(p.TopicActivityMetrics, data)
}

func (p *Producer) PublishPerformanceMetrics(data []byte) {
	p.sendMessage(p.TopicPerformanceMetrics, data)
}

func (p *Producer) PublishGasMetrics(data []byte) {
	p.sendMessage(p.TopicGasMetrics, data)
}

func (p *Producer) PublishCumulativeMetrics(data []byte) {
	p.sendMessage(p.TopicCumulativeMetrics, data)
}
