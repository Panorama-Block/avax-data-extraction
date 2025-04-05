package kafka

import (
    "fmt"
    "log"
    "strings"
    "sync"
    "time"
    
    "github.com/confluentinc/confluent-kafka-go/kafka"
)

type Consumer struct {
    consumer   *kafka.Consumer

    mutex             sync.Mutex
    messagesConsumed  int64
    lastConsumeTime   time.Time
    running           bool
}

func NewConsumer(
    bootstrapServers string,
    groupID string,
    config *kafka.ConfigMap,
) (*Consumer, error) {
    if config == nil {
        config = &kafka.ConfigMap{}
    }
    
    err := config.SetKey("bootstrap.servers", bootstrapServers)
    if err != nil {
        return nil, fmt.Errorf("falha ao definir bootstrap servers: %w", err)
    }
    
    err = config.SetKey("group.id", groupID)
    if err != nil {
        return nil, fmt.Errorf("falha ao definir group ID: %w", err)
    }
    

    err = config.SetKey("auto.offset.reset", "latest")
    if err != nil {
        return nil, fmt.Errorf("falha ao definir auto.offset.reset: %w", err)
    }
    
    err = config.SetKey("enable.auto.commit", true)
    if err != nil {
        return nil, fmt.Errorf("falha ao definir enable.auto.commit: %w", err)
    }
    
    consumer, err := kafka.NewConsumer(config)
    if err != nil {
        return nil, fmt.Errorf("falha ao criar consumidor Kafka: %w", err)
    }
    
    return &Consumer{
        consumer: consumer,
        running:  true,
    }, nil
}

func (c *Consumer) Close() error {
    c.mutex.Lock()
    c.running = false
    c.mutex.Unlock()
    
    return c.consumer.Close()
}

func (c *Consumer) Subscribe(topics []string) error {
    expandedTopics := make([]string, 0, len(topics))
    
    for _, topic := range topics {
        if strings.Contains(topic, "*") {
            metadata, err := c.consumer.GetMetadata(nil, true, 10000)
            if err != nil {
                return fmt.Errorf("falha ao obter metadados: %w", err)
            }
            
            wildcard := strings.Replace(topic, "*", ".*", -1)
            
            for kafkaTopic := range metadata.Topics {
                if matchesWildcard(kafkaTopic, wildcard) {
                    expandedTopics = append(expandedTopics, kafkaTopic)
                }
            }
        } else {
            expandedTopics = append(expandedTopics, topic)
        }
    }
    
    if len(expandedTopics) == 0 {
        return fmt.Errorf("nenhum tópico correspondeu aos curingas")
    }
    
    log.Printf("Inscrevendo-se em tópicos Kafka: %v", expandedTopics)
    
    return c.consumer.SubscribeTopics(expandedTopics, nil)
}

func (c *Consumer) Poll(timeoutMs int) (*Message, error) {
    c.mutex.Lock()
    if !c.running {
        c.mutex.Unlock()
        return nil, fmt.Errorf("consumer está fechado")
    }
    c.mutex.Unlock()

    ev := c.consumer.Poll(timeoutMs)
    if ev == nil {
        return nil, nil
    }
    
    switch e := ev.(type) {
    case *kafka.Message:
        c.mutex.Lock()
        c.messagesConsumed++
        c.lastConsumeTime = time.Now()
        c.mutex.Unlock()

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
        return nil, nil
    }
}

type Message struct {
    Topic     string
    Partition int
    Offset    int64
    Key       []byte
    Value     []byte
    Timestamp time.Time
}

func (c *Consumer) Status() map[string]interface{} {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    lastConsumeStr := "Nunca"
    if !c.lastConsumeTime.IsZero() {
        lastConsumeStr = c.lastConsumeTime.Format(time.RFC3339)
    }
    
    subscription, err := c.consumer.Subscription()
    if err != nil {
        subscription = []string{"erro ao obter inscrição"}
    }
    
    return map[string]interface{}{
        "messagesConsumed": c.messagesConsumed,
        "lastConsumeTime":  lastConsumeStr,
        "running":          c.running,
        "subscription":     subscription,
    }
}

func matchesWildcard(topic, pattern string) bool {

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
