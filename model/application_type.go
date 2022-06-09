package model

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/datatypes"
)

type ApplicationType struct {
	//fields for gorm
	Id        int64     `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name                         string         `json:"name"`
	DisplayName                  string         `json:"display_name"`
	DependentApplications        datatypes.JSON `json:"dependent_applications"`
	SupportedSourceTypes         datatypes.JSON `json:"supported_source_types"`
	SupportedAuthenticationTypes datatypes.JSON `json:"supported_authentication_types"`
	ResourceOwnership            *string        `json:"resource_ownership"`

	Applications []Application
	Sources      []*Source `gorm:"many2many:applications;"`
	MetaData     []MetaData
}

func (a *ApplicationType) ToResponse() *ApplicationTypeResponse {
	id := strconv.Itoa(int(a.Id))

	// returning the address of the new struct.
	return &ApplicationTypeResponse{
		Id:                           id,
		CreatedAt:                    util.DateTimeToRFC3339(a.CreatedAt),
		UpdatedAt:                    util.DateTimeToRFC3339(a.UpdatedAt),
		Name:                         a.Name,
		DisplayName:                  a.DisplayName,
		DependentApplications:        a.DependentApplications,
		SupportedSourceTypes:         a.SupportedSourceTypes,
		SupportedAuthenticationTypes: a.SupportedAuthenticationTypes,
	}
}

// AvailabilityCheckURL returns the application's availability check URL, e.g. where to send the
// request for the client to re-check the application's availability status.
func (at *ApplicationType) AvailabilityCheckURL() *url.URL {
	// Transforms the path-style name to a prefix set in the ENV
	// e.g. /insights/platform/cloud-meter -> CLOUD_METER
	parts := strings.Split(at.Name, "/")
	env_prefix := strings.ToUpper(parts[len(parts)-1])
	env_prefix = strings.ReplaceAll(env_prefix, "-", "_")

	// if the url isn't set don't even try to parse it just return.
	uri, ok := os.LookupEnv(env_prefix + "_AVAILABILITY_CHECK_URL")
	if !ok {
		return nil
	}

	url, err := url.Parse(uri)
	if err != nil {
		return nil
	}

	return url
}
