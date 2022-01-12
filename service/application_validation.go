package service

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// by default we'll be using an empty instanct of the apptype dao - replacing it
// in tests.
var appTypeDao dao.ApplicationTypeDao = &dao.ApplicationTypeDaoImpl{}

/*
	Go through and validate the application create request.

	Really not much here other than validating the application type is
	compatible with the specified source type.
*/
func ValidateApplicationCreateRequest(appReq *m.ApplicationCreateRequest) error {
	// need both source id + application type id
	if appReq.SourceIDRaw == nil || appReq.ApplicationTypeIDRaw == nil {
		return fmt.Errorf("missing required parameters of source_id or application_type_id")
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
	compatible, err := appTypeDao.ApplicationTypeCompatibleWithSource(appReq.ApplicationTypeID, appReq.SourceID)
	if err != nil {
		return fmt.Errorf("failed to check compatibility between application and source type")
	}

	if !compatible {
		return fmt.Errorf("source type is not compatible with this application type")
	}

	return nil
}
