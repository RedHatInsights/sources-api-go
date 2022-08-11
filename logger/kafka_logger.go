package logger

import "github.com/segmentio/kafka-go"

func KafkaLogger() kafka.LoggerFunc {
	return Log.WithField("kafka", "debug").Debugf
}

func KafkaErrorLogger() kafka.LoggerFunc {
	return Log.WithField("kafka", "error").Errorf
}
