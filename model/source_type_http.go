package model

import (
	"time"

	"gorm.io/datatypes"
)

type SourceTypeResponse struct {
	//fields for gorm
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name        string         `json:"name"`
	ProductName string         `json:"product_name"`
	Vendor      string         `json:"vendor"`
	Schema      datatypes.JSON `json:"schema"`
	IconUrl     string         `json:"icon_url"`
}
