package jobs

import (
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

var ch chan Job

// fire up a worker, initializing the channel to read from inline
func Init() {
	// initialize the channel to be used by the job worker. The `RunWorker`
	// function just takes a channel so we can swap out the backing store for
	// the jobs for anything we like as long as it can speak through a channel
	// (maybe redis groups someday.)
	ch = make(chan Job)
	l.Log.Infof("Starting Worker for Delayed Jobs")
	go RunWorker(ch)
}

// Package level function to enqueue jobs to the job channel
func Enqueue(j Job) {
	ch <- j
}

// Runs the worker just consuming off of a channel, right now just firing off
// individual jobs in goroutines.

// TODO maybe add a choke channel to limit the amount of jobs running
// concurrently.
func RunWorker(ch chan Job) {
	for j := range ch {
		go RunJobNow(j)
	}
}

// Package level function to run a job - call this one directly instead of
// `Enqueue` if you want to run the job immediately.
func RunJobNow(j Job) {
	l.Log.Infof("Waiting %v", j.Delay())
	time.Sleep(j.Delay())

	err := j.Run()
	if err != nil {
		l.Log.Warnf("Error running job [ %v ], args [ %v ] : [ %v ]", j.Name(), j.Arguments(), err)
	}
}
