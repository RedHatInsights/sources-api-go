package dao

import (
	"fmt"
	"strings"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

const (
	DEFAULT_LIMIT  = 100
	DEFAULT_OFFSET = 0
)

func GetFromResourceType(resourceType string) (*m.EventModelDao, error) {
	var resource m.EventModelDao
	switch strings.ToLower(resourceType) {
	case "source":
		resource = GetSourceDao(nil)
	case "endpoint":
		resource = GetEndpointDao(nil)
	case "application":
		resource = GetApplicationDao(nil)
	case "authentication":
		resource = GetAuthenticationDao(nil)
	default:
		return nil, fmt.Errorf("invalid resource_type (%s) to get DAO instance", resourceType)
	}

	return &resource, nil
}

func GetAvailabilityStatusFromStatusMessage(tenantID int64, resourceID string, resourceType string) (string, error) {
	switch resourceType {
	case "Source":
		recordID, err := util.InterfaceToInt64(resourceID)
		if err != nil {
			return "", err
		}
		resource, err := GetSourceDao(&tenantID).GetById(&recordID)
		if err != nil {
			return "", err
		}
		return resource.AvailabilityStatus, err
	case "Endpoint":
		recordID, err := util.InterfaceToInt64(resourceID)
		if err != nil {
			return "", err
		}
		resource, err := GetEndpointDao(&tenantID).GetById(&recordID)
		if err != nil {
			return "", err
		}
		return resource.AvailabilityStatus, err
	case "Application":
		recordID, err := util.InterfaceToInt64(resourceID)
		if err != nil {
			return "", err
		}
		resource, err := GetApplicationDao(&tenantID).GetById(&recordID)
		if err != nil {
			return "", err
		}
		return resource.AvailabilityStatus, err
	case "Authentication":
		resource, err := GetAuthenticationDao(&tenantID).GetById(resourceID)
		if err != nil {
			return "", err
		}
		return resource.AvailabilityStatus, err
	default:
		return "", fmt.Errorf("invalid resource_type (%s) to get DAO instance", resourceType)
	}
}

/*
	Method generates bulk message for Source record.
	authentication - specify resource (ResourceID and ResourceType) of
                     which authentications are fetched to BulkMessage
                   - specify application_authentications in BulkMessage otherwise
                     application_authentications are obtained from authentications UIDs
					 in BulkMessage
*/
func BulkMessageFromSource(source *m.Source, authentication *m.Authentication) (map[string]interface{}, error) {
	result := DB.
		Preload("Tenant").
		Preload("Applications.Tenant").
		Preload("Endpoints.Tenant").
		Find(&source)

	if result.Error != nil {
		return nil, result.Error
	}

	bulkMessage := map[string]interface{}{}
	bulkMessage["source"] = source.ToEvent()

	endpoints := make([]interface{}, len(source.Endpoints))
	for i, endpoint := range source.Endpoints {
		endpoints[i] = endpoint.ToEvent()
	}

	bulkMessage["endpoints"] = endpoints

	applications := make([]interface{}, len(source.Applications))
	for i, application := range source.Applications {
		applications[i] = application.ToEvent()
	}

	bulkMessage["applications"] = applications

	authDao := GetAuthenticationDao(&source.TenantID)
	authenticationsByResource, err := authDao.AuthenticationsByResource(authentication)
	if err != nil {
		return nil, err
	}

	authentications := make([]interface{}, len(authenticationsByResource))
	for i := 0; i < len(authenticationsByResource); i++ {
		authenticationsByResource[i].Tenant = source.Tenant
		authentications[i] = authenticationsByResource[i].ToEvent()
	}

	applicationAuthenticationDao := GetApplicationAuthenticationDao(&source.TenantID)
	applicationAuthenticationsFromResource, err := applicationAuthenticationDao.ApplicationAuthenticationsByResource(authentication.ResourceType, source.Applications, authenticationsByResource)

	if err != nil {
		return nil, err
	}

	applicationAuthentications := make([]interface{}, len(applicationAuthenticationsFromResource))
	for i := 0; i < len(applicationAuthenticationsFromResource); i++ {
		applicationAuthenticationsFromResource[i].Tenant = source.Tenant
		applicationAuthentications[i] = applicationAuthenticationsFromResource[i].ToEvent()
	}

	bulkMessage["application_authentications"] = applicationAuthentications
	bulkMessage["authentications"] = authentications

	return bulkMessage, nil
}
