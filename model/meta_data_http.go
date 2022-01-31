package model

import (
	"gorm.io/datatypes"
)

type MetaDataResponse struct {
	ID        string  `json:"id"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`

	Name              string         `json:"name"`
	Payload           datatypes.JSON `json:"payload"`
	ApplicationTypeId string         `json:"application_type_id"`
}
