package kafka

import (
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

func (message *Message) ParseTo(output interface{}) error {
	err := json.Unmarshal(message.Value, &output)
	if err != nil {
		return err
	}

	return nil
}

func (message *Message) AddHeaders(headers []Header) {
	message.Headers = make([]kafka.Header, len(headers))
	for index, header := range headers {
		message.Headers[index] = kafka.Header{Key: header.Key, Value: header.Value}
	}
}

func (message *Message) GetHeader(name string) string {
	for _, header := range message.Headers {
		if header.Key == name {
			return string(header.Value)
		}
	}

	return ""
}

func (message *Message) AddValue(record []byte) {
	message.Value = record
}

func (message *Message) AddValueAsJSON(record interface{}) error {
	out, err := json.Marshal(record)
	if err != nil {
		return err
	}

	message.AddValue(out)

	return nil
}

func (message *Message) isEmpty() bool {
	isEmptyHeaders := message.Headers == nil || len(message.Headers) == 0

	return isEmptyHeaders && message.Value == nil
}

// translate a kafka message's headers from segmentio/kafka -> our kafka
func (message *Message) TranslateHeaders() []Header {
	if len(message.Headers) < 1 {
		return []Header{}
	}

	headers := make([]Header, len(message.Headers))
	for index, header := range message.Headers {
		headers[index] = Header{Key: header.Key, Value: header.Value}
	}

	return headers
}
