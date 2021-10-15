package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
)

func (manager *Manager) Produce(message *Message) {
	if manager.Producer() == nil {
		return
	}

	if !message.isEmpty() {
		manager.Producer().WriteMessages(context.Background(),
			kafka.Message{
				Headers: message.Headers,
				Value:   message.Value,
			})
	}
}

func (manager *Manager) Producer() *kafka.Writer {
	if manager.producer != nil {
		return manager.producer
	}

	if len(manager.Config.KafkaBrokers) == 0 {
		return nil
	}

	manager.producer = &kafka.Writer{
		Addr:  kafka.TCP(manager.Config.KafkaBrokers[0]),
		Topic: manager.Config.ProducerConfig.Topic,
	}

	return manager.producer
}

func (manager *Manager) Consume(consumerHandler func(Message)) {
	if manager.Consumer() == nil {
		return
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
