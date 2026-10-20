package jobs

import "time"

/*
Interface to codify jobs against, mostly consisting of a few methods:

 1. Delay, how long to wait before executing. Will mostly be a method that
    returns 0 for jobs that don't need to be delayed, otherwise it'll be a
    set amount of time.

2. Arguments, for pretty logging

3. Name, for pretty logging

4. Run, what do we do?

5. ToJSON, serialize the job into a byte array for sending off to valkey
*/
type Job interface {
	// how long to wait until performing (just do a sleep)
	Delay() time.Duration
	// any arguments for said job
	Arguments() map[string]interface{}
	// pretty name to print
	Name() string
	// run the job, using any args on the struct
	Run() error
	// serialize the job into JSON
	ToJSON() []byte
}
