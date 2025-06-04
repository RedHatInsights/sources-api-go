package statuslistener

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
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
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

const (
	sourcesStatusRequestedTopic = "platform.sources.status"
	groupID                     = "sources-api-status-worker"
	eventAvailabilityStatus     = "availability_status"
	eventHealthcheck            = "healthcheck"
	healthCheckInterval         = 30
)

var (
	config             = c.Get()
	sourcesStatusTopic = config.KafkaTopic(sourcesStatusRequestedTopic)
)

type AvailabilityStatusListener struct {
	*events.EventStreamProducer
	healthcheck chan struct{}
	lastMsg     time.Time
}

func Run(shutdown chan struct{}) {
	l.Log.Infof("Starting Availability Status Listener on topic [%v]", sourcesStatusTopic)

	avs := AvailabilityStatusListener{
		EventStreamProducer: NewEventStreamProducer(),
		healthcheck:         make(chan struct{}, 10),
	}
	avs.subscribeToAvailabilityStatus(shutdown)
}

func NewEventStreamProducer() *events.EventStreamProducer {
	sender := &events.EventStreamSender{}
	return &events.EventStreamProducer{Sender: sender}
}

func (avs *AvailabilityStatusListener) subscribeToAvailabilityStatus(shutdown chan struct{}) {
	if l.Log == nil {
		panic("logging is not initialized")
	}

	kf, err := kafka.GetReader(&kafka.Options{
		BrokerConfig: config.KafkaBrokerConfig,
		GroupID:      util.StringRef(groupID),
		Topic:        sourcesStatusTopic,
		Logger:       l.Log,
	})
	if err != nil {
		l.Log.Errorf(`could not get a Kafka reader: %s`, err)
		return
	}

	// run async for graceful shutdown handling
	go kafka.Consume(kf, avs.ConsumeStatusMessage)
	go avs.Healthcheck()

	// let the healthcheck thread know we are good to go since we connected successfully
	avs.healthcheck <- struct{}{}

	<-shutdown
	kafka.CloseReader(kf, "subscribe availability status. Shutdown signal received")
	shutdown <- struct{}{}
}

func (avs *AvailabilityStatusListener) ConsumeStatusMessage(message kafka.Message) {
	l.Log.WithFields(logrus.Fields{
		"kafka_headers":     message.Headers,
		"kafka_message":     message.Value,
		"kafka_message_key": message.Key,
	}).Tracef("Kafka message received")

	// if it's a healthcheck message it isn't really an invalid type, but we
	// don't need to do anything with it
	if message.GetHeader("event_type") == eventHealthcheck {
		return
	}

	if message.GetHeader("event_type") != eventAvailabilityStatus {
		l.Log.Warnf("Skipping invalid event_type %q", message.GetHeader("event_type"))
		return
	}

	var statusMessage types.StatusMessage
	err := message.ParseTo(&statusMessage)
	if err != nil {
		l.Log.Errorf("Error in parsing status message %v", err)
		return
	}

	// parse the resource_id field - which can be either an integer or string
	id, err := util.InterfaceToString(statusMessage.ResourceIDRaw)
	if err != nil {
		l.Log.Errorf("Invalid ID Passed: %v", statusMessage.ResourceIDRaw)
	}
	statusMessage.ResourceID = id

	headers := message.TranslateHeaders()
	avs.processEvent(statusMessage, headers)
}

func (avs *AvailabilityStatusListener) processEvent(statusMessage types.StatusMessage, headers []kafka.Header) {
	l.Log.WithFields(logrus.Fields{
		"headers":       headers,
		"resource_type": statusMessage.ResourceType,
		"resource_id":   statusMessage.ResourceIDRaw,
		"status":        statusMessage.Status,
		"error":         statusMessage.Error,
	}).Debugf("Status message received")

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

	id, err := util.IdentityFromKafkaHeaders(headers)
	if err != nil {
		l.Log.Error(err)
		return
	}

	tenantDao := dao.GetTenantDao()
	tenant, err := tenantDao.TenantByIdentity(id)
	if err != nil {
		l.Log.Error(err)
		return
	}

	if id.OrgID == "" {
		id.OrgID = tenant.OrgID
	}

	updateAttributes := avs.attributesForUpdate(statusMessage)
	modelEventDao, err := dao.GetFromResourceType(statusMessage.ResourceType, tenant.Id)
	if err != nil {
		l.Log.Error(err)
		return
	}

	previousStatus, err := dao.GetAvailabilityStatusFromStatusMessage(tenant.Id, statusMessage.ResourceID, statusMessage.ResourceType)
	if err != nil {
		l.Log.Errorf("[tenant_id: %d][resource_type: %s][resource_id: %s] unable to get status availability: %s", tenant.Id, statusMessage.ResourceType, statusMessage.ResourceID, err)
		return
	}

	resource.TenantID = tenant.Id
	resource.AccountNumber = tenant.ExternalTenant
	resultRecord, err := modelEventDao.FetchAndUpdateBy(*resource, updateAttributes)
	if err != nil {
		l.Log.Errorf("[tenant_id: %d][resource_type: %s][resource_id: %d][resource_uuid: %s] unable to update availability status: %s", resource.TenantID, resource.ResourceType, resource.ResourceID, resource.ResourceUID, err)
		return
	}

	if previousStatus != statusMessage.Status {
		if statusMessage.ResourceType == "Application" {
			appDao := dao.GetApplicationDao(&dao.RequestParams{TenantID: &tenant.Id})
			app, err := appDao.GetById(&resource.ResourceID)
			if err != nil {
				l.Log.Errorf("[tenant_id: %d][application_id: %d] unable to fetch application: %s", tenant.Id, resource.ResourceID, err)
				return
			}

			err = service.UpdateSourceFromApplicationAvailabilityStatus(app, previousStatus)
			if err != nil {
				return
			}
		}

		emailInfo, ok := resultRecord.(m.EmailNotification)
		if !ok {
			l.Log.Errorf("error in type assert of %v", resultRecord)
		}

		if emailInfo != nil {
			err = service.EmitAvailabilityStatusNotification(id, emailInfo.ToEmail(previousStatus), "status-listener")
			if err != nil {
				l.Log.Errorf("[tenant_id: %d][resource_type: %s][resource_id: %d][resource_uuid: %s] unable to emit notification: %v", resource.TenantID, resource.ResourceType, resource.ResourceID, resource.ResourceUID, err)
			}
		}
	}

	updateAttributeKeys := make([]string, 0)
	for k := range updateAttributes {
		updateAttributeKeys = append(updateAttributeKeys, k)

	}
	sort.Strings(updateAttributeKeys)

	err = avs.RaiseEventForUpdate(modelEventDao, *resource, updateAttributeKeys, headers)

	if err != nil {
		l.Log.Errorf("Error in raising event for update: %s, resource: %s(%s)", err.Error(), statusMessage.ResourceType, statusMessage.ResourceID)
	}
}

var statusErrorModels = []string{"application", "authentication", "endpoint"}

func (avs *AvailabilityStatusListener) attributesForUpdate(statusMessage types.StatusMessage) map[string]interface{} {
	updateAttributes := make(map[string]interface{})

	// TODO: const this? are we using it elsewhere?
	updateAttributes["last_checked_at"] = time.Now().Format("2006-01-02T15:04:05.999Z")
	updateAttributes["availability_status"] = statusMessage.Status

	if util.SliceContainsString(statusErrorModels, strings.ToLower(statusMessage.ResourceType)) {
		updateAttributes["availability_status_error"] = statusMessage.Error
	}

	if statusMessage.Status == "available" {
		updateAttributes["last_available_at"] = updateAttributes["last_checked_at"]
	}

	return updateAttributes
}

func (avs *AvailabilityStatusListener) Healthcheck() {
	// run the healthcheck consumer/producer in the background
	go avs.healthCheckProducer()
	go avs.healthCheckConsumer()

	// using echo rather than stdlib so we can have many independent endpoints
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/health", func(c echo.Context) error {
		// /health should return a 500 if there hasn't been a message yet OR it
		// has been more than <healthcheckinterval> seconds since the last
		// successful kafka message production
		if avs.lastMsg.IsZero() || avs.lastMsg.Before(time.Now().Add(-healthCheckInterval*time.Second)) {
			errstr := fmt.Sprintf("no successful kafka production in %v seconds or more", healthCheckInterval)
			l.Log.Warn(errstr)
			return c.String(http.StatusInternalServerError, errstr)
		}

		// we're good!
		return c.NoContent(204)
	})
	l.Log.Fatal(e.Start(":8000"))
}

// healthCheckConsumer runs a loop against the availability status listener's
// healthcheck channel and updates the shared timestamp with the last time we
// got a message
func (avs *AvailabilityStatusListener) healthCheckConsumer() {
	for range avs.healthcheck {
		now := time.Now()
		avs.lastMsg = now
		l.Log.Debugf("Got Healthcheck Messsage %v", now.Format(time.Kitchen))
	}
}

// healthCheckProducer produces a message on the platform.sources.status topic
// with the event_type of "healthcheck" so that we know that kafka is working
// even when we don't have any traffic coming through
func (avs *AvailabilityStatusListener) healthCheckProducer() {
	// send a message every <healthCheckInterval> seconds
	for range time.NewTicker(healthCheckInterval * time.Second).C {
		w, err := kafka.GetWriter(&kafka.Options{
			BrokerConfig: config.KafkaBrokerConfig,
			Topic:        sourcesStatusTopic,
			Logger:       l.Log,
		})
		if err != nil {
			l.Log.Warn(err)
			return
		}

		msg := kafka.Message{}
		msg.AddHeaders([]kafka.Header{
			{Key: "event_type", Value: []byte(eventHealthcheck)},
		})
		now := time.Now()

		// should be 0, since we have produced 0 messages.
		before := w.Stats().Writes

		l.Log.Debugf("Producing Healthcheck message %v", now.Format(time.Kitchen))
		err = kafka.Produce(w, &msg)
		if err != nil {
			l.Log.Warnf("Failed to produce healthcheck msg at %v", now.Format(time.Kitchen))
		}

		// should be 1, since we successfully produced a message. Otherwise
		// it'll be the same and we don't send the signal to the consumer
		// routine.
		after := w.Stats().Writes

		// send the message if and only if we successfully wrote a message
		if after > before {
			avs.healthcheck <- struct{}{}
		}

		kafka.CloseWriter(w, "healthcheck producer")
	}
}
