package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
	Oh boy. The big one.

	So basically this function goes through a the parsed BulkCreateRequest model
	and creates the resources in this order:

	1. Sources
	2. Endpoints/Applications

	It dynamically looks up both the SourceType as well as ApplicationType if
	given the *_type_name paremeters.

	3. Saving the Authentications
	4. Saving the ApplicationAuthentications if necessary
*/
func BulkAssembly(req m.BulkCreateRequest, tenant *m.Tenant) (*m.BulkCreateOutput, error) {
	// the output from this request.
	var output m.BulkCreateOutput

	// initiate a transaction that we'll rollback if anything bad happens.
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		var err error

		// parse the sources, then save them in the transaction.
		output.Sources, err = parseSources(req.Sources, tenant)
		if err != nil {
			return err
		}
		err = tx.Omit(clause.Associations).Create(&output.Sources[0]).Error
		if err != nil {
			return err
		}

		output.Applications, err = parseApplications(req.Applications, &output, tenant)
		if err != nil {
			return err
		}
		err = tx.Omit(clause.Associations).Create(&output.Applications).Error
		if err != nil && !errors.Is(err, gorm.ErrEmptySlice) {
			return err
		}

		output.Endpoints, err = parseEndpoints(req.Endpoints, &output, tenant)
		if err != nil {
			return err
		}
		err = tx.Create(&output.Endpoints).Error
		if err != nil && !errors.Is(err, gorm.ErrEmptySlice) {
			return err
		}

		// link up the authentications to their polymorphic relations.
		output.Authentications, err = linkUpAuthentications(req, &output, tenant)
		if err != nil {
			return err
		}

		for i := 0; i < len(output.Authentications); i++ {
			err = dao.GetAuthenticationDao(&tenant.Id).BulkCreate(&output.Authentications[i])
			if err != nil {
				return err
			}

			if strings.ToLower(output.Authentications[i].ResourceType) == "application" {
				output.ApplicationAuthentications = append(output.ApplicationAuthentications, m.ApplicationAuthentication{
					// TODO: After vault migration.
					// VaultPath:         output.Authentications[i].Path(),
					ApplicationID:    output.Authentications[i].ResourceID,
					AuthenticationID: output.Authentications[i].DbID,
					TenantID:         tenant.Id,
					Tenant:           *tenant,
				})
			}
		}

		err = tx.Omit(clause.Associations).Create(&output.ApplicationAuthentications).Error
		if err != nil && !errors.Is(err, gorm.ErrEmptySlice) {
			return err
		}

		return nil
	})

	return &output, err
}

func parseSources(reqSources []m.BulkCreateSource, tenant *m.Tenant) ([]m.Source, error) {
	sources := make([]m.Source, len(reqSources))

	for i, source := range reqSources {
		s := m.Source{}
		var sourceType *m.SourceType
		var err error

		switch {
		case source.SourceTypeIDRaw != nil:
			// look up by id if an id was specified
			id, err := util.InterfaceToInt64(source.SourceTypeIDRaw)
			if err != nil {
				return nil, err
			}

			sourceType, err = dao.GetSourceTypeDao().GetById(&id)
			if err != nil {
				return nil, err
			}

		case source.SourceTypeName != "":
			// look up the source type dynamically....or set it by ID later
			sourceType, err = dao.GetSourceTypeDao().GetByName(source.SourceTypeName)
			if err != nil {
				return nil, fmt.Errorf("invalid source_type_name for lookup: %v", source.SourceTypeName)
			}

			source.SourceTypeIDRaw = sourceType.Id
		default:
			return nil, fmt.Errorf("no source type present, need either [source_type_name] or [source_type_id]")
		}

		// set up the source type + id for validation
		s.SourceType = *sourceType

		// validate the source request
		err = ValidateSourceCreationRequest(dao.GetSourceDao(&tenant.Id), &source.SourceCreateRequest)
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
		s.AvailabilityStatus = source.AvailabilityStatus
		s.SourceTypeID = *source.SourceTypeID
		s.Tenant = *tenant
		s.TenantID = tenant.Id

		// populate the child relation slices
		s.Endpoints = make([]m.Endpoint, 0)
		s.Applications = make([]m.Application, 0)
		s.Authentications = make([]m.Authentication, 0)

		// add it to the list
		sources[i] = s
	}

	return sources, nil
}

func parseApplications(reqApplications []m.BulkCreateApplication, current *m.BulkCreateOutput, tenant *m.Tenant) ([]m.Application, error) {
	applications := make([]m.Application, 0)

	for _, app := range reqApplications {
		a, err := applicationFromBulkCreateApplication(&app, tenant)
		if err != nil {
			return nil, err
		}

		// loop through and find the source which this application belongs to.
		for _, src := range current.Sources {
			if src.Name != app.SourceName {
				continue
			}

			// check compatibility with the source type
			err := dao.GetApplicationTypeDao(&tenant.Id).ApplicationTypeCompatibleWithSourceType(a.ApplicationType.Id, src.SourceType.Id)
			if err != nil {
				return nil, err
			}

			// fill out whats left of the application (spoiler: not much)
			a.Extra = app.Extra
			a.SourceID = src.ID
			a.Tenant = *tenant
			a.TenantID = tenant.Id

			applications = append(applications, *a)
		}
	}

	// if all of the applications did not get linked up - there was a problem
	// with the request.
	if len(applications) != len(reqApplications) {
		return nil, fmt.Errorf("failed to link up all applications - check to make sure the names match up")
	}

	return applications, nil
}

func parseEndpoints(reqEndpoints []m.BulkCreateEndpoint, current *m.BulkCreateOutput, tenant *m.Tenant) ([]m.Endpoint, error) {
	endpoints := make([]m.Endpoint, 0)

	for _, endpt := range reqEndpoints {
		e := m.Endpoint{}

		err := ValidateEndpointCreateRequest(dao.GetEndpointDao(&tenant.Id), &endpt.EndpointCreateRequest)
		if err != nil {
			return nil, err
		}

		e.Scheme = endpt.Scheme
		e.Host = &endpt.Host
		e.Path = &endpt.Path
		e.Port = endpt.Port
		e.VerifySsl = endpt.VerifySsl
		e.Tenant = *tenant
		e.TenantID = tenant.Id

		for _, src := range current.Sources {
			if src.Name != endpt.SourceName {
				continue
			}

			e.SourceID = src.ID
			endpoints = append(endpoints, e)
		}
	}

	// if all of the endpoints did not get linked up - there was a problem
	// with the request.
	if len(endpoints) != len(reqEndpoints) {
		return nil, fmt.Errorf("failed to link up all endpoints - check to make sure the names match up")
	}

	return endpoints, nil
}

func linkUpAuthentications(req m.BulkCreateRequest, current *m.BulkCreateOutput, tenant *m.Tenant) ([]m.Authentication, error) {
	authentications := make([]m.Authentication, 0)

	for _, auth := range req.Authentications {
		a := m.Authentication{}

		a.ResourceType = util.Capitalize(auth.ResourceType)
		a.AuthType = auth.AuthType
		a.Username = util.StringValueOrNil(auth.Username)
		a.Password = auth.Password
		// TODO: set based on vault or not.
		a.ExtraDb, _ = json.Marshal(auth.Extra)
		a.Name = auth.Name
		a.Tenant = *tenant
		a.TenantID = tenant.Id

		// if an id was passed in we're just adding an authentication to
		// an already-existing resource.
		id, err := strconv.ParseInt(auth.ResourceName, 10, 64)
		if err == nil {
			var err error
			switch strings.ToLower(auth.ResourceType) {
			case "source":
				_, err = dao.GetSourceDao(&tenant.Id).GetById(&id)
				if err == nil {
					l.Log.Debugf("Found existing Source with id %v, adding to list and continuing", id)
					a.ResourceID = id
					authentications = append(authentications, a)
					continue
				}
			case "application":
				_, err = dao.GetApplicationDao(&tenant.Id).GetById(&id)
				if err == nil {
					l.Log.Debugf("Found existing Application with id %v, adding to list and continuing", id)
					a.ResourceID = id
					authentications = append(authentications, a)
					continue
				}
			case "endpoint":
				_, err = dao.GetEndpointDao(&tenant.Id).GetById(&id)
				if err == nil {
					l.Log.Debugf("Found existing Endpoint with id %v, adding to list and continuing", id)
					a.ResourceID = id
					authentications = append(authentications, a)
					continue
				}
			}
		}

		// lookup the polymorphic resource based on the resource type + name
		switch strings.ToLower(auth.ResourceType) {
		case "source":
			a.ResourceID = current.Sources[0].ID
			a.SourceID = current.Sources[0].ID
			l.Log.Infof("Source Authentication does not need linked - continuing")

		case "application":
			id, err := linkupApplication(auth.ResourceName, current.Applications, &tenant.Id)
			if err != nil {
				return nil, err
			}

			a.ResourceID = id
			a.SourceID = current.Sources[0].ID

		case "endpoint":
			id, err := linkupEndpoint(auth.ResourceName, current.Endpoints)
			if err != nil {
				return nil, err
			}

			a.ResourceID = id
			a.SourceID = current.Sources[0].ID

		default:
			return nil, fmt.Errorf("failed to link authentication: no resource type present")
		}

		// checking to make sure the polymorphic relationship is set.
		if a.ResourceID != 0 && a.ResourceType != "" {
			authentications = append(authentications, a)
		}
	}

	if len(authentications) != len(req.Authentications) {
		return nil, fmt.Errorf("failed to link up all authentications - check to make sure the names match up")
	}

	return authentications, nil
}

// authentications are attached to an application that has the same "resource
// name" which is passed in the payload.
func linkupApplication(name string, apps []m.Application, tenantID *int64) (int64, error) {
	at, err := dao.GetApplicationTypeDao(tenantID).GetByName(name)
	if err != nil {
		return 0, err
	}

	for _, app := range apps {
		if at.Id == app.ApplicationTypeID {
			return app.ID, nil
		}
	}

	return 0, fmt.Errorf("failed to find application for authentication type %v", at.Name)
}

// authentications are attached to an endpoint that has the same hostname passed
// in the payload.
func linkupEndpoint(name string, endpoints []m.Endpoint) (int64, error) {
	for _, endpt := range endpoints {
		if strings.Contains(strings.ToLower(*endpt.Host), strings.ToLower(name)) {
			return endpt.ID, nil
		}
	}

	return 0, fmt.Errorf("failed to find endpoint for hostname %v", name)
}

// send all the messages on the event-stream for what we created. this involves
// doing some checks for superkey related things etc.
func SendBulkMessages(out *m.BulkCreateOutput, headers []kafka.Header, identity string) {
	// do this async, since it could potentially take a while.
	go func() {
		for i := range out.Sources {
			src := out.Sources[i]
			err := RaiseEvent("Source.create", &src, headers)
			if err != nil {
				l.Log.Warnf("Failed to raise event: %v", err)
			}
		}

		for i := range out.Endpoints {
			endpt := out.Endpoints[i]
			err := RaiseEvent("Endpoint.create", &endpt, headers)
			if err != nil {
				l.Log.Warnf("Failed to raise event: %v", err)
			}
		}

		for i := range out.Applications {
			app := out.Applications[i]
			if out.Sources[0].AppCreationWorkflow == m.AccountAuth {
				err := SendSuperKeyCreateRequest(&app, headers)
				if err != nil {
					l.Log.Warnf("Error sending superkey create request: %v", err)
				}
			} else {
				// only raise create if it was _NOT_ a superkey source
				err := RaiseEvent("Application.create", &app, headers)
				if err != nil {
					l.Log.Warnf("Failed to raise event: %v", err)
				}
			}
		}

		for i := range out.ApplicationAuthentications {
			appAuth := out.ApplicationAuthentications[i]
			err := RaiseEvent("ApplicationAuthentication.create", &appAuth, headers)
			if err != nil {
				l.Log.Warnf("Failed to raise event: %v", err)
			}
		}

		for i := range out.Authentications {
			auth := out.Authentications[i]
			err := RaiseEvent("Authentication.create", &auth, headers)
			if err != nil {
				l.Log.Warnf("Failed to raise event: %v", err)
			}
		}
	}()
}

func applicationFromBulkCreateApplication(reqApplication *m.BulkCreateApplication, tenant *m.Tenant) (*m.Application, error) {
	a := m.Application{}

	switch {
	case reqApplication.ApplicationTypeIDRaw != nil:
		// look up by id if an id was specified
		id, err := util.InterfaceToInt64(reqApplication.ApplicationTypeIDRaw)
		if err != nil {
			return nil, err
		}

		appType, err := dao.GetApplicationTypeDao(&tenant.Id).GetById(&id)
		if err != nil {
			return nil, err
		}

		a.ApplicationType = *appType
		a.ApplicationTypeID = appType.Id

	case reqApplication.ApplicationTypeName != "":
		// dynamically look up the application type by name if passed
		appType, err := dao.GetApplicationTypeDao(&tenant.Id).GetByName(reqApplication.ApplicationTypeName)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup application_type_name %v", reqApplication.ApplicationTypeName)
		}

		a.ApplicationType = *appType
		a.ApplicationTypeID = appType.Id

	default:
		return nil, fmt.Errorf("no application type present, need either [application_type_name] or [application_type_id]")
	}

	return &a, nil
}
