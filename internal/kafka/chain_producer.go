package kafka

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/Panorama-Block/avax/internal/types"
)

type Producer struct {
	Producer *kafka.Producer
	Topic    string
}

func NewProducer(broker, topic string) *Producer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		log.Fatalf("Erro ao conectar ao Kafka: %v", err)
	}
	return &Producer{Producer: p, Topic: topic}
}

func (p *Producer) Send(chain *types.Chain) {
	data, _ := json.Marshal(chain)
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)
	if err != nil {
		log.Printf("Erro ao enviar mensagem para Kafka: %v", err)
	} else {
		log.Println("Publicado no Kafka:", string(data))
	}

	p.Producer.Flush(1000)
}

