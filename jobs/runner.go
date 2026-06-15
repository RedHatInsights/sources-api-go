package jobs

import (
	"context"

	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
)

// JobRunner is the interface for enqueueing jobs. Implementations handle
// serialization and transport (e.g., valkey list, in-memory queue for tests).
type JobRunner interface {
	Enqueue(j Job)
}

// ValkeyJobRunner implements JobRunner by pushing serialized jobs onto a
// valkey list. The background worker pops and executes them.
type ValkeyJobRunner struct{}

// NewValkeyJobRunner creates a ValkeyJobRunner.
func NewValkeyJobRunner() *ValkeyJobRunner {
	return &ValkeyJobRunner{}
}

func (r *ValkeyJobRunner) Enqueue(j Job) {
	l.Log.Infof("Submitting job %v to valkey with %v", j.Name(), j.Arguments())

	req := JobRequest{
		JobName: j.Name(),
		JobRaw:  j.ToJSON(),
	}

	data, err := req.MarshalBinary()
	if err != nil {
		l.Log.Warnf("Failed to marshal job: %v", err)

		return
	}

	err = redis.Client.Do(context.Background(), redis.Client.B().Rpush().Key(workQueue).Element(string(data)).Build()).Error()
	if err != nil {
		l.Log.Warnf("Failed to submit job: %v", err)
	}
}
