package model

import "strconv"

// Source struct that includes all of the fields on the table
// used internally for business logic
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

	SourceTypeId *int64 `json:"source_type_id"`

	Applications []Application `bun:"rel:has-many"`
	Endpoints    []Endpoint    `bun:"rel:has-many"`
}

func (src *Source) ToResponse() *SourceResponse {
	id := strconv.FormatInt(src.Id, 10)
	stid := strconv.FormatInt(*src.SourceTypeId, 10)

	return &SourceResponse{
		AvailabilityStatus:  src.AvailabilityStatus,
		Pause:               src.Pause,
		TimeStamps:          src.TimeStamps,
		Id:                  &id,
		Name:                src.Name,
		Uid:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: src.AppCreationWorkflow,
		SourceTypeId:        &stid,
	}
}
