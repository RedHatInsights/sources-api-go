package metrics

// availabilityRequestOutcome represents the outcome of the requested availability check.
type availabilityRequestOutcome int

const (
	// resultSuccess signals that the availability request succeeded.
	resultSuccess availabilityRequestOutcome = iota
	// resultFailure signals that the availability request failed.
	resultFailure availabilityRequestOutcome = iota
)

// outcomeName is a helper map that can be used by implementers of the interface to use a unified set of label values.
var outcomeName = map[availabilityRequestOutcome]string{
	resultSuccess: "success",
	resultFailure: "failure",
}

// ErrorOrigin represents where the error occurred.
type ErrorOrigin int

const (
	// OriginInternal signals that the error originated internally, and therefore we were not able to perform the
	// availability check request due to internal reasons.
	OriginInternal ErrorOrigin = iota
	// OriginExternal signals that the error originated externally, and therefore we were not able to perform the
	// availability check request due to external reasons.
	OriginExternal ErrorOrigin = iota
)

// originName is a helper map that can be used by implementers of the interface to use a unified set of label values.
var originName = map[ErrorOrigin]string{
	OriginInternal: "internal",
	OriginExternal: "external",
}

// MetricsService declares the universal methods that should exist to handle our metrics, regardless of the underlying
// metrics backend that is used.
type MetricsService interface {
	// IncrementSourcesAvailabilityCheckRequestsCounter increments the counter of the successful availability check
	// requests sent to downstream services.
	IncrementSourcesAvailabilityCheckRequestsCounter()

	// IncrementSourcesAvailabilityCheckFailedRequestsCounter increments the counter of the failed availability check
	// requests sent to downstream services. The "errorOrigin" argument aims to help identifying where the errors are
	// originating.
	IncrementSourcesAvailabilityCheckFailedRequestsCounter(origin ErrorOrigin)
}
