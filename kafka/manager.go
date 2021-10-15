package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

func (manager *Manager) Produce(message *Message) error {
	if manager.Producer() == nil {
		return fmt.Errorf("producer is not initialized")
	}

	if !message.isEmpty() {
		err := manager.Producer().WriteMessages(context.Background(),
			kafka.Message{
				Headers: message.Headers,
				Value:   message.Value,
			})
		if err != nil {
			return err
		}
	}

	return nil
}

func (manager *Manager) Producer() *kafka.Writer {
	if manager.producer != nil {
		return manager.producer
	}

	if len(manager.Config.KafkaBrokers) == 0 {
		return nil
	}

	manager.producer = &kafka.Writer{
		Addr:  kafka.TCP(manager.Config.KafkaBrokers...),
		Topic: manager.Config.ProducerConfig.Topic,
	}

	return manager.producer
}

func (manager *Manager) Consume(consumerHandler func(Message)) error {
	if manager.Consumer() == nil {
		return fmt.Errorf("consumer is not initialized")
	}

	for {
		message, err := manager.Consumer().ReadMessage(context.Background())
		if err != nil {
			continue
		}

		go consumerHandler(Message(message))
	}
}

func (manager *Manager) Consumer() *kafka.Reader {
	if manager.consumer != nil {
		return manager.consumer
	}

	manager.consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers: manager.Config.KafkaBrokers,
		GroupID: manager.ConsumerConfig.GroupID,
		Topic:   manager.ConsumerConfig.Topic,
	})
	return manager.consumer
}
