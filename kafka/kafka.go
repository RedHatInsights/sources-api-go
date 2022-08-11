package kafka

import (
	"context"
	"errors"
	"fmt"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/segmentio/kafka-go"
)

type Options struct {
	// REQUIRED FIELDS
	BrokerConfig *clowder.BrokerConfig
	Topic        string

	// only used for reader, optional.
	GroupID *string

	// logging functions to pass along
	LoggerFunction      kafka.LoggerFunc
	ErrorLoggerFunction kafka.LoggerFunc
}

// CloseReader attempts to close the provided reader and logs the error if it fails.
func CloseReader(reader *kafka.Reader, context string) {
	err := reader.Close()
	if err != nil && reader.Config().ErrorLogger != nil {
		reader.Config().ErrorLogger.Printf(`unable to close reader in the "%s" context: %s`, context, err)
	}
}

// CloseWriter attempts to close the provided writer and logs the error if it fails.
func CloseWriter(writer *kafka.Writer, context string) {
	err := writer.Close()
	if err != nil && writer.ErrorLogger != nil {
		writer.ErrorLogger.Printf(`unable to close writer in the "%s" context: %s`, context, err)
	}
}

// Consume consumes a message from the reader with the provided handler function.
func Consume(reader *kafka.Reader, consumerHandler func(Message)) {
	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			if reader.Config().ErrorLogger != nil {
				reader.Config().ErrorLogger.Printf(`error reading Kafka message: %s. Continuing...`, err)
			}

			continue
		}

		go consumerHandler(Message(message))
	}
}

// GetReader returns a Kafka reader configured with the specified settings.
func GetReader(conf *Options) (*kafka.Reader, error) {
	if conf.BrokerConfig == nil {
		return nil, errors.New("could not create Kafka reader: the provided configuration is empty")
	}

	if conf.BrokerConfig.Port == nil || *conf.BrokerConfig.Port == 0 {
		return nil, errors.New("could not create a Kafka reader: the provided port is empty")
	}

	if conf.Topic == "" {
		return nil, errors.New("could not create a Kafka reader: a topic is required")
	}

	readerConfig := kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", conf.BrokerConfig.Hostname, *conf.BrokerConfig.Port)},
		Topic:       conf.Topic,
		Logger:      conf.LoggerFunction,
		ErrorLogger: conf.ErrorLoggerFunction,
	}

	if conf.GroupID != nil {
		readerConfig.GroupID = *conf.GroupID
	}

	// When using managed Kafka, Clowder will add some Sasl authentication details so that the services can connect to
	// it. The following code block sets up "kafka-go" to work with these settings.
	if conf.BrokerConfig.Authtype != nil {
		dialer, err := CreateDialer(conf.BrokerConfig)
		if err != nil {
			return nil, fmt.Errorf(`unable to create the dialer for the Kafka reader: %w`, err)
		}

		readerConfig.Dialer = dialer
	}

	return kafka.NewReader(readerConfig), nil
}

// GetWriter returns a Kafka writer configured with the specified settings.
func GetWriter(conf *Options) (*kafka.Writer, error) {
	if conf.BrokerConfig == nil {
		return nil, errors.New("could not create Kafka writer: the provided configuration is empty")
	}

	if conf.BrokerConfig.Port == nil || *conf.BrokerConfig.Port == 0 {
		return nil, errors.New("could not create a Kafka writer: the provided port is empty")
	}

	if conf.Topic == "" {
		return nil, errors.New("could not create a Kafka writer: a topic is required")
	}

	kafkaWriter := &kafka.Writer{
		Addr:        kafka.TCP(fmt.Sprintf("%s:%d", conf.BrokerConfig.Hostname, *conf.BrokerConfig.Port)),
		Topic:       conf.Topic,
		Logger:      conf.LoggerFunction,
		ErrorLogger: conf.ErrorLoggerFunction,
	}

	if conf.BrokerConfig.Authtype != nil {
		tls := CreateTLSConfig(conf.BrokerConfig.Cacert)

		mechanism, err := CreateSaslMechanism(conf.BrokerConfig.Sasl)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Kafka producer's Sasl mechanism: %w`, err)
		}

		kafkaWriter.Transport = CreateTransport(mechanism, tls)
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
			headersMap := make(map[string]string)
			for _, h := range message.Headers {
				headersMap[h.Key] = string(h.Value)
			}

			return fmt.Errorf(`could not produce Kafka message. Headers: %s, content: "%s": %w`, headersMap, string(message.Value), err)
		}
	}

	return nil
}
