package events

import (
	c "github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const EventStreamTopic = "platform.sources.event-stream"

var config = c.Get()

type EventStreamProducer struct {
	Sender
}

type Sender interface {
	RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error
}

type EventStreamSender struct {
}

func (esp *EventStreamSender) RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error {
	logging.Log.Debugf("publishing message %v to topic %q...", eventType, EventStreamTopic)

	producerConfig := kafka.ProducerConfig{Topic: config.KafkaTopic(EventStreamTopic)}
	kafkaConfig := kafka.Config{KafkaBrokers: config.KafkaBrokers, ProducerConfig: producerConfig}
	kf := &kafka.Manager{Config: kafkaConfig}

	m := &kafka.Message{}

	for index, header := range headers {
		if header.Key == "event_type" {
			headers[index] = kafka.Header{Key: "event_type", Value: []byte(eventType)}
			break
		}
	}
	headers = append(headers, kafka.Header{Key: "encoding", Value: []byte("json")})

	m.AddHeaders(headers)
	m.AddValue(payload)

	err := kf.Produce(m)
	if err != nil {
		return err
	}

	logging.Log.Debugf("publishing message %v to topic %q...Complete", eventType, EventStreamTopic)

	return kf.Producer().Close()
}

func (esp *EventStreamProducer) RaiseEventIf(allowed bool, eventType string, payload []byte, headers []kafka.Header) error {
	if allowed {
		return esp.Sender.RaiseEvent(eventType, payload, headers)
	}

	return nil
}

func (esp *EventStreamProducer) RaiseEventForUpdate(resource util.Resource, updateAttributes []string, headers []kafka.Header) error {
	allowed := esp.RaiseEventAllowed(resource.ResourceType, updateAttributes)
	eventModelDao, err := dao.GetFromResourceType(resource.ResourceType)
	if err != nil {
		return err
	}

	resourceJSON, err := eventModelDao.ToEventJSON(resource)
	if err != nil {
		return err
	}

	err = esp.RaiseEventIf(allowed, resource.ResourceType+".update", resourceJSON, headers)
	if err != nil {
		return err
	}

	message, err := m.UpdateMessage(eventModelDao, resource, updateAttributes)
	if err != nil {
		return err
	}

	err = esp.RaiseEventIf(allowed, "Records.update", message, headers)
	if err != nil {
		return err
	}

	return nil
}

func (esp *EventStreamProducer) RaiseEventAllowed(resourceType string, attributes []string) bool {
	if resourceType != "Application" {
		return true
	}

	return !util.SliceContainsString(attributes, "_superkey")
}
