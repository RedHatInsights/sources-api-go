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

	endpoints := make([]m.EndpointEvent, len(source.Endpoints))
	for i, endpoint := range source.Endpoints {
		endpoints[i] = *endpoint.ToEvent()
	}

	bulkMessage["endpoints"] = endpoints

	applications := make([]m.ApplicationEvent, len(source.Applications))
	for i, application := range source.Applications {
		applications[i] = *application.ToEvent()
	}

	bulkMessage["applications"] = applications

	err := AddAuthenticationEvents(bulkMessage, authentication, source.TenantID)
	if err != nil {
		return nil, err
	}

	AddApplicationsAuthenticationEvents(bulkMessage, authentication.ApplicationAuthentications)

	return bulkMessage, nil
}

func AddApplicationsAuthentications(bulkMessage map[string]interface{}, resourceApplicationsAuthentications []m.ApplicationAuthentication) {
	var aa []m.ApplicationAuthentication

	if len(resourceApplicationsAuthentications) == 0 {
		var authUIDs []string

		as, success := bulkMessage["authentications"].([]m.Authentication)

		if success {
			for _, auth := range as {
				authUIDs = append(authUIDs, auth.ID)
			}
		}

		authenticationEvents, success := bulkMessage["authentications"].([]m.AuthenticationEvent)

		if success {
			for _, auth := range authenticationEvents {
				authUIDs = append(authUIDs, auth.ID)
			}
		}

		DB.Preload("Tenant").Where("authentication_uid IN ?", authUIDs).Find(&aa)
	}

	if aa != nil {
		bulkMessage["application_authentications"] = aa
	} else {
		bulkMessage["application_authentications"] = []m.ApplicationAuthentication{}
	}
}

func AddApplicationsAuthenticationEvents(bulkMessage map[string]interface{}, resourceApplicationsAuthentications []m.ApplicationAuthentication) {
	applicationsAuthentications := make([]m.ApplicationAuthenticationEvent, len(resourceApplicationsAuthentications))

	for i, auth := range resourceApplicationsAuthentications {
		applicationsAuthentications[i] = *auth.ToEvent()
	}

	if len(resourceApplicationsAuthentications) == 0 {
		var aa []m.ApplicationAuthentication
		var authUIDs []string

		authenticationEvents, success := bulkMessage["authentications"].([]m.AuthenticationEvent)

		if success {
			for _, auth := range authenticationEvents {
				authUIDs = append(authUIDs, auth.ID)
			}
		}

		DB.Preload("Tenant").Where("authentication_uid IN ?", authUIDs).Find(&aa)
		applicationsAuthentications = make([]m.ApplicationAuthenticationEvent, len(aa))
		for i, auth := range aa {
			applicationsAuthentications[i] = *auth.ToEvent()
		}
	}

	bulkMessage["application_authentications"] = applicationsAuthentications
}

func AddAuthenticationEvents(bulkMessage map[string]interface{}, authentication *m.Authentication, tenantID int64) error {
	err := AddAuthentications(bulkMessage, authentication, tenantID)
	aa, success := bulkMessage["authentications"].([]m.Authentication)
	if !success {
		panic("unable to cast bulkMessage authentications")
	}

	authentications := make([]m.AuthenticationEvent, len(aa))

	for index, auth := range aa {
		authentications[index] = *auth.ToEvent()
	}

	if authentications == nil {
		authentications = []m.AuthenticationEvent{}
	}

	bulkMessage["authentications"] = authentications

	return err
}

func AddAuthentications(bulkMessage map[string]interface{}, authentication *m.Authentication, tenantID int64) error {
	var err error
	var resourceAuthentications []m.Authentication
	defaultLimit := 100
	defaultOffset := 0

	authenticationDao := &AuthenticationDaoImpl{TenantID: &tenantID}
	switch authentication.ResourceType {
	case "Source":
		resourceAuthentications, _, err = authenticationDao.ListForSource(authentication.ResourceID, defaultLimit, defaultOffset, nil)
	case "Endpoint":
		resourceAuthentications, _, err = authenticationDao.ListForEndpoint(authentication.ResourceID, defaultLimit, defaultOffset, nil)
	case "Application":
		resourceAuthentications, _, err = authenticationDao.ListForApplication(authentication.ResourceID, defaultLimit, defaultOffset, nil)
	default:
		return fmt.Errorf("unable to fetch authentications for %s", authentication.ResourceType)
	}

	if err != nil {
		return err
	}

	tenant := m.Tenant{Id: tenantID}
	result := DB.First(&tenant)
	if result.Error != nil {
		return result.Error
	}

	for i, auth := range resourceAuthentications {
		auth.TenantID = tenantID
		auth.Tenant = m.Tenant{ExternalTenant: tenant.ExternalTenant}
		resourceAuthentications[i] = auth
	}

	bulkMessage["authentications"] = resourceAuthentications

	return nil
}
