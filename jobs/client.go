package jobs

import (
	"encoding/json"
	"fmt"

	l "github.com/RedHatInsights/sources-api-go/logger"
)

type JobRequest struct {
	JobName string
	JobRaw  []byte
	Job     Job
}

// MarshalBinary implements the encoding.BinaryMarshaler interface for valkey encoding.
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

// Enqueue adds a job to the worker queue. It is a function variable so
// that tests can swap it with a mock and production code can delegate to
// a ValkeyJobRunner. The default uses ValkeyJobRunner.
var Enqueue func(j Job) = func(j Job) {
	NewValkeyJobRunner().Enqueue(j)
}
