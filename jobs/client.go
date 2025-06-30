package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
)

type JobRequest struct {
	JobName string
	JobRaw  []byte
	Job     Job
}

// implementing binary mashaler/unmarshaler interfaces for redis encoding/decoding.
func (jr JobRequest) MarshalBinary() (data []byte, err error) {
	return json.Marshal(&jr)
}
func (jr *JobRequest) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, jr)
}

func (jr *JobRequest) Parse() error {
	switch jr.JobName {
	case "SuperkeyDestroyJob":
		sdj := SuperkeyDestroyJob{}

		err := json.Unmarshal([]byte(jr.JobRaw), &sdj)
		if err != nil {
			return err
		}

		jr.Job = &sdj
	case "AsyncDestroyJob":
		adj := AsyncDestroyJob{}

		err := json.Unmarshal(jr.JobRaw, &adj)
		if err != nil {
			return err
		}

		jr.Job = &adj
	default:
		l.Log.Warnf("Unsupported job: %v", jr.JobName)
		return fmt.Errorf("unsupported job %v", jr.JobName)
	}

	l.Log.Debugf("Successfully parsed job %v, args %v", jr.Job.Name(), jr.Job.Arguments())

	return nil
}

// Throws a `job` on the redis list to be picked up by the worker
func Enqueue(j Job) {
	l.Log.Infof("Submitting job %v to redis with %v", j.Name(), j.Arguments())

	req := JobRequest{
		JobName: j.Name(),
		JobRaw:  j.ToJSON(),
	}

	err := redis.Client.RPush(context.Background(), workQueue, req).Err()
	if err != nil {
		l.Log.Warnf("Failed to submit job: %v", err)
	}
}
