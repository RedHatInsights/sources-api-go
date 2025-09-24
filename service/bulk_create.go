package service

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
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
func BulkAssembly(req m.BulkCreateRequest, tenant *m.Tenant, user *m.User) (*m.BulkCreateOutput, error) {
	// the output from this request.
	var output m.BulkCreateOutput

	// initiate a transaction that we'll rollback if anything bad happens.
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		var err error

		userResource, err := userResourceFromBulkCreateApplications(user, req.Applications, tenant)
		if err != nil {
			return err
		}

		userResource.User = user

		// parse the sources, then save them in the transaction.
		output.Sources, err = parseSources(req.Sources, tenant, userResource)
		if err != nil {
			return err
		}

		err = tx.Omit(clause.Associations).Create(&output.Sources[0]).Error
		if err != nil {
			return err
		}

		output.Applications, err = parseApplications(req.Applications, &output, tenant, userResource)
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
		output.Authentications, err = linkUpAuthentications(req, &output, tenant, userResource)
		if err != nil {
			return err
		}

		for i := 0; i < len(output.Authentications); i++ {
			err = dao.GetAuthenticationDao(&dao.RequestParams{TenantID: &tenant.Id}).BulkCreate(&output.Authentications[i])
			if err != nil {
				return err
			}

			if strings.ToLower(output.Authentications[i].ResourceType) == "application" {
				applicationAuthentication := m.ApplicationAuthentication{
					// TODO: After vault migration.
					// VaultPath:         output.Authentications[i].Path(),
					ApplicationID:    output.Authentications[i].ResourceID,
					AuthenticationID: output.Authentications[i].DbID,
					TenantID:         tenant.Id,
					Tenant:           *tenant,
				}

				applicationTypeID := findApplicationTypeIdByApplicationID(output.Authentications[i].ResourceID, output.Applications)
				if applicationTypeID != nil {
					applicationType := &m.ApplicationType{}
					if tx.Debug().Where("application_types.id = ?", applicationTypeID).First(&applicationType).Error == nil {
						if userResource.OwnershipPresentForApplication(applicationType.Name) {
							applicationAuthentication.UserID = &userResource.User.Id
						}
					}
				}

				output.ApplicationAuthentications = append(output.ApplicationAuthentications, applicationAuthentication)
			}
		}

		err = tx.Omit(clause.Associations).Create(&output.ApplicationAuthentications).Error
		if err != nil && !errors.Is(err, gorm.ErrEmptySlice) {
			return err
		}

		return nil
	})

	// Log all the created resources when the transaction did not get rolled back.
	if err != nil {
		for _, createdSource := range output.Sources {
			l.Log.WithFields(logrus.Fields{"tenant_id": createdSource.Tenant.Id, "source_id": createdSource.ID, "source_type_id": createdSource.SourceTypeID}).Infof("Source created")
		}

		for _, createdApplication := range output.Applications {
			l.Log.WithFields(logrus.Fields{"tenant_id": createdApplication.Tenant.Id, "source_id": createdApplication.SourceID, "application_id": createdApplication.ID, "application_type_id": createdApplication.ApplicationTypeID}).Infof("Application created")
		}

		for _, createdEndpoint := range output.Endpoints {
			l.Log.WithFields(logrus.Fields{"tenant_id": createdEndpoint.Tenant.Id, "source_id": createdEndpoint.SourceID, "endpoint_id": createdEndpoint.ID}).Infof("Endpoint created")
		}

		for _, createdAuthentication := range output.Authentications {
			l.Log.WithFields(logrus.Fields{"tenant_id": createdAuthentication.Tenant.Id, "resource_id": createdAuthentication.ResourceID, "resource_type": createdAuthentication.ResourceType, "authentication_id": createdAuthentication.ID}).Infof("Authentication created")
		}

		for _, createdAppAuth := range output.ApplicationAuthentications {
			l.Log.WithFields(logrus.Fields{"tenant_id": createdAppAuth.Tenant.Id, "application_id": createdAppAuth.ApplicationID, "authentication_id": createdAppAuth.ApplicationID}).Infof("Authentication created")
		}
	}

	return &output, err
}

func findApplicationTypeIdByApplicationID(applicationID int64, applications []m.Application) *int64 {
	for _, currentApplication := range applications {
		if currentApplication.ID == applicationID {
			return &currentApplication.ApplicationTypeID
		}
	}

	return nil
}

func parseSources(reqSources []m.BulkCreateSource, tenant *m.Tenant, userResource *m.UserResource) ([]m.Source, error) {
	sources := make([]m.Source, len(reqSources))

	for i, source := range reqSources {
		s := m.Source{}

		var (
			sourceType *m.SourceType
			err        error
		)

		switch {
		case source.SourceTypeIDRaw != nil:
			// look up by id if an id was specified
			id, err := util.InterfaceToInt64(source.SourceTypeIDRaw)
			if err != nil {
				return nil, util.NewErrBadRequest(fmt.Sprintf("invalid source type id, original error: %s", err))
			}

			sourceType, err = dao.GetSourceTypeDao().GetById(&id)
			if err != nil {
				return nil, util.NewErrNotFound(fmt.Sprintf("the specified source type was not found: %s", err))
			}

		case source.SourceTypeName != "":
			// look up the source type dynamically....or set it by ID later
			sourceType, err = dao.GetSourceTypeDao().GetByName(source.SourceTypeName)
			if err != nil {
				return nil, util.NewErrBadRequest(fmt.Sprintf("invalid source_type_name for lookup: %v", source.SourceTypeName))
			}

			source.SourceTypeIDRaw = sourceType.Id
		default:
			return nil, util.NewErrBadRequest("no source type present, need either [source_type_name] or [source_type_id]")
		}

		// set up the source type + id for validation
		s.SourceType = *sourceType

		// validate the source request
		err = ValidateSourceCreationRequest(dao.GetSourceDao(&dao.RequestParams{TenantID: &tenant.Id}), &source.SourceCreateRequest)
		if err != nil {
			return nil, util.NewErrBadRequest(fmt.Sprintf("Validation failed: %v", err))
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

		if userResource.OwnershipPresentForSource(s.Name) {
			s.UserID = &userResource.User.Id
		}

		// add it to the list
		sources[i] = s
	}

	return sources, nil
}

func parseApplications(reqApplications []m.BulkCreateApplication, current *m.BulkCreateOutput, tenant *m.Tenant, userResource *m.UserResource) ([]m.Application, error) {
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
			err = dao.GetApplicationTypeDao(&tenant.Id).ApplicationTypeCompatibleWithSourceType(a.ApplicationType.Id, src.SourceType.Id)
			if err != nil {
				return nil, util.NewErrBadRequest("the application type is not compatible with the source type")
			}

			// fill out whats left of the application (spoiler: not much)
			a.Extra = app.Extra
			a.SourceID = src.ID
			a.Tenant = *tenant
			a.TenantID = tenant.Id

			if userResource.OwnershipPresentForSourceAndApplication(app.SourceName, app.ApplicationTypeName) {
				a.UserID = &userResource.User.Id
			}

			applications = append(applications, *a)
		}
	}

	// if all of the applications did not get linked up - there was a problem
	// with the request.
	if len(applications) != len(reqApplications) {
		return nil, util.NewErrBadRequest("failed to link up all applications - check to make sure the names match up")
	}

	return applications, nil
}

func parseEndpoints(reqEndpoints []m.BulkCreateEndpoint, current *m.BulkCreateOutput, tenant *m.Tenant) ([]m.Endpoint, error) {
	endpoints := make([]m.Endpoint, 0)

	for _, endpt := range reqEndpoints {
		e := m.Endpoint{}

		// The source ID needs to be set before validating the endpoint, since otherwise the validation always fails.
		for _, src := range current.Sources {
			if src.Name != endpt.SourceName {
				continue
			}

			// "IDRaw" is the one validated by the validator.
			endpt.SourceIDRaw = src.ID
			// We will pick this one, however, to assign it to the endpoint struct.
			endpt.SourceID = src.ID
		}

		err := ValidateEndpointCreateRequest(dao.GetEndpointDao(&tenant.Id), &endpt.EndpointCreateRequest)
		if err != nil {
			return nil, util.NewErrBadRequest(err)
		}

		e.Scheme = endpt.Scheme
		e.Host = &endpt.Host
		e.Path = &endpt.Path
		e.Port = endpt.Port
		e.VerifySsl = endpt.VerifySsl
		e.SourceID = endpt.SourceID
		e.Tenant = *tenant
		e.TenantID = tenant.Id

		endpoints = append(endpoints, e)
	}

	// if all of the endpoints did not get linked up - there was a problem
	// with the request.
	if len(endpoints) != len(reqEndpoints) {
		return nil, util.NewErrBadRequest("failed to link up all endpoints - check to make sure the names match up")
	}

	return endpoints, nil
}

func linkUpAuthentications(req m.BulkCreateRequest, current *m.BulkCreateOutput, tenant *m.Tenant, userResource *m.UserResource) ([]m.Authentication, error) {
	authentications := make([]m.Authentication, 0)

	for _, auth := range req.Authentications {
		a := m.Authentication{}

		a.ResourceType = util.Capitalize(auth.ResourceType)
		a.AuthType = auth.AuthType
		a.Username = util.StringValueOrNil(auth.Username)

		// pull the password & extra properly per secret store
		err := a.SetPassword(auth.Password)
		if err != nil {
			return nil, err
		}

		err = a.SetExtra(auth.Extra)
		if err != nil {
			return nil, err
		}

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
				_, err = dao.GetSourceDao(&dao.RequestParams{TenantID: &tenant.Id, UserID: &userResource.User.Id}).GetById(&id)
				if err == nil {
					l.Log.Debugf("Found existing Source with id %v, adding to list and continuing", id)
					a.ResourceID = id
					authentications = append(authentications, a)

					continue
				}
			case "application":
				_, err = dao.GetApplicationDao(&dao.RequestParams{TenantID: &tenant.Id, UserID: &userResource.User.Id}).GetById(&id)
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

			if userResource.OwnershipPresentForSource(current.Sources[0].Name) {
				a.UserID = &userResource.User.Id
			}

			l.Log.Infof("Source Authentication does not need linked - continuing")

		case "application":
			id, err := linkupApplication(auth.ResourceName, current.Applications, &tenant.Id)
			if err != nil {
				return nil, util.NewErrBadRequest(err)
			}

			a.ResourceID = id
			a.SourceID = current.Sources[0].ID

			if userResource.OwnershipPresentForSourceAndApplication(current.Sources[0].Name, auth.ResourceName) {
				a.UserID = &userResource.User.Id
			}

		case "endpoint":
			id, err := linkupEndpoint(auth.ResourceName, current.Endpoints)
			if err != nil {
				return nil, util.NewErrBadRequest(err)
			}

			a.ResourceID = id
			a.SourceID = current.Sources[0].ID

		default:
			return nil, util.NewErrBadRequest("failed to link authentication: no resource type present")
		}

		auth.ResourceIDRaw = a.ResourceID
		auth.ResourceType = a.ResourceType

		err = ValidateAuthenticationCreationRequest(&auth.AuthenticationCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("validation failed for authentication: %w", err)
		}

		// checking to make sure the polymorphic relationship is set.
		if a.ResourceID != 0 && a.ResourceType != "" {
			authentications = append(authentications, a)
		}
	}

	if len(authentications) != len(req.Authentications) {
		return nil, util.NewErrBadRequest("failed to link up all authentications - check to make sure the names match up")
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
			return nil, util.NewErrBadRequest("application type id cannot be converted to an integer")
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
			return nil, util.NewErrBadRequest(fmt.Sprintf("failed to lookup application_type_name %s", reqApplication.ApplicationTypeName))
		}

		a.ApplicationType = *appType
		a.ApplicationTypeID = appType.Id

	default:
		return nil, util.NewErrBadRequest("no application type present, need either [application_type_name] or [application_type_id]")
	}

	return &a, nil
}

func loadUserResourceSettingFromBulkCreateApplication(userResource *m.UserResource, bulkCreateApplication *m.BulkCreateApplication, tenant *m.Tenant) error {
	app, err := applicationFromBulkCreateApplication(bulkCreateApplication, tenant)
	if err != nil {
		return err
	} else if app.ApplicationType.UserResourceOwnership() {
		userResource.AddSourceAndApplicationTypeNames(bulkCreateApplication.SourceName, bulkCreateApplication.ApplicationTypeName)
	}

	return nil
}

func userResourceFromBulkCreateApplications(user *m.User, applications []m.BulkCreateApplication, tenant *m.Tenant) (*m.UserResource, error) {
	userResource := &m.UserResource{User: user}

	for _, reqApp := range applications {
		err := loadUserResourceSettingFromBulkCreateApplication(userResource, &reqApp, tenant)
		if err != nil {
			return nil, err
		}
	}

	return userResource, nil
}
