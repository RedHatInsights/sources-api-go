package service

import (
	"errors"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
)

// Declare the valid workflow statuses at package level to avoid instantiating the slice every time a request
// needs to be validated.
var validWorkflowStatuses = []string{model.AccountAuth, model.ManualConfig}

// ValidateSourceCreationRequest validates that the required fields of the SourceCreateRequest request hold proper
// values. In the specific case of the UUID, if an empty or nil one is provided, a new random UUID is generated and
// appended to the request.
func ValidateSourceCreationRequest(sourceDao dao.SourceDao, req *model.SourceCreateRequest) error {
	if req.Name == nil || *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if sourceDao.NameExistsInCurrentTenant(*req.Name) {
		return fmt.Errorf("name already exists in tenant")
	}

	// Generate a new UUID and assign it to the source, as the field is not received from the
	generatedUuid := uuid.New()
	uuids := generatedUuid.String()
	req.Uid = &uuids

	if !util.SliceContainsString(validWorkflowStatuses, req.AppCreationWorkflow) {
		req.AppCreationWorkflow = model.ManualConfig
	}

	// The team decided that the availability statuses will default to "in_progress" whenever they come empty, since
	// setting them as "unavailable" by default may lead to some confusion to the calling clients.
	if req.AvailabilityStatus == "" {
		req.AvailabilityStatus = model.InProgress
	} else {
		if _, ok := model.ValidAvailabilityStatuses[req.AvailabilityStatus]; !ok {
			return fmt.Errorf("invalid status")
		}
	}

	// Try to get the SourceTypeID. If an error occurs, the user gets a generic error message, as they are not
	// interested in the underlying ones
	if req.SourceTypeIDRaw == nil {
		return errors.New("source type id cannot be empty")
	}

	value, err := util.InterfaceToInt64(req.SourceTypeIDRaw)
	if err != nil {
		return errors.New("the source type id is not valid")
	}

	if value < 1 {
		return fmt.Errorf("source type id must be greater than 0")
	}

	// Check that SourceTypeId exists
	sourceTypeName := dao.Static.GetSourceTypeName(value)
	if sourceTypeName == "" {
		return fmt.Errorf("source type id not found")
	}

	req.SourceTypeID = &value

	return nil
}

func ValidateSourceEditRequest(dao dao.SourceDao, editRequest *model.SourceEditRequest) error {
	if editRequest.Name != nil {
		if *editRequest.Name == "" {
			return errors.New("name cannot be empty")
		}

		if dao.NameExistsInCurrentTenant(*editRequest.Name) {
			return fmt.Errorf("source name already exists in same tenant")
		}
	}

	// On source edits, we don't set any default values and don't allow empty availability status values.
	if editRequest.AvailabilityStatus != nil {
		if _, ok := model.ValidAvailabilityStatuses[*editRequest.AvailabilityStatus]; !ok {
			return errors.New(`availability status invalid. Must be one of "available", "in_progress", "partially_available" or "unavailable"`)
		}
	}

	return nil
}
