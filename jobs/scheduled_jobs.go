package jobs

import (
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

// ScheduledJob is a struct which stores any jobs we want ran on a consistent
// schedule.
//
// it has 2 fields:
//
//	Interval: how often to run the job
//	Job: self-explanatory.
type ScheduledJob struct {
	Interval time.Duration
	Job      Job
}

// runForever is a method on the scheduled job that basically runs a sleep + run
// loop forever. This way one can just call `runForever()` on a job and it will
// do as the name implies
func (sj *ScheduledJob) runForever() {
	go func() {
		for {
			time.Sleep(sj.Interval)

			RunJobNow(sj.Job)
		}
	}()
}

// buildSchedule returns the list of scheduled jobs, reading configurable
// intervals from the environment so that values like the retry-create interval
// can be tuned via app-interface without a code change.
func buildSchedule() []ScheduledJob {
	return []ScheduledJob{
		// Scheduled job that re-sends create events for unavailable sources.
		// Interval is configurable via RETRY_CREATE_JOB_INTERVAL_MINUTES
		// (default: 2 minutes).
		{Interval: RetryCreateJobInterval(), Job: &RetryCreateJob{}},
	}
}

// runScheduledJobs runs all of the jobs on a schedule forever.
func runScheduledJobs() {
	schedule := buildSchedule()

	l.Log.Infof("Running [%v] Background Job goroutines", len(schedule))

	for _, sj := range schedule {
		l.Log.Infof("Running Job [%v] on Interval [%v]", sj.Job.Name(), sj.Interval)
		sj.runForever()
	}
}
