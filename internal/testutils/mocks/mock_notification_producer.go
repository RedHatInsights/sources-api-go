package mocks

import m "github.com/RedHatInsights/sources-api-go/model"

type MockAvailabilityStatusNotificationProducer struct {
	EmitAvailabilityStatusCallCounter int
	AccountNumber                     string
	EmailNotificationInfo             *m.EmailNotificationInfo
}

func (producer *MockAvailabilityStatusNotificationProducer) EmitAvailabilityStatusNotification(accountNumber string, emailNotificationInfo *m.EmailNotificationInfo) error {
	producer.EmitAvailabilityStatusCallCounter++
	producer.EmailNotificationInfo = emailNotificationInfo
	producer.AccountNumber = accountNumber
	return nil
}
