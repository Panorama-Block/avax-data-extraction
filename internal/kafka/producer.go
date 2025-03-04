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

func (p *Producer) SendChain(chain *types.Chain) {
	data, _ := json.Marshal(chain)
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          data,
	}
	if err := p.Producer.Produce(message, nil); err != nil {
		log.Printf("Erro ao enviar chain para Kafka: %v", err)
	} else {
		log.Println("Chain publicada no Kafka:", chain.ChainID)
	}
	p.Producer.Flush(1000)
}

func (p *Producer) PublishTransactions(txChan <-chan *types.Transaction) {
	for tx := range txChan {
		data, _ := json.Marshal(tx)
		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
			Value:          data,
		}
		if err := p.Producer.Produce(message, nil); err != nil {
			log.Printf("Erro ao enviar transação para Kafka: %v", err)
		} else {
			log.Println("Transação publicada no Kafka:", tx.TxHash)
		}
	}
}

func (p *Producer) PublishERC20Transfers(erc20Chan <-chan types.ERC20Transfer) {
	for transfer := range erc20Chan {
		data, _ := json.Marshal(transfer)
		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
			Value:          data,
		}
		if err := p.Producer.Produce(message, nil); err != nil {
			log.Printf("Erro ao enviar transferência ERC20 para Kafka: %v", err)
		} else {
			log.Println("Transferência ERC20 publicada no Kafka:", transfer.TransactionHash)
		}
	}
}

func (p *Producer) PublishLogs(logChan <-chan types.Log) {
	for logData := range logChan {
		data, _ := json.Marshal(logData)
		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
			Value:          data,
		}
		if err := p.Producer.Produce(message, nil); err != nil {
			log.Printf("Erro ao enviar log para Kafka: %v", err)
		} else {
			log.Println("Log publicado no Kafka para o endereço:", logData.Address)
		}
	}
}
