package model

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
)

type SourceType struct {
	//fields for gorm
	Id        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name        string         `json:"name"`
	ProductName string         `json:"product_name"`
	Vendor      string         `json:"vendor"`
	Schema      datatypes.JSON `json:"schema"`
	IconUrl     string         `json:"icon_url"`

	Sources []Source
}

func (a *SourceType) ToResponse() *SourceTypeResponse {
	id := strconv.Itoa(int(a.Id))

	// returning the address of the new struct.
	return &SourceTypeResponse{
		Id:          id,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Name:        a.Name,
		ProductName: a.ProductName,
		Vendor:      a.Vendor,
		Schema:      a.Schema,
		IconUrl:     a.IconUrl,
	}
}

type SourceTypeSeed struct {
	ProductName string      `yaml:"product_name"`
	Schema      interface{} `yaml:"schema"`
	Vendor      string      `yaml:"vendor"`
	IconURL     string      `yaml:"icon_url"`
}
