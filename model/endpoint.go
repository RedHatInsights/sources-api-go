package model

import (
	"gorm.io/gorm"
	"time"
)

type Endpoint struct {
	AvailabilityStatus
	Tenancy

	Id        int64          `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:paused_at" json:"paused_at"`

	Role                    *string `json:"role,omitempty"`
	Port                    *int    `json:"port,omitempty"`
	Default                 *bool   `json:"default,omitempty"`
	Scheme                  *string `json:"scheme,omitempty"`
	Host                    *string `json:"host,omitempty"`
	Path                    *string `json:"path,omitempty"`
	VerifySsl               *bool   `json:"verify_ssl,omitempty"`
	CertificateAuthority    *string `json:"certificate_authority,omitempty"`
	ReceptorNode            *string `json:"receptor_node,omitempty"`
	AvailabilityStatusError *string `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`

	Source *Source
}
