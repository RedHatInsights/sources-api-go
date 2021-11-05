package model

import (
	"time"

	"gorm.io/datatypes"
)

type ApplicationTypeResponse struct {
	//fields for gorm
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name                         string         `json:"name"`
	DisplayName                  string         `json:"display_name"`
	DependentApplications        datatypes.JSON `json:"dependent_applications"`
	SupportedSourceTypes         datatypes.JSON `json:"supported_source_types"`
	SupportedAuthenticationTypes datatypes.JSON `json:"supported_authentication_types"`
}
