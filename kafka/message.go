package kafka

import "encoding/json"
import "github.com/segmentio/kafka-go"

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
