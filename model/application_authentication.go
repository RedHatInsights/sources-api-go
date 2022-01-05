package model

import (
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
)

type ApplicationAuthentication struct {
	Pause

	ID        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	TenantID int64
	Tenant   Tenant

	ApplicationID int64 `json:"application_id"`
	Application   Application
	// TODO: fix correctly PR#40
	AuthenticationID int64 `json:"authentication_id"`
	//Authentication   Authentication
}

func (aa *ApplicationAuthentication) ToEvent() *ApplicationAuthenticationEvent {
	aaEvent := &ApplicationAuthenticationEvent{
		ID:               aa.ID,
		PauseEvent:       PauseEvent{PausedAt: util.DateTimeToRecordFormat(aa.PausedAt)},
		CreatedAt:        util.DateTimeToRecordFormat(aa.CreatedAt),
		UpdatedAt:        util.DateTimeToRecordFormat(aa.UpdatedAt),
		ApplicationID:    aa.ApplicationID,
		AuthenticationID: aa.AuthenticationID,
		Tenant:           &aa.Tenant.ExternalTenant,
	}

	return aaEvent
}

func (aa *ApplicationAuthentication) ToResponse() *ApplicationAuthenticationResponse {
	id := strconv.FormatInt(aa.ID, 10)
	appId := strconv.FormatInt(aa.ApplicationID, 10)
	authId := strconv.FormatInt(aa.AuthenticationID, 10)

	return &ApplicationAuthenticationResponse{
		ID:               id,
		CreatedAt:        aa.CreatedAt,
		UpdatedAt:        aa.UpdatedAt,
		ApplicationID:    appId,
		AuthenticationID: authId,
	}
}
