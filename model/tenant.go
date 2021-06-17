package model

type Tenant struct {
	TimeStamps

	Id             int64
	ExternalTenant string
}
