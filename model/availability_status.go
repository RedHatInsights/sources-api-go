package model

// Availability statuses.
const (
	Available          string = "available"
	InProgress         string = "in_progress"
	PartiallyAvailable string = "partially_available"
	Unavailable        string = "unavailable"
)

// ValidAvailabilityStatuses is a map containing the valid availability statuses. It has this form because a map is
// faster for these lookups.
var ValidAvailabilityStatuses = map[string]struct{}{
	Available:          {},
	InProgress:         {},
	PartiallyAvailable: {},
	Unavailable:        {},
}

// AvailabilityStatuses contains the possible valid values of the status of a source
var AvailabilityStatuses = []string{
	"",
	Available,
	InProgress,
	PartiallyAvailable,
	Unavailable,
}
