package service

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// by default we'll be using an empty instance of the appType dao - replacing it
// in tests.
var AppTypeDao = dao.GetApplicationTypeDao(nil)

/*
	Go through and validate the application create request.

	Really not much here other than validating the application type is
	compatible with the specified source type.
*/
func ValidateApplicationCreateRequest(appReq *m.ApplicationCreateRequest) error {
	// need both source id + application type id
	if appReq.SourceIDRaw == nil {
		return fmt.Errorf("missing required parameter source_id")
	}

	if appReq.ApplicationTypeIDRaw == nil {
		return fmt.Errorf("missing required parameter application_type_id")
	}

	// parse both the ids
	appTypeID, err := util.InterfaceToInt64(appReq.ApplicationTypeIDRaw)
	if err != nil {
		return fmt.Errorf("invalid application type id %v", appReq.ApplicationTypeIDRaw)
	}
	appReq.ApplicationTypeID = appTypeID

	source, err := util.InterfaceToInt64(appReq.SourceIDRaw)
	if err != nil {
		return fmt.Errorf("invalid source id %v", appReq.SourceIDRaw)
	}
	appReq.SourceID = source

	// check that the application type supports the source type we're attaching
	// it to.
	err = AppTypeDao.ApplicationTypeCompatibleWithSource(appReq.ApplicationTypeID, appReq.SourceID)
	if err != nil {
		return fmt.Errorf("source type is not compatible with this application type")
	}

	return nil
}

// ValidateApplicationEditRequest validates that the edit request received for an application is valid.
func ValidateApplicationEditRequest(editReq *m.ApplicationEditRequest) error {
	// The availability status could be "nil" if the JSON is missing the key. But that's okay, since the user might be
	// purposely omitting it because the availability status didn't change.
	if editReq.AvailabilityStatus != nil {
		if _, ok := m.ValidAvailabilityStatuses[*editReq.AvailabilityStatus]; !ok {
			return errors.New(`availability status invalid. Must be one of "available", "in_progress", "partially_available" or "unavailable"`)
		}
	}

	return nil
}
