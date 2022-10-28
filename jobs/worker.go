package jobs

import (
	"context"
	"errors"
	"net/http"
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/labstack/echo/v4"
)

// the queue on redis we'll be sending the jobs to
const workQueue = "sources_api_jobs"

// Runs the worker just consuming off of a redis list
func Run(shutdown chan struct{}) {
	l.Log.Infof("Starting up Background worker listening to redis queue [%v]", workQueue)

	go func() {
		runScheduledJobs()

		for {
			// This is a BLocking Pop that will effectively wait forever until
			// something gets queued.
			//
			// Once it does succeed - we check to see if the error is just
			// `redis.Nil`, in which case things are fine and we continue.
			val, err := redis.Client.BLPop(context.Background(), 0, workQueue).Result()
			if err != nil {
				if !errors.Is(err, redis.Nil) {
					l.Log.Warnf("Failed to pop job from queue: %v", err)
				}

				continue
			}

			jr := JobRequest{}
			// the val that is returned is a slice in the form [listname, value]
			// where val[0] is the name of the list (e.g. workQueue above) and
			// val[1] is the string value, e.g. the output from
			// jobRequest.MarshalBinary. So we need to unmarshal it.
			err = jr.UnmarshalBinary([]byte(val[1]))
			if err != nil {
				l.Log.Warnf("Failed to unmarshal job from redis: %v", err)
				continue
			}

			// Parse is the _very_ important method that looks at the job name and
			// figures out what kind of job to unmarshal
			err = jr.Parse()
			if err != nil {
				l.Log.Warnf("Failed to unmarshal job from redis: %v", err)
				continue
			}

			RunJobAsync(jr.Job)
		}
	}()

	//create struct for healthcheck struct and run the healthcheck thread
	healthCheck := BackgroundWorkerHealthChecker{timeStamp: time.Now(), timeChannel: make(chan time.Time, 10)}
	go healthCheck.healthChecker()

	<-shutdown
	shutdown <- struct{}{}

}

// Run a Job with a delay but in the background so it does _not_ block the
// calling routine.
func RunJobAsync(j Job) {
	l.Log.Infof("Running job asyncronously %v, %v", j.Name(), j.Arguments())

	go func() {
		RunJob(j)
	}()
}

// Run a job with a delay in front. This will block the calling thread. If you
// do not want to block use `RunJobAsync`
func RunJob(j Job) {
	l.Log.Infof("Waiting %v for %v, %v", j.Delay(), j.Name(), j.Arguments())
	time.Sleep(j.Delay())

	RunJobNow(j)
}

// Package level function to run a job - call this one directly instead of
// `Enqueue` if you want to run the job immediately.
func RunJobNow(j Job) {
	l.Log.Infof("Running Job %v with %v", j.Name(), j.Arguments())
	err := j.Run()
	if err != nil {
		l.Log.Warnf("Error running job [ %v ], args [ %v ] : [ %v ]", j.Name(), j.Arguments(), err)
		return
	}
	l.Log.Infof("Finished Job %v with %v", j.Name(), j.Arguments())
}

// BackgroundWorkerHealthChecker is a struct that contains the time for the health checker and a channel to pass the shared memory for the time through time channel
type BackgroundWorkerHealthChecker struct {
	timeStamp   time.Time
	timeChannel chan time.Time
}

func (hc *BackgroundWorkerHealthChecker) healthChecker() {
	//run healthcheck loop and consumer in the background
	go hc.healthcheckLoop()
	go hc.healthCheckConsumer()

	//create new GET request through echo with the same path name "/health"
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/health", func(c echo.Context) error {
		//health should return a 500 if there hasn't been a response from ping request within 30 seconds
		if hc.timeStamp.Before(time.Now().Add(-30 * time.Second)) {
			return c.String(http.StatusInternalServerError, "Failed to hit redis for more than 30 seconds.")
		}
		return c.String(http.StatusOK, "OK")
	})

	l.Log.Fatal(e.Start(":8000"))
}

// healthcheckConsumer runs a loop against the healthcheck time channel and updates the shared timestamp with the last time we received a response
func (hc *BackgroundWorkerHealthChecker) healthCheckConsumer() {
	for range hc.timeChannel {
		hc.timeStamp = time.Now()
	}
}

func (hc *BackgroundWorkerHealthChecker) healthcheckLoop() {
	//every 30 seconds, send a ping request to redis server. If ping request returns a <nil> error, then update the time
	for range time.NewTicker(30 * time.Second).C {
		err := redis.Client.Ping(context.Background()).Err()
		if err == nil {
			hc.timeChannel <- time.Now()
		} else {
			l.Log.Warnf("Failed to hit redis: %v", err)
		}
	}
}
