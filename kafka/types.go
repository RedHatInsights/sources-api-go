package kafka

import (
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/segmentio/kafka-go"
)

// Options is a struct for creating a reader/writer
type Options struct {
	// REQUIRED FIELDS
	BrokerConfig *clowder.BrokerConfig
	Topic        string

	// only used for reader, optional.
	GroupID *string

	// logger to pass along
	Logger Logger
}

type Header kafka.Header
type Message kafka.Message

// wrapping the reader/writer types we're using so we can change them under the
// hood in the future if necessary
type Reader struct {
	*kafka.Reader

	Options *Options
}

type Writer struct {
	*kafka.Writer

	Options *Options
}

// wrapper around the logger methods we need
type Logger interface {
	Debugf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
}
