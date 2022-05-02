package service

import (
	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
)

func UpdateSourceFromApplicationAvailabilityStatus(application *m.Application, previousStatus string) error {
	if application == nil {
		return nil
	}

	if previousStatus != application.AvailabilityStatus {
		source := &m.Source{}
		source.ID = application.SourceID
		source.AvailabilityStatus = application.AvailabilityStatus
		if application.LastCheckedAt != nil && !application.LastCheckedAt.IsZero() {
			source.LastCheckedAt = application.LastCheckedAt
		}

		if application.LastAvailableAt != nil && !application.LastAvailableAt.IsZero() {
			source.LastAvailableAt = application.LastAvailableAt
		}

		sourceDao := dao.GetSourceDao(&application.TenantID)
		err := sourceDao.Update(source)
		if err != nil {
			l.Log.Errorf("unable to load source: %v", err.Error())
			return err
		}
	}

	return nil
}
