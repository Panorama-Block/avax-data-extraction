package kafka

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Consumer handles consuming data from Kafka topics
type Consumer struct {
	consumer   *kafka.Consumer
	
	// Stats
	mutex             sync.Mutex
	messagesConsumed  int64
	lastConsumeTime   time.Time
	running           bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	bootstrapServers string,
	groupID string,
	config *kafka.ConfigMap,
) (*Consumer, error) {
	// Add bootstrap servers and group ID to config
	if config == nil {
		config = &kafka.ConfigMap{}
	}
	
	err := config.SetKey("bootstrap.servers", bootstrapServers)
	if err != nil {
		return nil, fmt.Errorf("failed to set bootstrap servers: %w", err)
	}
	
	err = config.SetKey("group.id", groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to set group ID: %w", err)
	}
	
	// Set some sensible defaults
	err = config.SetKey("auto.offset.reset", "latest")
	if err != nil {
		return nil, fmt.Errorf("failed to set auto.offset.reset: %w", err)
	}
	
	err = config.SetKey("enable.auto.commit", true)
	if err != nil {
		return nil, fmt.Errorf("failed to set enable.auto.commit: %w", err)
	}
	
	// Create consumer
	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	
	return &Consumer{
		consumer: consumer,
		running:  true,
	}, nil
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	c.mutex.Lock()
	c.running = false
	c.mutex.Unlock()
	
	return c.consumer.Close()
}

// Subscribe subscribes to Kafka topics
func (c *Consumer) Subscribe(topics []string) error {
	// Process wildcards in topics
	expandedTopics := make([]string, 0, len(topics))
	
	for _, topic := range topics {
		if strings.Contains(topic, "*") {
			// Topic contains a wildcard, we need to list topics and filter
			metadata, err := c.consumer.GetMetadata(nil, true, 10000)
			if err != nil {
				return fmt.Errorf("failed to get metadata: %w", err)
			}
			
			wildcard := strings.Replace(topic, "*", ".*", -1)
			
			for kafkaTopic := range metadata.Topics {
				if matchesWildcard(kafkaTopic, wildcard) {
					expandedTopics = append(expandedTopics, kafkaTopic)
				}
			}
		} else {
			// Regular topic
			expandedTopics = append(expandedTopics, topic)
		}
	}
	
	if len(expandedTopics) == 0 {
		return fmt.Errorf("no topics matched the wildcards")
	}
	
	log.Printf("Subscribing to Kafka topics: %v", expandedTopics)
	
	return c.consumer.SubscribeTopics(expandedTopics, nil)
}

// Poll polls for a message from Kafka
func (c *Consumer) Poll(timeoutMs int) (*Message, error) {
	// Check if we're still running
	c.mutex.Lock()
	if !c.running {
		c.mutex.Unlock()
		return nil, fmt.Errorf("consumer is closed")
	}
	c.mutex.Unlock()
	
	// Poll for a message
	ev := c.consumer.Poll(timeoutMs)
	if ev == nil {
		return nil, nil
	}
	
	switch e := ev.(type) {
	case *kafka.Message:
		// Update stats
		c.mutex.Lock()
		c.messagesConsumed++
		c.lastConsumeTime = time.Now()
		c.mutex.Unlock()
		
		// Convert to our message type
		return &Message{
			Topic:     *e.TopicPartition.Topic,
			Partition: int(e.TopicPartition.Partition),
			Offset:    int64(e.TopicPartition.Offset),
			Key:       e.Key,
			Value:     e.Value,
			Timestamp: e.Timestamp,
		}, nil
	case kafka.Error:
		return nil, fmt.Errorf("Kafka error: %v", e)
	default:
		// Ignore other event types
		return nil, nil
	}
}

// Message represents a Kafka message
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Timestamp time.Time
}

// Status returns the current status of the consumer
func (c *Consumer) Status() map[string]interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	lastConsumeStr := "Never"
	if !c.lastConsumeTime.IsZero() {
		lastConsumeStr = c.lastConsumeTime.Format(time.RFC3339)
	}
	
	subscription, err := c.consumer.Subscription()
	if err != nil {
		subscription = []string{"error getting subscription"}
	}
	
	return map[string]interface{}{
		"messagesConsumed": c.messagesConsumed,
		"lastConsumeTime":  lastConsumeStr,
		"running":          c.running,
		"subscription":     subscription,
	}
}

// Helper function to check if a topic matches a wildcard pattern
func matchesWildcard(topic, pattern string) bool {
	// Simple wildcard matching
	// This is a simplified version; for production, consider using a regex library
	parts := strings.Split(pattern, ".")
	topicParts := strings.Split(topic, ".")
	
	if len(parts) != len(topicParts) {
		return false
	}
	
	for i, part := range parts {
		if part != ".*" && part != topicParts[i] {
			return false
		}
	}
	
	return true
}