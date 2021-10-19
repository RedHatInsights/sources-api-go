package model

// Availability statuses.
const (
	Available          string = "available"
	InProgress         string = "in_progress"
	PartiallyAvailable string = "partially_available"
	Unavailable        string = "unavailable"
)

// AvailabilityStatuses contains the possible valid values of the status of a source
var AvailabilityStatuses = []string{
	"",
	Available,
	InProgress,
	PartiallyAvailable,
	Unavailable,
}
