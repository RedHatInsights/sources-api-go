package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/uuid"
)

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

	// If no valid UUID is provided in the request, we generate one
	if req.Uid == nil || *req.Uid == "" {
		id := uuid.New()
		stringId := id.String()
		req.Uid = &stringId
	}

	var validWorkflowStatuses = []string{AccountAuth, ManualConfig}
	if !util.IsStringPresentInSlice(req.AppCreationWorkflow, validWorkflowStatuses) {
		return fmt.Errorf("invalid workflow specified")
	}

	var validAvailabilityStatuses = []string{"", Available, InProgress, PartiallyAvailable, Unavailable}
	if !util.IsStringPresentInSlice(req.AvailabilityStatus, validAvailabilityStatuses) {
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
