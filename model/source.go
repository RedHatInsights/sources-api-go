package model

type Source struct {
	AvailabilityStatus
	Pause
	Tenancy
	TimeStamps

	Id                  int64   `json:"id"`
	Name                string  `json:"name"`
	Uid                 string  `json:"uid,omitempty"`
	Version             string  `json:"version,omitempty"`
	Imported            string  `json:"imported,omitempty"`
	SourceRef           string  `json:"source_ref,omitempty"`
	AppCreationWorkflow *string `json:"app_creation_workflow"`

	SourceTypeId int64 `json:"source_type_id"`
}
