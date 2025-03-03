package kafka

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/Panorama-Block/avax/internal/types"
)

// Publica transações completas no Kafka
func (p *Producer) SendTransaction(tx *types.Transaction) {
	data, _ := json.Marshal(tx)
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)

	if err != nil {
		log.Printf("Erro ao enviar transação para Kafka: %v", err)
	} else {
		log.Println("Transação publicada no Kafka:", tx.TxHash)
	}
}

// Publica transferências ERC20 no Kafka
func (p *Producer) SendERC20Transfer(transfer types.ERC20Transfer) {
	data, _ := json.Marshal(transfer)
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)

	if err != nil {
		log.Printf("Erro ao enviar ERC20 Transfer para Kafka: %v", err)
	} else {
		log.Println("Transferência ERC20 publicada no Kafka:", transfer.TransactionHash)
	}
}

// Publica logs de eventos no Kafka
func (p *Producer) SendLog(logData types.Log) {
	data, _ := json.Marshal(logData)
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}, nil)

	if err != nil {
		log.Printf("Erro ao enviar log para Kafka: %v", err)
	} else {
		log.Println("Log publicado no Kafka:", logData.Address)
	}
}
