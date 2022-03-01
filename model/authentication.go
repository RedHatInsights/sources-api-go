package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type Authentication struct {
	AvailabilityStatus

	DbID      int64     `gorm:"primaryKey; column:id" json:"-"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name                    string                 `json:"name,omitempty"`
	AuthType                string                 `gorm:"column:authtype" json:"authtype"`
	Username                string                 `json:"username"`
	Password                string                 `json:"password"`
	Extra                   map[string]interface{} `gorm:"-" json:"extra,omitempty"`
	ExtraDb                 datatypes.JSON         `gorm:"column:extra"`
	Version                 string                 `json:"version"`
	AvailabilityStatusError string                 `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`
	Source   Source

	TenantID int64 `json:"tenant_id"`
	Tenant   Tenant

	ResourceType string `json:"resource_type"`
	ResourceID   int64  `json:"resource_id"`

	ApplicationAuthentications []ApplicationAuthentication
}

func (auth *Authentication) BulkMessage() map[string]interface{} {
	bulkMessage := map[string]interface{}{}
	bulkMessage["applications"] = auth.Source.Applications
	bulkMessage["authentications"] = nil
	bulkMessage["application_authentications"] = nil
	bulkMessage["endpoints"] = auth.Source.Endpoints
	bulkMessage["source"] = auth.Source

	return bulkMessage
}

func (auth *Authentication) ToResponse() *AuthenticationResponse {
	resourceID := strconv.FormatInt(auth.ResourceID, 10)

	id, extra := mapIdExtraFields(auth)
	return &AuthenticationResponse{
		ID:                      id,
		CreatedAt:               util.DateTimeToRFC3339(auth.CreatedAt),
		Name:                    auth.Name,
		Version:                 auth.Version,
		AuthType:                auth.AuthType,
		Username:                auth.Username,
		Extra:                   extra,
		AvailabilityStatus:      auth.AvailabilityStatus.AvailabilityStatus,
		AvailabilityStatusError: auth.AvailabilityStatusError,
		ResourceType:            auth.ResourceType,
		ResourceID:              resourceID,
	}
}

func (auth *Authentication) ToInternalResponse() *AuthenticationInternalResponse {
	resourceID := strconv.FormatInt(auth.ResourceID, 10)

	id, extra := mapIdExtraFields(auth)
	return &AuthenticationInternalResponse{
		ID:                      id,
		CreatedAt:               auth.CreatedAt,
		Name:                    auth.Name,
		Version:                 auth.Version,
		AuthType:                auth.AuthType,
		Username:                auth.Username,
		Password:                auth.Password,
		Extra:                   extra,
		AvailabilityStatus:      auth.AvailabilityStatus.AvailabilityStatus,
		AvailabilityStatusError: auth.AvailabilityStatusError,
		ResourceType:            auth.ResourceType,
		ResourceID:              resourceID,
	}
}

/*
	This method translates an Authentication struct to a hash that will be
	accepted by vault, this format will also be deserialized properly by
	dao.authFromVault, so if we are to add more fields they will need to be
	added here as well.
*/
func (auth *Authentication) ToVaultMap() (map[string]interface{}, error) {
	data := map[string]interface{}{
		"name":                      auth.Name,
		"authtype":                  auth.AuthType,
		"username":                  auth.Username,
		"password":                  auth.Password,
		"extra":                     auth.Extra,
		"availability_status":       auth.AvailabilityStatus.AvailabilityStatus,
		"availability_status_error": auth.AvailabilityStatusError,
		"last_checked_at":           auth.AvailabilityStatus.LastCheckedAt,
		"last_available_at":         auth.AvailabilityStatus.LastAvailableAt,
		"resource_type":             auth.ResourceType,
		"resource_id":               strconv.FormatInt(auth.ResourceID, 10),
		"source_id":                 strconv.FormatInt(auth.SourceID, 10),
	}

	// Vault requires the hash to be wrapped in a "data" object in order to be accepted.
	return map[string]interface{}{"data": data}, nil
}

func (auth *Authentication) ToEvent() interface{} {
	asEvent := AvailabilityStatusEvent{AvailabilityStatus: util.StringValueOrNil(auth.AvailabilityStatus.AvailabilityStatus),
		LastAvailableAt: util.DateTimeToRecordFormat(auth.LastAvailableAt),
		LastCheckedAt:   util.DateTimeToRecordFormat(auth.LastCheckedAt)}

	id, extra := mapIdExtraFields(auth)
	return &AuthenticationEvent{
		AvailabilityStatusEvent: asEvent,
		ID:                      id,
		CreatedAt:               auth.CreatedAt,
		Name:                    auth.Name,
		AuthType:                auth.AuthType,
		Version:                 auth.Version,
		Username:                auth.Username,
		Extra:                   extra,
		AvailabilityStatusError: &auth.AvailabilityStatusError,
		ResourceType:            auth.ResourceType,
		ResourceID:              auth.ResourceID,
		Tenant:                  &auth.Tenant.ExternalTenant,
		SourceID:                auth.SourceID,
	}
}

func (auth *Authentication) UpdateBy(attributes map[string]interface{}) error {
	if attributes["last_checked_at"] != nil {
		auth.AvailabilityStatus.LastCheckedAt, _ = time.Parse(time.RFC3339Nano, attributes["last_checked_at"].(string))
	}

	if attributes["last_available_at"] != nil {
		auth.AvailabilityStatus.LastAvailableAt, _ = time.Parse(time.RFC3339Nano, attributes["last_available_at"].(string))
	}

	if attributes["availability_status_error"] != nil {
		auth.AvailabilityStatusError, _ = attributes["availability_status_error"].(string)
	}

	if attributes["availability_status"] != nil {
		auth.AvailabilityStatus.AvailabilityStatus, _ = attributes["availability_status"].(string)
	}

	return nil
}

func (auth *Authentication) Path() string {
	return fmt.Sprintf("secret/data/%d/%s_%v_%s", auth.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)
}

// mapIdExtraFields returns the ID and the Extra fields ready to be assigned to the response model, depending on
func mapIdExtraFields(auth *Authentication) (string, map[string]interface{}) {
	var id string
	var extra map[string]interface{}
	if config.Get().VaultOn {
		id = auth.ID
		extra = auth.Extra
	} else {
		id = strconv.FormatInt(auth.DbID, 10)

		err := json.Unmarshal(auth.ExtraDb, &extra)
		if err != nil {
			logger.Log.Errorf(`could not unmarshal "extra" field from authentication with ID "%s"`, id)
		}
	}

	return id, extra
}
