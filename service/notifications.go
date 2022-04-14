package service

import (
	"encoding/json"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
)

var NotificationProducer Notifier

func init() {
	NotificationProducer = &AvailabilityStatusNotifier{}
}

type Notifier interface {
	EmitAvailabilityStatusNotification(accountNumber string, emailNotificationInfo *m.EmailNotificationInfo) error
}

type AvailabilityStatusNotifier struct {
}

var (
	application                = "sources"
	notificationTopic          = config.Get().KafkaTopic("platform.notifications.ingress")
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
	Metadata string `json:"metadata"`
	Payload  string `json:"payload"`
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
	Context     string                   `json:"context"`
	Events      []notificationEvent      `json:"events"`
	Recipients  []notificationRecipients `json:"recipients"`
}

func (producer *AvailabilityStatusNotifier) EmitAvailabilityStatusNotification(accountNumber string, emailNotificationInfo *m.EmailNotificationInfo) error {
	mgr := &kafka.Manager{Config: kafka.Config{
		KafkaBrokers:   config.Get().KafkaBrokers,
		ProducerConfig: kafka.ProducerConfig{Topic: notificationTopic},
	}}

	context, err := json.Marshal(emailNotificationInfo)
	if err != nil {
		l.Log.Warnf("error in marshalling context ")
		return err
	}

	payload, err := json.Marshal(&notificationPayload{})
	if err != nil {
		l.Log.Warnf("error in marshalling payload ")
		return err
	}

	metadata, err := json.Marshal(&notificationMetadata{})
	if err != nil {
		l.Log.Warnf("error in marshalling payload ")
		return err
	}

	event := notificationEvent{Metadata: string(metadata), Payload: string(payload)}

	msg := &kafka.Message{}
	err = msg.AddValueAsJSON(&notificationMessage{
		Version:     notificationMessageVersion,
		Bundle:      bundle,
		Application: application,
		EventType:   statusEventType,
		Timestamp:   time.Now().Format(time.RFC3339),
		AccountID:   accountNumber,
		Events:      []notificationEvent{event},
		Context:     string(context),
		Recipients:  []notificationRecipients{},
	})

	if err != nil {
		l.Log.Warnf("Failed to add struct value as json to kafka message")
		return err
	}

	err = mgr.Produce(msg)
	if err != nil {
		l.Log.Warnf("Failed to produce kafka message to emit notification: %v, error: %v", statusEventType, err)
		return err
	}

	return nil
}

func EmitAvailabilityStatusNotification(accountNumber string, emailNotificationInfo *m.EmailNotificationInfo) error {
	return NotificationProducer.EmitAvailabilityStatusNotification(accountNumber, emailNotificationInfo)
}
