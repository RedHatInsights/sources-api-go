package model

import (
	"fmt"
	"strconv"
	"time"

	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type Authentication struct {
	DbID      int64     `gorm:"primaryKey; column:id" json:"-"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at" gorm:"-"`

	Name     *string `json:"name,omitempty"`
	AuthType string  `gorm:"column:authtype" json:"authtype"`
	Username *string `json:"username"`
	Version  string  `json:"version" gorm:"-"`

	// DO NOT set these fields directly, instead see secret_store_util.go for
	// interacting with store-specific fields.
	Password    *string                `json:"password_hash" gorm:"column:password_hash"`
	MiqPassword *string                `json:"password" gorm:"column:password"`
	Extra       map[string]interface{} `gorm:"-" json:"extra,omitempty"`
	ExtraDb     datatypes.JSON         `gorm:"column:extra"`

	AvailabilityStatus      *string    `gorm:"default:in_progress;not null" json:"availability_status,omitempty"`
	LastCheckedAt           *time.Time `json:"last_checked_at,omitempty"`
	LastAvailableAt         *time.Time `json:"last_available_at,omitempty"`
	AvailabilityStatusError *string    `json:"availability_status_error,omitempty"`

	SourceID int64 `json:"source_id"`
	Source   Source

	TenantID int64 `json:"tenant_id"`
	Tenant   Tenant
	UserID   *int64 `json:"user_id"`
	User     User

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

	return &AuthenticationResponse{
		ID:                      auth.GetID(),
		Name:                    util.ValueOrBlank(auth.Name),
		AuthType:                auth.AuthType,
		Username:                util.ValueOrBlank(auth.Username),
		Extra:                   auth.GetExtra(),
		AvailabilityStatus:      util.ValueOrBlank(auth.AvailabilityStatus),
		AvailabilityStatusError: util.ValueOrBlank(auth.AvailabilityStatusError),
		ResourceType:            auth.ResourceType,
		ResourceID:              resourceID,
	}
}

func (auth *Authentication) ToSecretResponse() *SecretResponse {
	return &SecretResponse{
		ID:       auth.GetID(),
		Name:     util.ValueOrBlank(auth.Name),
		AuthType: auth.AuthType,
		Username: util.ValueOrBlank(auth.Username),
		Extra:    auth.GetExtra(),
	}
}

func (auth *Authentication) ToInternalSecretResponse() *SecretInternalResponse {
	pass, err := auth.GetPassword()
	if err != nil {
		panic("failed to get password from secret: " + err.Error())
	}

	return &SecretInternalResponse{
		ID:       auth.GetID(),
		Name:     util.ValueOrBlank(auth.Name),
		AuthType: auth.AuthType,
		Username: util.ValueOrBlank(auth.Username),
		Extra:    auth.GetExtra(),
		Password: *pass,
	}
}

func (auth *Authentication) ToInternalResponse() *AuthenticationInternalResponse {
	resourceID := strconv.FormatInt(auth.ResourceID, 10)

	pass, err := auth.GetPassword()
	if err != nil {
		l.Log.Warnf("Failed to get password from authentication: %v", err)
	}
	// pass can be nil - but we can't have that for the response below.
	if pass == nil {
		pass = util.StringRef("")
	}

	return &AuthenticationInternalResponse{
		ID:                      auth.GetID(),
		CreatedAt:               auth.CreatedAt,
		Name:                    util.ValueOrBlank(auth.Name),
		Version:                 auth.Version,
		AuthType:                auth.AuthType,
		Username:                util.ValueOrBlank(auth.Username),
		Password:                *pass,
		Extra:                   auth.GetExtra(),
		AvailabilityStatus:      util.ValueOrBlank(auth.AvailabilityStatus),
		AvailabilityStatusError: util.ValueOrBlank(auth.AvailabilityStatusError),
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
		"availability_status":       auth.AvailabilityStatus,
		"availability_status_error": auth.AvailabilityStatusError,
		"last_checked_at":           auth.LastCheckedAt,
		"last_available_at":         auth.LastAvailableAt,
		"resource_type":             auth.ResourceType,
		"resource_id":               strconv.FormatInt(auth.ResourceID, 10),
		"source_id":                 strconv.FormatInt(auth.SourceID, 10),
		"created_at":                auth.CreatedAt,
	}

	// Vault requires the hash to be wrapped in a "data" object in order to be accepted.
	return map[string]interface{}{"data": data}, nil
}

func (auth *Authentication) ToEvent() interface{} {
	return &AuthenticationEvent{
		ID:                      auth.GetID(),
		CreatedAt:               auth.CreatedAt,
		Name:                    util.ValueOrBlank(auth.Name),
		AuthType:                auth.AuthType,
		Version:                 auth.Version,
		Username:                util.ValueOrBlank(auth.Username),
		Extra:                   auth.GetExtra(),
		AvailabilityStatus:      util.StringValueOrNil(auth.AvailabilityStatus),
		LastAvailableAt:         util.DateTimePointerToRecordFormat(auth.LastAvailableAt),
		LastCheckedAt:           util.DateTimePointerToRecordFormat(auth.LastCheckedAt),
		AvailabilityStatusError: auth.AvailabilityStatusError,
		ResourceType:            auth.ResourceType,
		ResourceID:              auth.ResourceID,
		Tenant:                  &auth.Tenant.ExternalTenant,
		SourceID:                auth.SourceID,
	}
}

func (auth *Authentication) UpdateBy(attributes map[string]interface{}) error {
	if attributes["last_checked_at"] != nil {
		lastCheckedAt, _ := time.Parse(time.RFC3339Nano, attributes["last_checked_at"].(string))
		auth.LastCheckedAt = &lastCheckedAt
	}

	if attributes["last_available_at"] != nil {
		lastAvailableAt, _ := time.Parse(time.RFC3339Nano, attributes["last_available_at"].(string))
		auth.LastAvailableAt = &lastAvailableAt
	}

	if attributes["availability_status_error"] != nil {
		availabilityStatusError, _ := attributes["availability_status_error"].(string)
		auth.AvailabilityStatusError = &availabilityStatusError
	}

	if attributes["availability_status"] != nil {
		availabilityStatus, _ := attributes["availability_status"].(string)
		auth.AvailabilityStatus = &availabilityStatus
	}

	return nil
}

func (auth *Authentication) Path() string {
	return fmt.Sprintf("secret/data/%d/%s_%v_%s", auth.TenantID, auth.ResourceType, auth.ResourceID, auth.ID)
}

func (auth *Authentication) ToEmail(previousStatus string) *EmailNotificationInfo {
	availabilityStatus := ""
	if auth.AvailabilityStatus != nil {
		availabilityStatus = *auth.AvailabilityStatus
	}

	return &EmailNotificationInfo{
		ResourceDisplayName:        "Authentication",
		CurrentAvailabilityStatus:  util.FormatAvailabilityStatus(availabilityStatus),
		PreviousAvailabilityStatus: util.FormatAvailabilityStatus(previousStatus),
		SourceName:                 auth.Source.Name,
		SourceID:                   strconv.FormatInt(auth.SourceID, 10),
		TenantID:                   strconv.FormatInt(auth.TenantID, 10),
	}
}
