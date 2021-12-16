package statuslistener

import (
	"sort"
	"time"

	c "github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const (
	SourcesStatusTopic      = "platform.sources.status"
	GroupID                 = "sources-api-status-worker"
	EventAvailabilityStatus = "availability_status"
)

var config = c.Get()

type AvailabilityStatusListener struct {
	*events.EventStreamProducer
}

func Run() {
	avs := AvailabilityStatusListener{EventStreamProducer: NewEventStreamProducer()}
	avs.subscribeToAvailabilityStatus()
}

func NewEventStreamProducer() *events.EventStreamProducer {
	sender := &events.EventStreamSender{}
	return &events.EventStreamProducer{Sender: sender}
}

func (avs *AvailabilityStatusListener) subscribeToAvailabilityStatus() {
	if logging.Log == nil {
		panic("logging is not initialized")
	}

	consumerConfig := kafka.ConsumerConfig{Topic: config.KafkaTopic(SourcesStatusTopic), GroupID: GroupID}
	kafkaConfig := kafka.Config{KafkaBrokers: config.KafkaBrokers, ConsumerConfig: consumerConfig}

	kf := &kafka.Manager{Config: kafkaConfig}

	errorConsume := kf.Consume(avs.ConsumeStatusMessage)

	if errorConsume != nil {
		logging.Log.Errorf("Consumer kafka message error: %s", errorConsume.Error())
	}
}

func (avs *AvailabilityStatusListener) ConsumeStatusMessage(message kafka.Message) {
	var statusMessage StatusMessage
	err := message.ParseTo(&statusMessage)
	if err != nil {
		logging.Log.Errorf("Error in parsing status message %s", err.Error())
		return
	}

	et := avs.eventType(message)
	if et != EventAvailabilityStatus {
		return
	}

	logging.Log.Infof("Kafka message %s, %s received with payload: %s", message.Headers, message.Key, message.Value)

	headers := avs.headersWithoutEventType(message)

	avs.processEvent(statusMessage, headers)
}

func (avs *AvailabilityStatusListener) headersWithoutEventType(message kafka.Message) []kafka.Header {
	if len(message.Headers) < 1 {
		return []kafka.Header{}
	}

	//headers := make([]kafka.Header, len(message.Headers))
	var headers []kafka.Header
	for _, header := range message.Headers {
		if header.Key != "event_type" {
			headers = append(headers, kafka.Header(header))
		}
	}

	return headers
}

func (avs *AvailabilityStatusListener) eventType(message kafka.Message) string {
	for _, header := range message.Headers {
		if header.Key == "event_type" {
			return string(header.Value)
		}
	}

	return ""
}

func (avs *AvailabilityStatusListener) processEvent(statusMessage StatusMessage, headers []kafka.Header) {
	resourceID, err := util.InterfaceToInt64(statusMessage.ResourceID)
	if err != nil {
		logging.Log.Errorf("Error parsing resource_id: %s", err.Error())
		return
	}

	if !util.SliceContainsString(m.AvailabilityStatuses, statusMessage.Status) {
		logging.Log.Errorf("Invalid Status: %s", statusMessage.Status)
		return
	}

	updateAttributes := avs.attributesForUpdate(statusMessage)
	modelEventDao, err := dao.GetFrom(statusMessage.ResourceType)
	if err != nil {
		logging.Log.Error(err.Error())
		return
	}

	err = (*modelEventDao).FetchAndUpdateBy(&resourceID, updateAttributes)
	if err != nil {
		logging.Log.Errorf("Update error in status availability: %s", err)
		return
	}

	updateAttributeKeys := make([]string, len(updateAttributes))
	i := 0
	for k := range updateAttributes {
		updateAttributeKeys[i] = k
		i++
	}
	sort.Strings(updateAttributeKeys)

	err = avs.EventStreamProducer.RaiseEventForUpdate(resourceID, statusMessage.ResourceType, updateAttributeKeys, headers)
	if err != nil {
		logging.Log.Errorf("Error in raising event for update: %s, resource: %s(%s)", err.Error(), statusMessage.ResourceType, statusMessage.ResourceID)
	}
}

func (avs *AvailabilityStatusListener) attributesForUpdate(statusMessage StatusMessage) map[string]interface{} {
	updateAttributes := make(map[string]interface{})

	updateAttributes["last_checked_at"] = time.Now().Format("2006-01-02T15:04:05.999Z")
	updateAttributes["availability_status"] = statusMessage.Status

	statusErrorModels := []string{"Application", "Authentication", "Endpoint"}
	if util.SliceContainsString(statusErrorModels, statusMessage.ResourceType) {
		updateAttributes["availability_status_error"] = statusMessage.Error
	}

	if statusMessage.Status == "available" {
		updateAttributes["last_available_at"] = updateAttributes["last_checked_at"]
	}

	return updateAttributes
}
