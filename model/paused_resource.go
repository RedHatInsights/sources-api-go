package model

import (
	"fmt"
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
)

type ResourceEditPausedRequest struct {
	AvailabilityStatus      *string `json:"availability_status"`
	AvailabilityStatusError *string `json:"availability_status_error"`
	LastAvailableAt         *string `json:"last_available_at"`
	LastCheckedAt           *string `json:"last_checked_at"`
}

func (app *Application) UpdateFromRequestPaused(req *ResourceEditPausedRequest) error {
	availabilityStatus := req.AvailabilityStatus
	availabilityStatusError := req.AvailabilityStatusError
	lastAvailableAt := req.LastAvailableAt
	lastCheckedAt := req.LastCheckedAt

	if availabilityStatus != nil {
		if _, ok := ValidAvailabilityStatuses[*req.AvailabilityStatus]; !ok {
			return fmt.Errorf(`invalid availability status. Must be one of "available", "in_progress", "partially_available" or "unavailable"`)
		}

		app.AvailabilityStatus = *availabilityStatus
	}

	if availabilityStatusError != nil {
		app.AvailabilityStatusError = *availabilityStatusError
	}

	if lastAvailableAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastAvailableAt)
		if err != nil {
			logging.Log.Warnf(`[application_id: %d] invalid "last available at" date received to update a paused application: %s`, app.ID, *lastAvailableAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		app.LastAvailableAt = &t
	}

	if lastCheckedAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastCheckedAt)
		if err != nil {
			logging.Log.Warnf(`[application_id: %d] invalid "last checked at" date received to update a paused application: %s`, app.ID, *lastCheckedAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		app.LastAvailableAt = &t
	}

	return nil
}

func (endpoint *Endpoint) UpdateFromRequestPaused(req *ResourceEditPausedRequest) error {
	availabilityStatus := req.AvailabilityStatus
	availabilityStatusError := req.AvailabilityStatusError
	lastAvailableAt := req.LastAvailableAt
	lastCheckedAt := req.LastCheckedAt

	if availabilityStatus != nil {
		endpoint.AvailabilityStatus = *availabilityStatus
	}

	if availabilityStatusError != nil {
		endpoint.AvailabilityStatusError = availabilityStatusError
	}

	if lastAvailableAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastAvailableAt)
		if err != nil {
			logging.Log.Warnf(`[endpoint_id: %d] invalid "last available at" date received to update a paused endpoint: %s`, endpoint.ID, *lastAvailableAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		endpoint.LastAvailableAt = &t
	}

	if lastCheckedAt != nil {
		t, err := time.Parse(util.RecordDateTimeFormat, *lastCheckedAt)
		if err != nil {
			logging.Log.Warnf(`[endpoint_id: %d] invalid "last checked at" date received to update a paused endpoint: %s`, endpoint.ID, *lastCheckedAt)

			return fmt.Errorf(`the provided date is in an invalid format. Expected format: "%s"`, util.RecordDateTimeFormat)
		}

		endpoint.LastAvailableAt = &t
	}

	return nil
}
