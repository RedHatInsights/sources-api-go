package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/logger"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/segmentio/kafka-go"
)

// CloseReader attempts to close the provided reader and logs the error if it fails.
func CloseReader(reader *kafka.Reader, context string) {
	err := reader.Close()
	if err != nil {
		logger.Log.Errorf(`unable to close reader in the "%s" context: %s`, context, err)
	}
}

// CloseWriter attempts to close the provided writer and logs the error if it fails.
func CloseWriter(writer *kafka.Writer, context string) {
	err := writer.Close()
	if err != nil {
		logger.Log.Errorf(`unable to close writer in the "%s" context: %s`, context, err)
	}
}

// Consume consumes a message from the reader with the provided handler function.
func Consume(reader *kafka.Reader, consumerHandler func(Message)) {
	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			logger.Log.Warnf(`error reading Kafka message: %s. Continuing...`, err)

			continue
		}

		go consumerHandler(Message(message))
	}
}

// GetReader returns a Kafka reader configured with the specified settings.
func GetReader(brokerConfig *clowder.BrokerConfig, groupId string, topic string) (*kafka.Reader, error) {
	if brokerConfig == nil {
		return nil, errors.New("could not create Kafka reader: the provided configuration is empty")
	}

	if brokerConfig.Port == nil || *brokerConfig.Port == 0 {
		return nil, errors.New("could not create a Kafka reader: the provided port is empty")
	}

	readerConfig := kafka.ReaderConfig{
		Brokers: []string{fmt.Sprintf("%s:%d", brokerConfig.Hostname, *brokerConfig.Port)},
		GroupID: groupId,
		Topic:   topic,
	}

	// When using managed Kafka, Clowder will add some Sasl authentication details so that the services can connect to
	// it. The following code block sets up "kafka-go" to work with these settings.
	if brokerConfig.Authtype != nil {
		dialer, err := CreateDialer(brokerConfig)
		if err != nil {
			return nil, fmt.Errorf(`unable to create the dialer for the Kafka reader: %w`, err)
		}

		readerConfig.Dialer = dialer
	}

	return kafka.NewReader(readerConfig), nil
}

// GetWriter returns a Kafka writer configured with the specified settings.
func GetWriter(brokerConfig *clowder.BrokerConfig, topic string) (*kafka.Writer, error) {
	if brokerConfig == nil {
		return nil, errors.New("could not create Kafka writer: the provided configuration is empty")
	}

	if brokerConfig.Port == nil || *brokerConfig.Port == 0 {
		return nil, errors.New("could not create a Kafka writer: the provided port is empty")
	}

	kafkaWriter := &kafka.Writer{
		Addr:  kafka.TCP(fmt.Sprintf("%s:%d", brokerConfig.Hostname, *brokerConfig.Port)),
		Topic: topic,
	}

	if brokerConfig.Authtype != nil {
		tls := CreateTLSConfig(brokerConfig.Cacert)

		mechanism, err := CreateSaslMechanism(brokerConfig.Sasl)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Kafka producer's Sasl mechanism: %w`, err)
		}

		kafkaWriter.Transport = &kafka.Transport{
			SASL: mechanism,
			TLS:  tls,
		}
	}

	return kafkaWriter, nil
}

// Produce produces a message with the writer.
func Produce(writer *kafka.Writer, message *Message) error {
	if !message.isEmpty() {
		err := writer.WriteMessages(
			context.Background(),
			kafka.Message{
				Headers: message.Headers,
				Value:   message.Value,
			},
		)

		if err != nil {
			return fmt.Errorf(`could not produce Kafka message "%v": %w`, message, err)
		}
	}

	return nil
}
