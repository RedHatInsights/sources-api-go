package service

import (
	"encoding/json"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var NotificationProducer Notifier

func init() {
	NotificationProducer = &AvailabilityStatusNotifier{}
}

type Notifier interface {
	EmitAvailabilityStatusNotification(xRhIdentity *identity.Identity, emailNotificationInfo *m.EmailNotificationInfo) error
}

type AvailabilityStatusNotifier struct {
}

var notificationTopic = config.Get().KafkaTopic("platform.notifications.ingress")

const (
	application                = "sources"
	statusEventType            = "availability-status"
	bundle                     = "console"
	notificationMessageVersion = "v1.1.0"
)

// notificationPayload and notificationMetadata are empty and
// they are used for proper formatting of final
// notification message JSON
// this struct could be extended if we need to pass
// more information to notification message
type notificationPayload struct {
}

type notificationMetadata struct {
}

type notificationEvent struct {
	Metadata notificationMetadata `json:"metadata"`
	Payload  string               `json:"payload"`
}

type notificationRecipients struct {
	OnlyAdmins            bool     `json:"only_admins"`
	IgnoreUserPreferences bool     `json:"ignore_user_preferences"`
	Users                 []string `json:"users"`
}

type notificationMessage struct {
	Version     string                   `json:"version"`
	Bundle      string                   `json:"bundle"`
	Application string                   `json:"application"`
	EventType   string                   `json:"event_type"`
	Timestamp   string                   `json:"timestamp"`
	AccountID   string                   `json:"account_id"`
	OrgId       string                   `json:"org_id"`
	Context     string                   `json:"context"`
	Events      []notificationEvent      `json:"events"`
	Recipients  []notificationRecipients `json:"recipients"`
}

func (producer *AvailabilityStatusNotifier) EmitAvailabilityStatusNotification(id *identity.Identity, emailNotificationInfo *m.EmailNotificationInfo) error {
	mgr := &kafka.Manager{Config: kafka.Config{
		KafkaBrokers:   config.Get().KafkaBrokers,
		ProducerConfig: kafka.ProducerConfig{Topic: notificationTopic},
	}}

	context, err := json.Marshal(emailNotificationInfo)
	if err != nil {
		l.Log.Warnf(`error when marshalling the email notification information: %s`, err)
		return err
	}

	payload, err := json.Marshal(&notificationPayload{})
	if err != nil {
		l.Log.Warnf(`error when marshalling the email notification payload: %s`, err)
		return err
	}

	event := notificationEvent{Metadata: notificationMetadata{}, Payload: string(payload)}

	msg := &kafka.Message{}
	err = msg.AddValueAsJSON(&notificationMessage{
		Version:     notificationMessageVersion,
		Bundle:      bundle,
		Application: application,
		EventType:   statusEventType,
		Timestamp:   time.Now().Format(time.RFC3339),
		AccountID:   id.AccountNumber,
		OrgId:       id.OrgID,
		Events:      []notificationEvent{event},
		Context:     string(context),
		Recipients:  []notificationRecipients{},
	})

	if err != nil {
		l.Log.Warnf("Failed to add struct value as json to kafka message: %s", err.Error())
		return err
	}

	err = mgr.Produce(msg)
	if err != nil {
		l.Log.Warnf("Failed to produce kafka message to emit notification: %v, error: %v", statusEventType, err)
		return err
	}

	return mgr.Producer().Close()
}

func EmitAvailabilityStatusNotification(id *identity.Identity, emailNotificationInfo *m.EmailNotificationInfo) error {
	l.Log.Infof("[tenant_id: %s][source_id: %s] Publishing status notification status changed from: %s to %s",
		emailNotificationInfo.TenantID,
		emailNotificationInfo.SourceID,
		emailNotificationInfo.PreviousAvailabilityStatus,
		emailNotificationInfo.CurrentAvailabilityStatus)

	return NotificationProducer.EmitAvailabilityStatusNotification(id, emailNotificationInfo)
}
