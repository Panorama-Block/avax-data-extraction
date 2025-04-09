package kafka

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// EventProducer adapts our event system to Kafka
type EventProducer struct {
	producer      *Producer
	topicMappings map[string]string
}

// NewEventProducer creates a new event producer
func NewEventProducer(producer *Producer) *EventProducer {
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

	// Publish single event
	return ep.publishToTopic(topic, []interface{}{event.Data})
}

func (ep *EventProducer) publishToTopic(topic string, events []interface{}) error {
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}

		if err := ep.producer.sendMessage(topic, data); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the producer
func (ep *EventProducer) Close() {
	ep.producer.Close()
} 