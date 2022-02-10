package model

type SourceRhcConnection struct {
	SourceId        int64 `gorm:"primaryKey"`
	RhcConnectionId int64 `gorm:"primaryKey"`
}
