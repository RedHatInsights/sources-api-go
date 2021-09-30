package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
)

// Declare the valid workflow statuses at package level to avoid instantiating the slice every time a request
// needs to be validated.
var validWorkflowStatuses = []string{AccountAuth, ManualConfig}

// SourceCreateRequest is a struct representing a request coming
// from the outside to create a struct, this is the way we will be marking
// fields as write-once. They are accepted on create but not edit.
type SourceCreateRequest struct {
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow string  `json:"app_creation_workflow"`
	AvailabilityStatus  string  `json:"availability_status"`

	SourceTypeID    *int64      `json:"-"`
	SourceTypeIDRaw interface{} `json:"source_type_id"`
}

// Validate validates that the required fields of the SourceCreateRequest request hold proper values. In the specific
// case of the UUID, if an empty or nil one is provided, a new random UUID is generated and appended to the request.
func (req *SourceCreateRequest) Validate() error {
	if req.Name == nil || *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Generate a new UUID and assign it to the source, as the field is not received from the
	generatedUuid := uuid.New()
	uuids := generatedUuid.String()
	req.Uid = &uuids

	if !util.SliceContainsString(validWorkflowStatuses, req.AppCreationWorkflow) {
		req.AppCreationWorkflow = ManualConfig
	}

	if !util.SliceContainsString(availabilityStatuses, req.AvailabilityStatus) {
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

// SourceEditRequest manages what we can/cannot update on the source
// object. Any extra params just will not serialize.
type SourceEditRequest struct {
	Name               *string `json:"name"`
	Version            *string `json:"version,omitempty"`
	Imported           *string `json:"imported,omitempty"`
	SourceRef          *string `json:"source_ref,omitempty"`
	AvailabilityStatus *string `json:"availability_status"`
}

// SourceResponse represents what we will always return to the users
// of the API after a request.
type SourceResponse struct {
	AvailabilityStatus
	Pause

	ID                  *string   `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Name                *string   `json:"name"`
	Uid                 *string   `json:"uid,omitempty"`
	Version             *string   `json:"version,omitempty"`
	Imported            *string   `json:"imported,omitempty"`
	SourceRef           *string   `json:"source_ref,omitempty"`
	AppCreationWorkflow *string   `json:"app_creation_workflow"`

	SourceTypeId *string `json:"source_type_id"`
}

func (src *Source) UpdateFromRequest(update *SourceEditRequest) {
	if update.Name != nil {
		src.Name = *update.Name
	}
	if update.Version != nil {
		src.Version = update.Version
	}
	if update.Imported != nil {
		src.Imported = update.Imported
	}
	if update.SourceRef != nil {
		src.SourceRef = update.SourceRef
	}
	if update.AvailabilityStatus != nil {
		src.AvailabilityStatus = AvailabilityStatus{
			AvailabilityStatus: *update.AvailabilityStatus,
		}
	}
}
