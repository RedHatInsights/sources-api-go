package jobs

import (
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RetryMax       = 5
	RecordAgeLimit = -1 * 30 * time.Minute
)

type RetryCreateJob struct{}

// implementing the interface - but these functions aren't really needed since
// this is a scheduled job.
func (r *RetryCreateJob) Delay() time.Duration              { return 0 }
func (r *RetryCreateJob) Arguments() map[string]interface{} { return map[string]interface{}{} }
func (r *RetryCreateJob) Name() string                      { return "RetryCreateJob" }
func (r *RetryCreateJob) ToJSON() []byte                    { panic("not implemented") }

// Run the job, using any args on the struct.
//
// Uses FOR UPDATE SKIP LOCKED when selecting retryable applications so that
// multiple pods running the same scheduled job concurrently each claim a
// disjoint set of rows, preventing duplicate Kafka messages.
//
// Message sending happens after the transaction commits so that events are
// only produced for rows whose retry_counter was successfully incremented.
func (r *RetryCreateJob) Run() error {
	apps := make([]m.Application, 0)

	// Phase 1: transaction to claim records with row-level locking.
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		// find all applications with retry counter < max and available, update
		// retry counter to max so they don't get picked up again.
		result := tx.Debug().
			Model(&m.Application{}).
			Where("availability_status = ? AND retry_counter < ?", m.Available, RetryMax).
			Update("retry_counter", RetryMax)
		if result.Error != nil {
			l.Log.Errorf("Error updating available applications' retry counters")
			return result.Error
		}

		l.Log.Infof("Updated %v applications that became available since last run but had less retry counters", result.RowsAffected)

		// find all applications that are unavailable/null/empty
		// AND created_at less than 30m ago
		// AND retry counter less than configured amount
		//
		// FOR UPDATE SKIP LOCKED ensures each pod locks a disjoint set of
		// rows — other pods running concurrently will skip already-locked
		// rows instead of blocking or processing them a second time.
		result = tx.Debug().
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Select("id", "tenant_id", "application_type_id").
			Model(&m.Application{}).
			Where("availability_status IS DISTINCT FROM ? ", m.Available).
			Where("created_at > ?", time.Now().Add(RecordAgeLimit)).
			Where("retry_counter < ?", RetryMax).
			Scan(&apps)
		if result.Error != nil {
			l.Log.Errorf("Error listing applications that meet retry criteria")
			return result.Error
		}

		if len(apps) == 0 {
			l.Log.Info("No retryable applications found - returning.")
			return nil
		}

		l.Log.Infof("Found %v Applications that need to be retried", len(apps))

		// increment retry counter before sending messages — once the
		// transaction commits these rows won't be selected by other pods.
		result = tx.Debug().
			Model(&apps).
			Update("retry_counter", gorm.Expr("retry_counter+1"))
		if result.Error != nil {
			l.Log.Errorf("Failed to increment retry_counter column")
			return result.Error
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Phase 2: send messages only after the transaction committed
	// successfully, so we never produce events for rows we failed to claim.
	for i := range apps {
		go resendCreateMessages(apps[i].ID, apps[i].ApplicationTypeID, apps[i].TenantID)
	}

	return nil
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
