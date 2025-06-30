package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type prometheusMetricsService struct {
	availabilityCheckRequestsCounter *prometheus.CounterVec
}

// NewPrometheusMetricsService creates and registers the metrics in order to satisfy the MetricsService interface.
// Returns an error when the metrics cannot be registered.
func NewPrometheusMetricsService() (MetricsService, error) {
	availabilityCheckRequestsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sources_availability_check_requests_total",
		Help: "Counts the number of availability check requests sent to downstream services that depend on Sources",
	}, []string{
		// What was the status of the operation?
		"status",
		// Where was the error originated?
		"error_origin",
	})

	err := prometheus.Register(availabilityCheckRequestsCounter)
	if err != nil {
		return nil, fmt.Errorf(`unable to register the "availability check requests" counter: %w`, err)
	}

	return &prometheusMetricsService{
		availabilityCheckRequestsCounter: availabilityCheckRequestsCounter,
	}, nil
}

func (s *prometheusMetricsService) IncrementSourcesAvailabilityCheckRequestsCounter() {
	s.availabilityCheckRequestsCounter.With(
		prometheus.Labels{
			"status": outcomeName[resultSuccess],
		},
	).Inc()
}

func (s *prometheusMetricsService) IncrementSourcesAvailabilityCheckFailedRequestsCounter(origin ErrorOrigin) {
	s.availabilityCheckRequestsCounter.With(
		prometheus.Labels{
			"status":       outcomeName[resultFailure],
			"error_origin": originName[origin],
		},
	).Inc()
}
