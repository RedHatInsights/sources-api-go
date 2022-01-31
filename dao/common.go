package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
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

func BulkMessageFromSource(source *m.Source) (map[string]interface{}, error) {
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

	endpoints := make([]m.EndpointEvent, len(source.Endpoints))
	for i, endpoint := range source.Endpoints {
		endpoints[i] = *endpoint.ToEvent()
	}

	bulkMessage["endpoints"] = endpoints

	applications := make([]m.ApplicationEvent, len(source.Applications))
	for i, application := range source.Applications {
		event, ok := application.ToEvent().(*m.ApplicationEvent)
		if !ok {
			return nil, fmt.Errorf("failed to create application event from application")
		}

		applications[i] = *event
	}

	bulkMessage["applications"] = applications

	bulkMessage["authentications"] = []m.Authentication{}
	bulkMessage["application_authentications"] = []m.ApplicationAuthenticationEvent{}

	return bulkMessage, nil
}
