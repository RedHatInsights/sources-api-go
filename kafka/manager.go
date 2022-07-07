package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/segmentio/kafka-go"
)

func (manager *Manager) Produce(message *Message) error {
	producer, err := manager.Producer()
	if err != nil {
		return err
	}

	if !message.isEmpty() {
		err = producer.WriteMessages(context.Background(),
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

// CloseConsumer attempts to close the consumer.
func (manager *Manager) CloseConsumer() error {
	if manager.consumer != nil {
		return manager.consumer.Close()
	}

	return nil
}

// CloseProducer attempts to close the producer.
func (manager *Manager) CloseProducer() error {
	if manager.producer != nil {
		return manager.producer.Close()
	}

	return nil
}

func (manager *Manager) Producer() (*kafka.Writer, error) {
	if manager.producer != nil {
		return manager.producer, nil
	}

	if len(manager.Config.KafkaBrokers) == 0 {
		return nil, errors.New("there are no Kafka brokers to create a producer from")
	}

	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP(manager.Config.KafkaBrokers...),
		Topic: manager.Config.ProducerConfig.Topic,
	}

	// When using managed Kafka, Clowder will add some Sasl authentication details so that the services can connect to
	// it. The following code block sets up "kafka-go" to work with these settings.
	if config.Get().KafkaSaslEnabled {
		var conf = config.Get()

		tls, err := CreateTLSConfig(conf.KafkaSaslCaPath)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Kafka producer's TLS configuration: %w`, err)
		}

		mechanism, err := CreateSaslMechanism(conf.KafkaSaslMechanism, conf.KafkaSaslUsername, conf.KafkaSaslPassword)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Kafka producer's mechanism: %s`, err)
		}

		kafkaWriter.Transport = &kafka.Transport{
			SASL: mechanism,
			TLS:  tls,
		}
	}

	manager.producer = kafkaWriter
	return manager.producer, nil
}

func (manager *Manager) Consume(consumerHandler func(Message)) error {
	consumer, err := manager.Consumer()
	if err != nil {
		return err
	}

	for {
		message, err := consumer.ReadMessage(context.Background())
		if err != nil {
			continue
		}

		go consumerHandler(Message(message))
	}
}

func (manager *Manager) Consumer() (*kafka.Reader, error) {
	if manager.consumer != nil {
		return manager.consumer, nil
	}

	readerConfig := kafka.ReaderConfig{
		Brokers: manager.Config.KafkaBrokers,
		GroupID: manager.ConsumerConfig.GroupID,
		Topic:   manager.ConsumerConfig.Topic,
	}

	// When using managed Kafka, Clowder will add some Sasl authentication details so that the services can connect to
	// it. The following code block sets up "kafka-go" to work with these settings.
	if config.Get().KafkaSaslEnabled {
		var conf = config.Get()
		dialer, err := CreateDialer(conf.KafkaSaslCaPath, conf.KafkaSaslMechanism, conf.KafkaSaslUsername, conf.KafkaSaslPassword)
		if err != nil {
			return nil, fmt.Errorf(`unable to create the dialer for the Kafka reader: %w`, err)
		}

		readerConfig.Dialer = dialer
	}

	manager.consumer = kafka.NewReader(readerConfig)
	return manager.consumer, nil
}
