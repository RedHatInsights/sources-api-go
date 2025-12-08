package mocks

import "github.com/RedHatInsights/sources-api-go/kafka"

type MockSender struct {
	Hit     int
	Headers []kafka.Header
	Body    string
}

func (m *MockSender) RaiseEvent(_ string, b []byte, headers []kafka.Header) error {
	m.Headers = headers
	m.Body = string(b)
	m.Hit++

	return nil
}
