package statuslistener

import (
	"sort"
	"strings"
	"time"

	c "github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/types"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const (
	sourcesStatusTopic      = "platform.sources.status"
	groupID                 = "sources-api-status-worker"
	eventAvailabilityStatus = "availability_status"
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
	if l.Log == nil {
		panic("logging is not initialized")
	}

	kafkaConfig := kafka.Config{
		KafkaBrokers: config.KafkaBrokers,
		ConsumerConfig: kafka.ConsumerConfig{
			Topic:   sourcesStatusTopic,
			GroupID: groupID},
	}

	kf := &kafka.Manager{Config: kafkaConfig}

	l.Log.Infof("Started Availability Status Listener, listening on [%v]", kafkaConfig.ConsumerConfig.Topic)
	kf.Consume(avs.ConsumeStatusMessage)
}

func (avs *AvailabilityStatusListener) ConsumeStatusMessage(message kafka.Message) {
	var statusMessage types.StatusMessage
	err := message.ParseTo(&statusMessage)
	if err != nil {
		l.Log.Errorf("Error in parsing status message %v", err)
		return
	}

	if message.GetHeader("event_type") != eventAvailabilityStatus {
		l.Log.Warnf("Skipping invalid event_type %q", message.GetHeader("event_type"))
		return
	}

	l.Log.Infof("Kafka message %s, %s received with payload: %s", message.Headers, message.Key, message.Value)

	headers := avs.headersFrom(message)

	avs.processEvent(statusMessage, headers)
}

func (avs *AvailabilityStatusListener) headersFrom(message kafka.Message) []kafka.Header {
	if len(message.Headers) < 1 {
		return []kafka.Header{}
	}

	headers := make([]kafka.Header, len(message.Headers))
	for index, header := range message.Headers {
		headers[index] = kafka.Header{Key: header.Key, Value: header.Value}
	}

	return headers
}

func (avs *AvailabilityStatusListener) processEvent(statusMessage types.StatusMessage, headers []kafka.Header) {
	resource := &util.Resource{}
	resource, err := util.ParseStatusMessageToResource(resource, statusMessage)
	if err != nil {
		l.Log.Errorf("Invalid Status: %s", statusMessage.Status)
		return
	}

	if !util.SliceContainsString(m.AvailabilityStatuses, statusMessage.Status) {
		l.Log.Errorf("Invalid Status: %s", statusMessage.Status)
		return
	}

	updateAttributes := avs.attributesForUpdate(statusMessage)
	modelEventDao, err := dao.GetFromResourceType(statusMessage.ResourceType)
	if err != nil {
		l.Log.Error(err)
		return
	}

	accountNumber, err := util.AccountNumberFromHeaders(headers)
	if err != nil {
		l.Log.Error(err)
		return
	}

	tenant, err := dao.TenantByAccountNumber(accountNumber)
	if err != nil {
		l.Log.Error(err)
		return
	}

	resource.TenantID = tenant.Id
	resource.AccountNumber = tenant.ExternalTenant
	err = (*modelEventDao).FetchAndUpdateBy(*resource, updateAttributes)

	if err != nil {
		l.Log.Errorf("Update error in status availability: %s", err)
		return
	}

	updateAttributeKeys := make([]string, 0)
	for k := range updateAttributes {
		updateAttributeKeys = append(updateAttributeKeys, k)

	}
	sort.Strings(updateAttributeKeys)

	err = avs.EventStreamProducer.RaiseEventForUpdate(*resource, updateAttributeKeys, headers)

	if err != nil {
		l.Log.Errorf("Error in raising event for update: %s, resource: %s(%s)", err.Error(), statusMessage.ResourceType, statusMessage.ResourceID)
	}
}

func (avs *AvailabilityStatusListener) attributesForUpdate(statusMessage types.StatusMessage) map[string]interface{} {
	updateAttributes := make(map[string]interface{})

	// TODO: const this? are we using it elsewhere?
	updateAttributes["last_checked_at"] = time.Now().Format("2006-01-02T15:04:05.999Z")
	updateAttributes["availability_status"] = statusMessage.Status

	statusErrorModels := []string{"application", "authentication", "endpoint"}
	if util.SliceContainsString(statusErrorModels, strings.ToLower(statusMessage.ResourceType)) {
		updateAttributes["availability_status_error"] = statusMessage.Error
	}

	if statusMessage.Status == "available" {
		updateAttributes["last_available_at"] = updateAttributes["last_checked_at"]
	}

	return updateAttributes
}
