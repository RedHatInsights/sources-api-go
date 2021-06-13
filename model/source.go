package model

// Source struct that includes all of the fields on the table
// used internallyfor business logic
type Source struct {
	AvailabilityStatus
	Pause
	Tenancy
	TimeStamps

	Id                  int64   `json:"id"`
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`

	SourceTypeId int64 `json:"source_type_id"`
}

// SourceCreateRequest is a struct representing a request coming
// from the outside to create a struct, this is the way we will be marking
// fields as write-once. They are accepted on create but not edit.
type SourceCreateRequest struct {
	Name                    *string `json:"name"`
	Uid                     *string `json:"uid,omitempty"`
	Version                 *string `json:"version,omitempty"`
	Imported                *string `json:"imported,omitempty"`
	SourceRef               *string `json:"source_ref,omitempty"`
	AppCreationWorkflow     *string `json:"app_creation_workflow"`
	AvailabilityStatus      *string `json:"availability_status"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`

	SourceTypeId int64 `json:"source_type_id"`
}

type SourceEditRequest struct {
	Name                    *string `json:"name"`
	Uid                     *string `json:"uid,omitempty"`
	Version                 *string `json:"version,omitempty"`
	Imported                *string `json:"imported,omitempty"`
	SourceRef               *string `json:"source_ref,omitempty"`
	AvailabilityStatus      *string `json:"availability_status"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`
}

// SourceResponse represents what we will always return to the users
// of the API after a request.
type SourceResponse struct {
	AvailabilityStatus
	Pause
	TimeStamps

	Id                  int64   `json:"id"`
	Name                *string `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`

	SourceTypeId int64 `json:"source_type_id"`
}

func (src *Source) ToResponse() *SourceResponse {
	return &SourceResponse{
		AvailabilityStatus:  src.AvailabilityStatus,
		Pause:               src.Pause,
		TimeStamps:          src.TimeStamps,
		Id:                  src.Id,
		Name:                src.Name,
		Uid:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: src.AppCreationWorkflow,
		SourceTypeId:        src.SourceTypeId,
	}
}
