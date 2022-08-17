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
func ValidateSourceCreationRequest(dao dao.SourceDao, req *model.SourceCreateRequest) error {

	if req.Name == nil || *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if dao.NameExistsInCurrentTenant(*req.Name) {
		return fmt.Errorf("name already exists in tenant")
	}

	// Generate a new UUID and assign it to the source, as the field is not received from the
	generatedUuid := uuid.New()
	uuids := generatedUuid.String()
	req.Uid = &uuids

	if !util.SliceContainsString(validWorkflowStatuses, req.AppCreationWorkflow) {
		req.AppCreationWorkflow = model.ManualConfig
	}

	if !util.SliceContainsString(model.AvailabilityStatuses, req.AvailabilityStatus) {
		return fmt.Errorf("invalid status")
	}

	// Try to get the SourceTypeID. If an error occurs, the user gets a generic error message, as they are not
	// interested in the underlying ones
	value, err := util.InterfaceToInt64(req.SourceTypeIDRaw)
	if err != nil {
		return errors.New("the source type id is not valid")
	}

	if value < 1 {
		return fmt.Errorf("source type id must be greater than 0")
	}

	req.SourceTypeID = &value

	return nil
}

func ValidateEditSourceNameRequest(dao dao.SourceDao, req *model.SourceEditRequest) error {
	if req.Name == nil || *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if dao.NameExistsInCurrentTenant(*req.Name) {
		return fmt.Errorf("source name already exists in same tenant")
	}

	return nil
}
