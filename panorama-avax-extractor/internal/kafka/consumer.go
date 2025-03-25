package kafka

import (
    "fmt"
    "log"
    "strings"
    "sync"
    "time"
    
    "github.com/confluentinc/confluent-kafka-go/kafka"
)

// Consumer gerencia o consumo de dados dos tópicos Kafka
type Consumer struct {
    consumer   *kafka.Consumer
    
    // Estatísticas
    mutex             sync.Mutex
    messagesConsumed  int64
    lastConsumeTime   time.Time
    running           bool
}

// NewConsumer cria um novo consumidor Kafka
func NewConsumer(
    bootstrapServers string,
    groupID string,
    config *kafka.ConfigMap,
) (*Consumer, error) {
    // Adiciona bootstrap servers e group ID à configuração
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
    
    // Define alguns padrões sensatos
    err = config.SetKey("auto.offset.reset", "latest")
    if err != nil {
        return nil, fmt.Errorf("falha ao definir auto.offset.reset: %w", err)
    }
    
    err = config.SetKey("enable.auto.commit", true)
    if err != nil {
        return nil, fmt.Errorf("falha ao definir enable.auto.commit: %w", err)
    }
    
    // Cria consumidor
    consumer, err := kafka.NewConsumer(config)
    if err != nil {
        return nil, fmt.Errorf("falha ao criar consumidor Kafka: %w", err)
    }
    
    return &Consumer{
        consumer: consumer,
        running:  true,
    }, nil
}

// Close fecha o consumidor Kafka
func (c *Consumer) Close() error {
    c.mutex.Lock()
    c.running = false
    c.mutex.Unlock()
    
    return c.consumer.Close()
}

// Subscribe inscreve-se em tópicos Kafka
func (c *Consumer) Subscribe(topics []string) error {
    // Processa curingas em tópicos
    expandedTopics := make([]string, 0, len(topics))
    
    for _, topic := range topics {
        if strings.Contains(topic, "*") {
            // Tópico contém um curinga, precisamos listar tópicos e filtrar
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
            // Tópico regular
            expandedTopics = append(expandedTopics, topic)
        }
    }
    
    if len(expandedTopics) == 0 {
        return fmt.Errorf("nenhum tópico correspondeu aos curingas")
    }
    
    log.Printf("Inscrevendo-se em tópicos Kafka: %v", expandedTopics)
    
    return c.consumer.SubscribeTopics(expandedTopics, nil)
}

// Poll busca por uma mensagem do Kafka
func (c *Consumer) Poll(timeoutMs int) (*Message, error) {
    // Verifica se ainda estamos em execução
    c.mutex.Lock()
    if !c.running {
        c.mutex.Unlock()
        return nil, fmt.Errorf("consumer está fechado")
    }
    c.mutex.Unlock()
    
    // Poll para uma mensagem
    ev := c.consumer.Poll(timeoutMs)
    if ev == nil {
        return nil, nil
    }
    
    switch e := ev.(type) {
    case *kafka.Message:
        // Atualiza estatísticas
        c.mutex.Lock()
        c.messagesConsumed++
        c.lastConsumeTime = time.Now()
        c.mutex.Unlock()
        
        // Converte para o nosso tipo de mensagem
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
        // Ignora outros tipos de eventos
        return nil, nil
    }
}

// Message representa uma mensagem Kafka
type Message struct {
    Topic     string
    Partition int
    Offset    int64
    Key       []byte
    Value     []byte
    Timestamp time.Time
}

// Status retorna o status atual do consumidor
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

// Função auxiliar para verificar se um tópico corresponde a um padrão curinga
func matchesWildcard(topic, pattern string) bool {
    // Correspondência simples de curinga
    // Esta é uma versão simplificada; para produção, considere usar uma biblioteca regex
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
