package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
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
		CreatedAt:   util.DateTimeToRFC3339(a.CreatedAt),
		UpdatedAt:   util.DateTimeToRFC3339(a.UpdatedAt),
		Name:        a.Name,
		ProductName: a.ProductName,
		Vendor:      a.Vendor,
		Schema:      a.Schema,
		IconUrl:     a.IconUrl,
	}
}
