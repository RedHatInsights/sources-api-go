package model

import "time"

type User struct {
	Id     int64  `gorm:"primarykey" json:"id"`
	UserID string `json:"user_id"`

	TenantID int64
	Tenant   Tenant

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
