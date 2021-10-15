package kafka

import "github.com/segmentio/kafka-go"

type ProducerConfig struct {
	Topic string
}

type ConsumerConfig struct {
	Topic   string
	GroupID string
}

type Config struct {
	ProducerConfig
	ConsumerConfig
	KafkaBrokers []string
}

type Manager struct {
	Config
	consumer *kafka.Reader
	producer *kafka.Writer
}

type Header kafka.Header
type Message kafka.Message
