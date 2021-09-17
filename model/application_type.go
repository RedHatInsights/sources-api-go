package model

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
)

type ApplicationType struct {
	//fields for gorm
	Id        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	DisplayName                  string         `json:"display_name"`
	DependentApplications        datatypes.JSON `json:"dependent_applications"`
	SupportedSourceTypes         datatypes.JSON `json:"supported_source_types"`
	SupportedAuthenticationTypes datatypes.JSON `json:"supported_authentication_types"`

	Applications []Application
	Sources      []*Source `gorm:"many2many:applications;"`
	MetaData     []MetaData
}

func (a *ApplicationType) ToResponse() *ApplicationTypeResponse {
	id := strconv.Itoa(int(a.Id))

	// returning the address of the new struct.
	return &ApplicationTypeResponse{
		Id:                           id,
		CreatedAt:                    a.CreatedAt,
		UpdatedAt:                    a.UpdatedAt,
		DisplayName:                  a.DisplayName,
		DependentApplications:        a.DependentApplications,
		SupportedSourceTypes:         a.SupportedSourceTypes,
		SupportedAuthenticationTypes: a.SupportedAuthenticationTypes,
	}
}
