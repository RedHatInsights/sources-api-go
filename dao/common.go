package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
)

const (
	defaultLimit  = 100
	defaultOffset = 0
)

func GetFromResourceType(resourceType string) (*m.EventModelDao, error) {
	var resource m.EventModelDao
	switch resourceType {
	case "Source":
		resource = &SourceDaoImpl{}
	case "Endpoint":
		resource = &EndpointDaoImpl{}
	case "Application":
		resource = &ApplicationDaoImpl{}
	case "Authentication":
		resource = &AuthenticationDaoImpl{}
	default:
		return nil, fmt.Errorf("invalid resource_type (%s) to get DAO instance", resourceType)
	}

	return &resource, nil
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

	bulkMessage["application_authentications"] = []m.ApplicationAuthenticationEvent{}

	authDao := &AuthenticationDaoImpl{TenantID: &source.TenantID}
	authenticationsByResource, err := authDao.AuthenticationsByResource(authentication)
	if err != nil {
		return nil, err
	}

	authentications := make([]interface{}, len(authenticationsByResource))
	for i := 0; i < len(authenticationsByResource); i++ {
		authenticationsByResource[i].Tenant = source.Tenant
		authentications[i] = authenticationsByResource[i].ToEvent()
	}

	bulkMessage["authentications"] = authentications

	return bulkMessage, nil
}
