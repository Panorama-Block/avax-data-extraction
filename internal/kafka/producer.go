package kafka

import (
	"fmt"
	"log"
	"sync"
	"time"
	
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Producer handles publishing data to Kafka topics
type Producer struct {
	producer    *kafka.Producer
	
	// Topic configuration
	topicPrefix string
	
	// Stats
	mutex               sync.Mutex
	messagesPublished   int64
	lastPublishTime     time.Time
}

// TopicConfig defines the topic configuration for different data types
type TopicConfig struct {
	BlockTopic             string
	TransactionTopic       string
	ERC20TransferTopic     string
	ERC721TransferTopic    string
	ContractEventTopic     string
	ValidatorTopic         string
	DelegatorTopic         string
	StakingMetricTopic     string
	PendingRewardTopic     string
	SubnetTopic            string
	BlockchainTopic        string
	AssetTopic             string
	ChainInfoTopic         string
	MetricTopic            string
	TeleporterMetricTopic  string
	WealthDistributionTopic string
	BtcHoldersTopic        string
	ValidatorAddressesTopic string
	WhaleAlertTopic        string
}

// NewProducer creates a new Kafka producer
func NewProducer(
	bootstrapServers string,
	topicPrefix string,
	config *kafka.ConfigMap,
) (*Producer, error) {
	// Add bootstrap servers to config
	if config == nil {
		config = &kafka.ConfigMap{}
	}
	
	err := config.SetKey("bootstrap.servers", bootstrapServers)
	if err != nil {
		return nil, fmt.Errorf("failed to set bootstrap servers: %w", err)
	}
	
	// Create producer
	producer, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	
	// Start a goroutine to handle delivery reports
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message to topic %s: %v", 
						*ev.TopicPartition.Topic, ev.TopicPartition.Error)
				}
			}
		}
	}()
	
	return &Producer{
		producer:    producer,
		topicPrefix: topicPrefix,
	}, nil
}

// Close closes the Kafka producer
func (p *Producer) Close() {
	p.producer.Close()
}

// PublishBlock publishes block data to Kafka
func (p *Producer) PublishBlock(chainID string, blockNumber uint64, data []byte) error {
	topic := fmt.Sprintf("%s.blocks.%s", p.topicPrefix, chainID)
	key := fmt.Sprintf("%d", blockNumber)
	
	return p.publish(topic, key, data)
}

// PublishTransaction publishes transaction data to Kafka
func (p *Producer) PublishTransaction(chainID, txHash string, data []byte) error {
	topic := fmt.Sprintf("%s.transactions.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, txHash, data)
}

// PublishERC20Transfer publishes ERC20 transfer data to Kafka
func (p *Producer) PublishERC20Transfer(chainID, tokenAddress string, data []byte) error {
	topic := fmt.Sprintf("%s.erc20transfers.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, tokenAddress, data)
}

// PublishERC721Transfer publishes ERC721 transfer data to Kafka
func (p *Producer) PublishERC721Transfer(chainID, tokenAddress string, data []byte) error {
	topic := fmt.Sprintf("%s.erc721transfers.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, tokenAddress, data)
}

// PublishContractEvent publishes contract event data to Kafka
func (p *Producer) PublishContractEvent(chainID, contractAddress string, data []byte) error {
	topic := fmt.Sprintf("%s.contractevents.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, contractAddress, data)
}

// PublishValidator publishes validator data to Kafka
func (p *Producer) PublishValidator(network, nodeID string, data []byte) error {
	topic := fmt.Sprintf("%s.validators.%s", p.topicPrefix, network)
	
	return p.publish(topic, nodeID, data)
}

// PublishDelegator publishes delegator data to Kafka
func (p *Producer) PublishDelegator(network, delegatorAddress string, data []byte) error {
	topic := fmt.Sprintf("%s.delegators.%s", p.topicPrefix, network)
	
	return p.publish(topic, delegatorAddress, data)
}

// PublishStakingMetric publishes staking metric data to Kafka
func (p *Producer) PublishStakingMetric(networkOrSubnet, metricName string, data []byte) error {
	topic := fmt.Sprintf("%s.stakingmetrics.%s", p.topicPrefix, networkOrSubnet)
	
	return p.publish(topic, metricName, data)
}

// PublishPendingReward publishes pending reward data to Kafka
func (p *Producer) PublishPendingReward(network, rewardOwner string, data []byte) error {
	topic := fmt.Sprintf("%s.pendingrewards.%s", p.topicPrefix, network)
	
	return p.publish(topic, rewardOwner, data)
}

// PublishSubnet publishes subnet data to Kafka
func (p *Producer) PublishSubnet(network, subnetID string, data []byte) error {
	topic := fmt.Sprintf("%s.subnets.%s", p.topicPrefix, network)
	
	return p.publish(topic, subnetID, data)
}

// PublishBlockchain publishes blockchain data to Kafka
func (p *Producer) PublishBlockchain(network, blockchainID string, data []byte) error {
	topic := fmt.Sprintf("%s.blockchains.%s", p.topicPrefix, network)
	
	return p.publish(topic, blockchainID, data)
}

// PublishAsset publishes asset data to Kafka
func (p *Producer) PublishAsset(network, chainID, assetID string, data []byte) error {
	topic := fmt.Sprintf("%s.assets.%s.%s", p.topicPrefix, network, chainID)
	
	return p.publish(topic, assetID, data)
}

// PublishChainInfo publishes chain info data to Kafka
func (p *Producer) PublishChainInfo(chainID string, data []byte) error {
	topic := fmt.Sprintf("%s.chaininfo", p.topicPrefix)
	
	return p.publish(topic, chainID, data)
}

// PublishMetric publishes metric data to Kafka
func (p *Producer) PublishMetric(chainID, metricName string, data []byte) error {
	topic := fmt.Sprintf("%s.metrics.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, metricName, data)
}

// PublishTeleporterMetric publishes teleporter metric data to Kafka
func (p *Producer) PublishTeleporterMetric(metricName string, data []byte) error {
	topic := fmt.Sprintf("%s.teleportermetrics", p.topicPrefix)
	
	return p.publish(topic, metricName, data)
}

// PublishWealthDistribution publishes wealth distribution data to Kafka
func (p *Producer) PublishWealthDistribution(chainID, threshold string, data []byte) error {
	topic := fmt.Sprintf("%s.wealthdistribution.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, threshold, data)
}

// PublishBtcHolders publishes BTC.b holders data to Kafka
func (p *Producer) PublishBtcHolders(threshold string, data []byte) error {
	topic := fmt.Sprintf("%s.btcholders", p.topicPrefix)
	
	return p.publish(topic, threshold, data)
}

// PublishValidatorAddresses publishes validator addresses data to Kafka
func (p *Producer) PublishValidatorAddresses(network string, data []byte) error {
	topic := fmt.Sprintf("%s.validatoraddresses.%s", p.topicPrefix, network)
	
	return p.publish(topic, "validators", data)
}

// PublishWhaleAlert publishes whale alert data to Kafka
func (p *Producer) PublishWhaleAlert(chainID, txHash string, data []byte) error {
	topic := fmt.Sprintf("%s.whalealerts.%s", p.topicPrefix, chainID)
	
	return p.publish(topic, txHash, data)
}

// publish is a generic method to publish data to a Kafka topic
func (p *Producer) publish(topic, key string, value []byte) error {
	// Create message
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: value,
		Timestamp: time.Now(),
	}
	
	// Set key if provided
	if key != "" {
		message.Key = []byte(key)
	}
	
	// Produce message
	if err := p.producer.Produce(message, nil); err != nil {
		return fmt.Errorf("failed to produce message to topic %s: %w", topic, err)
	}
	
	// Update stats
	p.mutex.Lock()
	p.messagesPublished++
	p.lastPublishTime = time.Now()
	p.mutex.Unlock()
	
	return nil
}

// Status returns the current status of the producer
func (p *Producer) Status() map[string]interface{} {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	lastPublishStr := "Never"
	if !p.lastPublishTime.IsZero() {
		lastPublishStr = p.lastPublishTime.Format(time.RFC3339)
	}
	
	return map[string]interface{}{
		"messagesPublished": p.messagesPublished,
		"lastPublishTime":   lastPublishStr,
		"topicPrefix":       p.topicPrefix,
	}
}