package jobs

import (
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

const (
	RETRY_MAX        = 5
	RECORD_AGE_LIMIT = -1 * 30 * time.Minute
)

type RetryCreateJob struct{}

// implementing the interface - but these functions aren't really needed since
// this is a scheduled job.
func (r *RetryCreateJob) Delay() time.Duration              { return 0 }
func (r *RetryCreateJob) Arguments() map[string]interface{} { return map[string]interface{}{} }
func (r *RetryCreateJob) Name() string                      { return "RetryCreateJob" }
func (r *RetryCreateJob) ToJSON() []byte                    { panic("not implemented") }

// run the job, using any args on the struct
func (r *RetryCreateJob) Run() error {
	// running all of this as a transaction so it is idempotent if something goes wrong.
	return dao.DB.Transaction(func(tx *gorm.DB) error {
		// find all applications with retry counter > 5 and available, update
		// retry counter to 5 so they don't get picked up again.
		result := tx.Debug().
			Model(&m.Application{}).
			Where("availability_status = ? AND retry_counter < ?", m.Available, RETRY_MAX).
			Update("retry_counter", RETRY_MAX)
		if result.Error != nil {
			l.Log.Errorf("Error updating available applications' retry counters")
			return result.Error
		}

		l.Log.Infof("Updated %v applications that became available since last run but had less retry counters", result.RowsAffected)

		// find all applications that are unavailable/null/empty
		// AND
		// created_at less than 30m ago
		// AND
		// retry counter less than configured amount
		apps := make([]m.Application, 0)
		result = tx.Debug().
			Select("id", "tenant_id", "application_type_id").
			Model(&m.Application{}).
			Where("availability_status IS DISTINCT FROM ? ", m.Available).
			Where("created_at > ?", time.Now().Add(RECORD_AGE_LIMIT)).
			Where("retry_counter < ?", RETRY_MAX).
			Scan(&apps)
		if result.Error != nil {
			l.Log.Errorf("Error listing applications that meet retry criteria")
			return result.Error
		}
		if result.RowsAffected == 0 {
			l.Log.Info("No retryable applications found - returning.")
			return nil
		}

		l.Log.Infof("Found %v Applications that need to be retried", result.RowsAffected)

		// resend messages
		for i := range apps {
			go resendCreateMessages(apps[i].ID, apps[i].ApplicationTypeID, apps[i].TenantID)
		}

		// increment retry counter on the apps we sent create messages for
		result = tx.Debug().
			Model(&apps).
			Update("retry_counter", gorm.Expr("retry_counter+1"))
		if result.Error != nil {
			l.Log.Errorf("Failed to increment retry_counter column")
			return result.Error
		}

		return nil
	})
}

// resend the messages that would have been sent out for the application.
func resendCreateMessages(applicationId, applicationTypeId, tenantId int64) {
	// checking to see if the application is "opted in" to retrying first
	optedIn, err := dao.GetMetaDataDao().ApplicationOptedIntoRetry(applicationTypeId)
	if err != nil {
		l.Log.Warnf("Failed to check if application type %v is opted in for retrying", applicationTypeId)
		return
	}
	if !optedIn {
		l.Log.Debugf("Application %v not opted into retrying, returning.", applicationId)
		return
	}

	// if we're good, load up the required fields
	app, err := dao.GetApplicationDao(&dao.RequestParams{TenantID: &tenantId}).GetByIdWithPreload(&applicationId, "Source", "Tenant", "ApplicationAuthentications")
	if err != nil {
		l.Log.Warnf("Error fetching application %v from db: %v", applicationId, err)
		return
	}

	authentications, _, err := dao.GetAuthenticationDao(&dao.RequestParams{TenantID: &app.TenantID}).ListForApplication(app.ID, 100, 0, []util.Filter{})
	if err != nil {
		l.Log.Warnf("Error listing authentications for application %v: %v", applicationId, err)
		return
	}

	// generate the forwardable headers from what we have in the tenant table
	headers := app.Tenant.GetHeadersWithGeneratedXRHID()

	// raise ALL THE EVENTS...AGAIN!
	err = service.RaiseEvent("Source.create", &app.Source, headers)
	if err != nil {
		l.Log.Warnf("Failed to raise Source.create event for source %v: %v", app.SourceID, err)
	}
	err = service.RaiseEvent("Application.create", app, headers)
	if err != nil {
		l.Log.Warnf("Failed to raise Application.create event for application %v: %v", app.ID, err)
	}
	for i := range authentications {
		err = service.RaiseEvent("Authentication.create", &authentications[i], headers)
		if err != nil {
			l.Log.Warnf("Failed to raise Authentication.create event for authentication %v: %v", authentications[i].ID, err)
		}
	}
	for i := range app.ApplicationAuthentications {
		err = service.RaiseEvent("ApplicationAuthentication.create", &app.ApplicationAuthentications[i], headers)
		if err != nil {
			l.Log.Warnf("Failed to raise ApplicationAuthentication.create event for appAuth %v: %v", app.ApplicationAuthentications[i].ID, err)
		}
	}
}
