package service

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

/*
	Oh boy. The big one.

	So basically this function goes through a the parsed BulkCreateRequest model
	and creates the resources in this order:

	1. Sources
	2. Endpoints/Applications
	3. Authentications

	It dynamically looks up both the SourceType as well as ApplicationType if
	given the *_type_name paremeters.

	For the final output the base Source models have the relations built out
	below them so a single Save() call will persist all the models at once.
*/
func ParseBulkCreateRequest(req m.BulkCreateRequest, tenantID *int64) (*m.BulkCreateOutput, error) {
	var err error
	var output m.BulkCreateOutput

	output.Sources, err = parseSources(req.Sources, tenantID)
	if err != nil {
		return nil, err
	}

	output.Applications, err = parseApplications(req.Applications, &output, tenantID)
	if err != nil {
		return nil, err
	}

	output.Endpoints, err = parseEndpoints(req.Endpoints, &output, tenantID)
	if err != nil {
		return nil, err
	}

	output.Authentications, err = parseAuthentications(req.Authentications, &output, tenantID)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func parseSources(reqSources []m.BulkCreateSource, tenantID *int64) ([]m.Source, error) {
	sources := make([]m.Source, len(reqSources))

	for i, source := range reqSources {
		s := m.Source{}

		// look up the source type dynamically....or set it by ID later
		sourceType, err := dao.GetSourceTypeDao().GetByName(source.SourceTypeName)
		if err != nil {
			return nil, fmt.Errorf("invalid source_type_name for lookup: %v", source.SourceTypeName)
		}

		// set up the source type + id for validation
		s.SourceType = *sourceType
		source.SourceCreateRequest.SourceTypeIDRaw = sourceType.Id

		// validate the source request
		err = ValidateSourceCreationRequest(dao.GetSourceDao(tenantID), &source.SourceCreateRequest)
		if err != nil {
			return nil, err
		}

		// copy the fields into the alloc'd source struct
		s.Name = *source.Name
		s.Uid = source.Uid
		s.Version = source.Version
		s.Imported = source.Imported
		s.SourceRef = source.SourceRef
		s.AppCreationWorkflow = source.AppCreationWorkflow
		s.AvailabilityStatus = m.AvailabilityStatus{AvailabilityStatus: source.AvailabilityStatus}
		s.TenantID = *tenantID

		// populate the child relation slices
		s.Endpoints = make([]m.Endpoint, 0)
		s.Applications = make([]m.Application, 0)
		// s.Authentications = make([]*m.Authentication, 0)

		// add it to the list
		sources[i] = s
	}

	return sources, nil
}

func parseApplications(reqApplications []m.BulkCreateApplication, current *m.BulkCreateOutput, tenantID *int64) ([]m.Application, error) {
	applications := make([]m.Application, 0)

	for _, app := range reqApplications {
		a := m.Application{}

		switch {
		case app.ApplicationTypeIDRaw != nil:
			// look up by id if an id was specified
			id, err := util.InterfaceToInt64(app.ApplicationTypeIDRaw)
			if err != nil {
				return nil, err
			}

			apptype, err := dao.GetApplicationTypeDao(tenantID).GetById(&id)
			if err != nil {
				return nil, err
			}

			a.ApplicationType = *apptype
		case app.ApplicationTypeName != "":
			// dynamically look up the application type by name if passed
			apptype, err := dao.GetApplicationTypeDao(tenantID).GetByName(app.ApplicationTypeName)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup application_type_name %v", app.ApplicationTypeName)
			}

			a.ApplicationType = *apptype
		default:
			return nil, fmt.Errorf("no application type present, need either [application_type_name] or [application_type_id]")
		}

		// loop through and find the source which this application belongs to.
		for i, src := range current.Sources {
			if src.Name != app.SourceName {
				continue
			}

			// check compatibility with the source type
			err := dao.GetApplicationTypeDao(tenantID).ApplicationTypeCompatibleWithSourceType(a.ApplicationType.Id, src.SourceType.Id)
			if err != nil {
				return nil, err
			}

			// fill out whats left of the application (spoiler: not much)
			a.Extra = app.Extra
			a.TenantID = *tenantID

			// Add the application to the source's list. It'll get persisted all
			// at once when we call the final save.
			//
			// NOTE: need to use the index here because for...range creates a
			// copy of the source!
			applications = append(applications, a)
			current.Sources[i].Applications = append(current.Sources[i].Applications, a)
		}
	}

	// if all of the applications did not get linked up - there was a problem
	// with the request.
	if len(applications) != len(reqApplications) {
		return nil, fmt.Errorf("failed to link up all applications - check to make sure the names match up")
	}

	return applications, nil
}

func parseEndpoints(reqEndpoints []m.BulkCreateEndpoint, current *m.BulkCreateOutput, tenantID *int64) ([]m.Endpoint, error) {
	endpoints := make([]m.Endpoint, 0)

	for _, endpt := range reqEndpoints {
		e := m.Endpoint{}

		err := ValidateEndpointCreateRequest(dao.GetEndpointDao(tenantID), &endpt.EndpointCreateRequest)
		if err != nil {
			return nil, err
		}

		e.Scheme = endpt.Scheme
		e.Host = &endpt.Host
		e.Path = &endpt.Path
		e.Port = endpt.Port
		e.VerifySsl = endpt.VerifySsl
		e.TenantID = *tenantID

		for i, src := range current.Sources {
			if src.Name != endpt.SourceName {
				continue
			}

			endpoints = append(endpoints, e)
			current.Sources[i].Endpoints = append(current.Sources[i].Endpoints, e)
		}
	}

	// if all of the endpoints did not get linked up - there was a problem
	// with the request.
	if len(endpoints) != len(reqEndpoints) {
		return nil, fmt.Errorf("failed to link up all endpoints - check to make sure the names match up")
	}

	return endpoints, nil
}

func parseAuthentications(reqAuthentications []m.BulkCreateAuthentication, current *m.BulkCreateOutput, tenantID *int64) ([]m.Authentication, error) {
	authentications := make([]m.Authentication, 0)

	for _, auth := range reqAuthentications {
		a := m.Authentication{}

		switch strings.ToLower(auth.ResourceType) {
		case "source":
		case "application":
		case "endpoint":
		default:
			return nil, fmt.Errorf("failed to link authentication: no resource type present")
		}

		// copy over fields
		a.AuthType = auth.AuthType
		a.TenantID = *tenantID

		//TODO: the rest.

		authentications = append(authentications, a)
	}

	if len(authentications) != len(reqAuthentications) {
		return nil, fmt.Errorf("failed to link up all authentications - check to make sure the names match up")
	}

	return authentications, nil
}
