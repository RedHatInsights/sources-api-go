package mocks

import (
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type MockAvailabilityStatusNotificationProducer struct {
	EmitAvailabilityStatusCallCounter int
	AccountNumber                     string
	OrgId                             string
	EmailNotificationInfo             *m.EmailNotificationInfo
}

func (producer *MockAvailabilityStatusNotificationProducer) EmitAvailabilityStatusNotification(id *identity.Identity, emailNotificationInfo *m.EmailNotificationInfo) error {
	producer.EmitAvailabilityStatusCallCounter++
	producer.EmailNotificationInfo = emailNotificationInfo
	producer.AccountNumber = id.AccountNumber
	producer.OrgId = id.OrgID
	return nil
}
