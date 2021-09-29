package model

// Availability statuses.
const (
	Available          string = "available"
	InProgress         string = "in_progress"
	PartiallyAvailable string = "partially_available"
	Unavailable        string = "unavailable"
)

// A package level defined slice to avoid instantiating it every time.
var availabilityStatuses = []string{
	"",
	Available,
	InProgress,
	PartiallyAvailable,
	Unavailable,
}
