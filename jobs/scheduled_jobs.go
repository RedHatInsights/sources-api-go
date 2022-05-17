package jobs

import (
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

// ScheduledJob is a struct which stores any jobs we want ran on a consistent
// schedule.
//
// it has 2 fields:
// 		Interval: how often to run the job
// 		Job: self-explanatory.
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

// schedule is a slice of scheduled jobs that will then get ran forever, if we
// are adding a new job that we want run on a schedule, add it here.
//
// example: var schedule = []ScheduledJob{{Interval: 5 * time.Second, Job: &AsyncDestroyJob{}}}
var schedule = []ScheduledJob{
	// scheduled job that runs every 2 minutes and re-sends any unavailable
	// sources that haven't ever went available
	{Interval: 2 * time.Minute, Job: &RetryCreateJob{}},
}

// runScheduledJobs runs all of the jobs on a schedule forever.
func runScheduledJobs() {
	l.Log.Infof("Running [%v] Background Job goroutines", len(schedule))

	for _, sj := range schedule {
		l.Log.Infof("Running Job [%v] on Interval [%v]", sj.Job.Name(), sj.Interval)
		sj.runForever()
	}
}
