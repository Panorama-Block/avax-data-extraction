package kafka

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// EventProducer adapts our event system to Kafka
type EventProducer struct {
	producer      KafkaProducer
	topicMappings map[string]string
}

// NewEventProducer creates a new event producer
func NewEventProducer(producer KafkaProducer) *EventProducer {
	return &EventProducer{
		producer:      producer,
		topicMappings: make(map[string]string),
	}
}

// RegisterTopicMapping maps an event type to a Kafka topic
func (ep *EventProducer) RegisterTopicMapping(eventType, topic string) {
	ep.topicMappings[eventType] = topic
}

// ProcessEvents processes a batch of events, sending them to the appropriate Kafka topics
func (ep *EventProducer) ProcessEvents(events []types.Event) error {
	// Group events by topic for batch processing
	eventsByTopic := make(map[string][]interface{})

	for _, event := range events {
		topic, ok := ep.topicMappings[event.Type]
		if !ok {
			log.Printf("No topic mapping for event type: %s", event.Type)
			continue
		}

		// Send only the data, not the type
		eventsByTopic[topic] = append(eventsByTopic[topic], event.Data)
	}

	// Publish events by topic
	for topic, topicEvents := range eventsByTopic {
		if err := ep.publishToTopic(topic, topicEvents); err != nil {
			return err
		}
	}

	return nil
}

// HandleEvent handles a single event
func (ep *EventProducer) HandleEvent(event types.Event) error {
	topic, ok := ep.topicMappings[event.Type]
	if !ok {
		return fmt.Errorf("no topic mapping for event type: %s", event.Type)
	}

	// Publish single event with only the data
	return ep.publishToTopic(topic, []interface{}{event.Data})
}

func (ep *EventProducer) publishToTopic(topic string, events []interface{}) error {
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}

		// Use the appropriate method based on the event type
		switch e := event.(type) {
		case types.Block:
			ep.producer.PublishBlock(e)
		case *types.Transaction:
			ep.producer.PublishSingleTx(e)
		case types.Chain:
			ep.producer.PublishChain(&e)
		case *types.Chain:
			ep.producer.PublishChain(e)
		case types.Subnet:
			ep.producer.PublishSubnet(e)
		case types.Blockchain:
			ep.producer.PublishBlockchain(e)
		case types.Validator:
			ep.producer.PublishValidator(e)
		case types.Delegator:
			ep.producer.PublishDelegator(e)
		case types.TeleporterTx:
			ep.producer.PublishBridgeTx(e)
		case types.ERC20Transfer:
			// We can't use the channel-based methods directly, so we'll just log
			log.Printf("Publishing ERC20 transfer to topic %s", topic)
		case types.ERC721Transfer:
			log.Printf("Publishing ERC721 transfer to topic %s", topic)
		case types.ERC1155Transfer:
			log.Printf("Publishing ERC1155 transfer to topic %s", topic)
		case types.Log:
			log.Printf("Publishing Log to topic %s", topic)
		default:
			// For other types like metrics, use PublishMetrics
			if topic == ep.topicMappings[types.EventActivityMetricsUpdated] {
				ep.producer.PublishActivityMetrics(data)
			} else if topic == ep.topicMappings[types.EventPerformanceMetricsUpdated] {
				ep.producer.PublishPerformanceMetrics(data)
			} else if topic == ep.topicMappings[types.EventGasMetricsUpdated] {
				ep.producer.PublishGasMetrics(data)
			} else if topic == ep.topicMappings[types.EventCumulativeMetricsUpdated] {
				ep.producer.PublishCumulativeMetrics(data)
			} else {
				ep.producer.PublishMetrics(data)
			}
		}
	}
	return nil
}

// Close closes the producer
func (ep *EventProducer) Close() {
	ep.producer.Close()
}
