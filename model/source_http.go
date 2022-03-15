package model

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
	AvailabilityStatusResponse

	ID                  string  `json:"id"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	PausedAt            string  `json:"paused_at,omitempty"`
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`

	SourceTypeId string `json:"source_type_id"`
}

// SourceInternalResponse represents the structure we will return
// when a source is requested from the internal endpoint.
type SourceInternalResponse struct {
	Id                 *string `json:"id"`
	AvailabilityStatus *string `json:"availability_status"`
	ExternalTenant     *string `json:"tenant"`
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
