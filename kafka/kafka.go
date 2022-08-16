package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// CloseReader attempts to close the provided reader and logs the error if it fails.
func CloseReader(reader *Reader, context string) {
	if reader == nil {
		return
	}

	err := reader.Close()
	if err != nil && reader.Options.Logger != nil {
		reader.Options.Logger.Errorf(`unable to close reader in the "%s" context: %s`, context, err)
	}
}

// CloseWriter attempts to close the provided writer and logs the error if it fails.
func CloseWriter(writer *Writer, context string) {
	if writer == nil {
		return
	}

	err := writer.Close()
	if err != nil && writer.ErrorLogger != nil {
		writer.Options.Logger.Errorf(`unable to close writer in the "%s" context: %s`, context, err)
	}
}

// Consume consumes a message from the reader with the provided handler function.
func Consume(reader *Reader, consumerHandler func(Message)) {
	if reader == nil || reader.Reader == nil {
		panic("cannot consume on a nil reader, be sure to initialize with kafka.NewReader()")
	}

	for {
		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			if reader.Options.Logger != nil {
				reader.Options.Logger.Errorf(`error reading Kafka message: %s. Continuing...`, err)
			}

			continue
		}

		go consumerHandler(Message(message))
	}
}

// GetReader returns a Kafka reader configured with the specified settings.
func GetReader(conf *Options) (*Reader, error) {
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
		Brokers: []string{fmt.Sprintf("%s:%d", conf.BrokerConfig.Hostname, *conf.BrokerConfig.Port)},
		Topic:   conf.Topic,
	}

	// set up the logger
	if conf.Logger != nil {
		readerConfig.Logger = kafka.LoggerFunc(conf.Logger.Debugf)
		readerConfig.ErrorLogger = kafka.LoggerFunc(conf.Logger.Errorf)
	}

	// set the group id if present
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

	return &Reader{Reader: kafka.NewReader(readerConfig), Options: conf}, nil
}

// GetWriter returns a Kafka writer configured with the specified settings.
func GetWriter(conf *Options) (*Writer, error) {
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
		Addr:  kafka.TCP(fmt.Sprintf("%s:%d", conf.BrokerConfig.Hostname, *conf.BrokerConfig.Port)),
		Topic: conf.Topic,
	}

	if conf.Logger != nil {
		kafkaWriter.Logger = kafka.LoggerFunc(conf.Logger.Debugf)
		kafkaWriter.ErrorLogger = kafka.LoggerFunc(conf.Logger.Errorf)
	}

	if conf.BrokerConfig.Authtype != nil {
		tls := CreateTLSConfig(conf.BrokerConfig.Cacert)

		mechanism, err := CreateSaslMechanism(conf.BrokerConfig.Sasl)
		if err != nil {
			return nil, fmt.Errorf(`unable to create Kafka producer's Sasl mechanism: %w`, err)
		}

		kafkaWriter.Transport = CreateTransport(mechanism, tls)
	}

	return &Writer{Writer: kafkaWriter, Options: conf}, nil
}

// Produce produces a message with the writer.
func Produce(writer *Writer, message *Message) error {
	if writer == nil || writer.Writer == nil {
		return fmt.Errorf("cannot produce on a nil writer - be sure to initialize with kafka.NewWriter()")
	}

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

			if writer.Options.Logger != nil {
				writer.Options.Logger.Errorf(`could not produce Kafka message. Headers: %s, content: "%s": %w`, headersMap, string(message.Value), err)
			}

			return fmt.Errorf(`could not produce Kafka message. Headers: %s, content: "%s": %w`, headersMap, string(message.Value), err)
		}
	}

	return nil
}
