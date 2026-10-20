package mocks

import (
	"github.com/RedHatInsights/sources-api-go/kafka"
	m "github.com/RedHatInsights/sources-api-go/model"
)

type MockSuperKeyProducer struct {
	CreateRequestCallCount int
	DeleteRequestCallCount int
	LastApplication        *m.Application
	LastHeaders            []kafka.Header
}

func (m *MockSuperKeyProducer) SendCreateRequest(application *m.Application, headers []kafka.Header) error {
	m.CreateRequestCallCount++
	m.LastApplication = application
	m.LastHeaders = headers

	return nil
}

func (m *MockSuperKeyProducer) SendDeleteRequest(application *m.Application, headers []kafka.Header) error {
	m.DeleteRequestCallCount++
	m.LastApplication = application
	m.LastHeaders = headers

	return nil
}
