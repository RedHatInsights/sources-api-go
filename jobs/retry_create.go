package jobs

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RetryMax       = 5
	RecordAgeLimit = -1 * 30 * time.Minute

	// DefaultRetryIntervalMinutes is the default interval (in minutes) between
	// retry job runs.  Override with RETRY_CREATE_JOB_INTERVAL_MINUTES.
	DefaultRetryIntervalMinutes = 2

	// retryBatchSize limits how many rows are locked and processed per chunk,
	// keeping transaction durations short and lock scope narrow.
	retryBatchSize = 100

	// maxRetryConcurrency caps the number of goroutines sending retry messages
	// concurrently.
	maxRetryConcurrency = 5
)

// Prometheus metrics for observability into the retry pipeline.
var (
	retryCreateProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_retry_create_processed_total",
		Help: "Total number of applications processed for retry create",
	})
	retryCreateMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_retry_create_messages_sent_total",
		Help: "Total number of retry create Kafka messages sent successfully",
	})
	retryCreateErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sources_retry_create_errors_total",
		Help: "Total number of errors encountered during retry create processing",
	})
)

// RetryCreateJobInterval reads the configurable interval from the
// RETRY_CREATE_JOB_INTERVAL_MINUTES environment variable, falling back to
// DefaultRetryIntervalMinutes.
func RetryCreateJobInterval() time.Duration {
	if v := os.Getenv("RETRY_CREATE_JOB_INTERVAL_MINUTES"); v != "" {
		if mins, err := strconv.Atoi(v); err == nil && mins > 0 {
			return time.Duration(mins) * time.Minute
		}
	}
	return time.Duration(DefaultRetryIntervalMinutes) * time.Minute
}

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
// Records are processed in chunks (retryBatchSize) with ORDER BY created_at
// to keep lock scope narrow and transaction durations short.
//
// Message sending happens after each transaction commits so that events are
// only produced for rows whose retry_counter was successfully incremented.
// Concurrency is capped at maxRetryConcurrency goroutines.
func (r *RetryCreateJob) Run() error {
	// Reset retry counters for applications that became available since the
	// last run — marks them as "done" so they won't be retried again.
	if err := resetAvailableRetryCounters(); err != nil {
		retryCreateErrors.Inc()
		return err
	}

	// Process retryable applications in chunks to avoid locking too many
	// rows at once.
	var totalProcessed int64
	for {
		apps, err := claimRetryBatch()
		if err != nil {
			retryCreateErrors.Inc()
			return err
		}
		if len(apps) == 0 {
			break
		}

		totalProcessed += int64(len(apps))
		sendRetryMessages(apps)
	}

	if totalProcessed == 0 {
		l.Log.Info("No retryable applications found - returning.")
	} else {
		l.Log.Infof("Processed %d applications for retry", totalProcessed)
	}

	return nil
}

// resetAvailableRetryCounters sets the retry counter to RetryMax for all
// applications that became available, ensuring they are not retried again.
func resetAvailableRetryCounters() error {
	result := dao.DB.Debug().
		Model(&m.Application{}).
		Where("availability_status = ? AND retry_counter < ?", m.Available, RetryMax).
		Update("retry_counter", RetryMax)
	if result.Error != nil {
		l.Log.Errorf("Error updating available applications' retry counters")
		return result.Error
	}

	if result.RowsAffected > 0 {
		l.Log.Infof("Updated %v applications that became available since last run but had less retry counters", result.RowsAffected)
	}

	return nil
}

// claimRetryBatch selects up to retryBatchSize retryable applications inside a
// transaction with FOR UPDATE SKIP LOCKED, increments their retry_counter, and
// returns the claimed rows.  Returns an empty slice when no more rows qualify.
func claimRetryBatch() ([]m.Application, error) {
	var apps []m.Application

	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE SKIP LOCKED ensures each pod locks a disjoint set of
		// rows — other pods running concurrently will skip already-locked
		// rows instead of blocking or processing them a second time.
		result := tx.Debug().
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "", Alias: "", Raw: false}, Options: "SKIP LOCKED"}).
			Select("id", "tenant_id", "application_type_id").
			Model(&m.Application{}).
			Where("availability_status IS DISTINCT FROM ? ", m.Available).
			Where("created_at > ?", time.Now().Add(RecordAgeLimit)).
			Where("retry_counter < ?", RetryMax).
			Order("created_at ASC").
			Limit(retryBatchSize).
			Scan(&apps)
		if result.Error != nil {
			l.Log.Errorf("Error listing applications that meet retry criteria")
			return result.Error
		}

		if len(apps) == 0 {
			return nil
		}

		l.Log.Infof("Found %v applications that need to be retried", len(apps))

		// Extract IDs for a targeted update — avoids passing a slice to
		// Model() which can cause unexpected GORM behavior.
		ids := make([]int64, len(apps))
		for i := range apps {
			ids[i] = apps[i].ID
		}

		result = tx.Debug().
			Model(&m.Application{}).
			Where("id IN ?", ids).
			Update("retry_counter", gorm.Expr("retry_counter + 1"))
		if result.Error != nil {
			l.Log.Errorf("Failed to increment retry_counter column")
			return result.Error
		}

		return nil
	})

	return apps, err
}

// sendRetryMessages fans out message-sending goroutines for the given
// applications, capping concurrency at maxRetryConcurrency.
func sendRetryMessages(apps []m.Application) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxRetryConcurrency)

	for i := range apps {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore slot

		go func(app m.Application) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore slot

			resendCreateMessages(app.ID, app.ApplicationTypeID, app.TenantID)
			retryCreateProcessed.Inc()
		}(apps[i])
	}

	wg.Wait()
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
	} else {
		retryCreateMessagesSent.Inc()
	}

	err = service.RaiseEvent("Application.create", app, headers)
	if err != nil {
		l.Log.Warnf("Failed to raise Application.create event for application %v: %v", app.ID, err)
	} else {
		retryCreateMessagesSent.Inc()
	}

	for i := range authentications {
		err = service.RaiseEvent("Authentication.create", &authentications[i], headers)
		if err != nil {
			l.Log.Warnf("Failed to raise Authentication.create event for authentication %v: %v", authentications[i].ID, err)
		} else {
			retryCreateMessagesSent.Inc()
		}
	}

	for i := range app.ApplicationAuthentications {
		err = service.RaiseEvent("ApplicationAuthentication.create", &app.ApplicationAuthentications[i], headers)
		if err != nil {
			l.Log.Warnf("Failed to raise ApplicationAuthentication.create event for appAuth %v: %v", app.ApplicationAuthentications[i].ID, err)
		} else {
			retryCreateMessagesSent.Inc()
		}
	}
}
