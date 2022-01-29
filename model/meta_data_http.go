package model

import (
	"time"

	"gorm.io/datatypes"
)

type MetaDataResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name              string         `json:"name"`
	Payload           datatypes.JSON `json:"payload"`
	ApplicationTypeId string         `json:"application_type_id"`
}
