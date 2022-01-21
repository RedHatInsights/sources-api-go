package model

import (
	"encoding/json"
	"strconv"
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type SourceType struct {
	//fields for gorm
	Id        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name         string            `json:"name"`
	ProductName  string            `json:"product_name"`
	Vendor       string            `json:"vendor"`
	IconUrl      string            `json:"icon_url"`
	Schema       datatypes.JSON    `json:"schema"`
	SchemaParsed *sourceTypeScheme `gorm:"-"`

	Sources []Source
}

func (st *SourceType) ToResponse() *SourceTypeResponse {
	id := strconv.Itoa(int(st.Id))

	// returning the address of the new struct.
	return &SourceTypeResponse{
		Id:          id,
		CreatedAt:   util.DateTimeToRFC3339(st.CreatedAt),
		UpdatedAt:   util.DateTimeToRFC3339(st.UpdatedAt),
		Name:        st.Name,
		ProductName: st.ProductName,
		Vendor:      st.Vendor,
		Schema:      st.Schema,
		IconUrl:     st.IconUrl,
	}
}

func (st *SourceType) SuperkeyAuthType() string {
	if st.SchemaParsed == nil {
		schema := sourceTypeScheme{}

		err := json.Unmarshal(st.Schema, &schema)
		if err != nil {
			l.Log.Warnf("Failed to parse SourceType schema for id [%v]", st.Id)
		}

		st.SchemaParsed = &schema
	}

	// default empty values will be present and this won't go if the parsing
	// fails.
	for _, authSchema := range st.SchemaParsed.Authentications {
		if authSchema.IsSuperkey {
			return authSchema.Type
		}
	}

	return ""
}

type sourceTypeScheme struct {
	Endpoint        sourceTypeEndpoint         `json:"endpoint"`
	Authentications []sourceTypeAuthentication `json:"authentication"`
}
type sourceTypeEndpointField struct {
	Name              string `json:"name"`
	Component         string `json:"component"`
	HideField         bool   `json:"hideField"`
	InitialValue      string `json:"initialValue"`
	InitializeOnMount bool   `json:"initializeOnMount"`
}
type sourceTypeEndpoint struct {
	Fields []sourceTypeEndpointField `json:"fields"`
	Hidden bool                      `json:"hidden"`
}
type sourceTypeAuthenticationField struct {
	Name              string `json:"name"`
	Component         string `json:"component"`
	HideField         bool   `json:"hideField,omitempty"`
	InitialValue      string `json:"initialValue,omitempty"`
	InitializeOnMount bool   `json:"initializeOnMount,omitempty"`
	Label             string `json:"label,omitempty"`
	Type              string `json:"type,omitempty"`
}
type sourceTypeAuthentication struct {
	Name       string                          `json:"name"`
	Type       string                          `json:"type"`
	Fields     []sourceTypeAuthenticationField `json:"fields"`
	IsSuperkey bool                            `json:"is_superkey,omitempty"`
}
