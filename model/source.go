package model

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Source struct that includes all of the fields on the table
// used internally for business logic
type Source struct {
	AvailabilityStatus
	Pause
	Tenancy

	//fields for gorm
	Id        int64          `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:paused_at" json:"paused_at"`

	// standard source fields
	Name                string  `json:"name"`
	Uid                 *string `json:"uid,omitempty"`
	Version             *string `json:"version,omitempty"`
	Imported            *string `json:"imported,omitempty"`
	SourceRef           *string `json:"source_ref,omitempty"`
	AppCreationWorkflow string  `gorm:"default:manual_configuration" json:"app_creation_workflow"`

	SourceTypeId *int64 `json:"source_type_id"`

	Applications []Application
	Endpoints    []Endpoint
}

func (src *Source) ToResponse() *SourceResponse {
	id := strconv.FormatInt(src.Id, 10)
	stid := strconv.FormatInt(*src.SourceTypeId, 10)

	return &SourceResponse{
		AvailabilityStatus:  src.AvailabilityStatus,
		Pause:               src.Pause,
		TimeStamps:          TimeStamps{CreatedAt: &src.CreatedAt, UpdatedAt: &src.UpdatedAt},
		Id:                  &id,
		Name:                &src.Name,
		Uid:                 src.Uid,
		Version:             src.Version,
		Imported:            src.Imported,
		SourceRef:           src.SourceRef,
		AppCreationWorkflow: &src.AppCreationWorkflow,
		SourceTypeId:        &stid,
	}
}
