package dao

import (
	"fmt"

	m "github.com/RedHatInsights/sources-api-go/model"
)

func GetFrom(resourceType string) (*m.EventModelDao, error) {
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
