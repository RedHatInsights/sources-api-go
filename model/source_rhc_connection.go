package model

type SourceRhcConnection struct {
	SourceId int64 `gorm:"primaryKey"`

	RhcConnectionId int64 `gorm:"primaryKey"`
	RhcConnection   RhcConnection

	TenantId int64
	Tenant   Tenant
}
