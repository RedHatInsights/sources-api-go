package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getApplicationDao func(c echo.Context) (dao.ApplicationDao, error)

func getApplicationDaoWithTenant(c echo.Context) (dao.ApplicationDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetApplicationDao(&tenantId), nil
}

func ApplicationList(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	var (
		applications []m.Application
		count        int64
	)

	applications, count, err = applicationDB.List(limit, offset, filters)
	if err != nil {
		return err
	}

	c.Logger().Infof("tenant: %v", *applicationDB.Tenant())

	out := make([]interface{}, len(applications))
	for i := 0; i < len(applications); i++ {
		out[i] = applications[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationGet(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	c.Logger().Infof("Getting Application ID %v", id)

	app, err := applicationDB.GetById(&id)

	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}

func ApplicationCreate(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	input := &m.ApplicationCreateRequest{}
	if err = c.Bind(input); err != nil {
		return err
	}

	err = service.ValidateApplicationCreateRequest(input)
	if err != nil {
		return util.NewErrBadRequest(fmt.Sprintf("Validation failed: %s", err.Error()))
	}

	application := &m.Application{
		Extra:             input.Extra,
		ApplicationTypeID: input.ApplicationTypeID,
		SourceID:          input.SourceID,
	}

	err = applicationDB.Create(application)
	if err != nil {
		return err
	}

	accountNumber, err := getAccountNumberFromEchoContext(c)
	if err != nil {
		c.Logger().Warn(err)
	}

	application.Tenant = m.Tenant{Id: application.TenantID, ExternalTenant: accountNumber}
	setEventStreamResource(c, application)

	// do not raise if it is a superkey application. The worker will post back
	// with the resources and then we raise the create event.
	if applicationDB.IsSuperkey(application.ID) {
		c.Set("skip_raise", true)

		forwardableHeaders, err := service.ForwadableHeaders(c)
		if err != nil {
			return err
		}

		// do the rest async. Don't want to be tied to kafka.
		go func() {
			err := service.SendSuperKeyCreateRequest(application, forwardableHeaders)
			if err != nil {
				c.Logger().Warnf("Error sending Superkey Create Request: %v", err)
			}
		}()
	}

	return c.JSON(http.StatusCreated, application.ToResponse())
}

func ApplicationEdit(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	app, err := applicationDB.GetByIdWithPreload(&id, "Tenant", "Source")
	if err != nil {
		return util.NewErrNotFound("application")
	}

	// Store the previous status before updating the application.
	previousStatus := app.AvailabilityStatus
	var statusFromRequest *string

	if app.PausedAt != nil {
		input := &m.ResourceEditPausedRequest{}
		if err := c.Bind(input); err != nil {
			return util.NewErrBadRequest(err)
		}
		statusFromRequest = input.AvailabilityStatus
		err := app.UpdateFromRequestPaused(input)
		if err != nil {
			return util.NewErrBadRequest(err)
		}
	} else {
		input := &m.ApplicationEditRequest{}
		if err := c.Bind(input); err != nil {
			return util.NewErrBadRequest(err)
		}
		statusFromRequest = input.AvailabilityStatus
		app.UpdateFromRequest(input)
	}

	err = applicationDB.Update(app)
	if err != nil {
		return err
	}

	setNotificationForAvailabilityStatus(c, previousStatus, app)
	setEventStreamResource(c, app)

	if statusFromRequest != nil {
		err := service.UpdateSourceFromApplicationAvailabilityStatus(app, previousStatus)
		if err != nil {
			return err
		}
	}

	// for this model we ONLY raise the update for superkey sources once the
	// worker posts back.
	if applicationDB.IsSuperkey(app.ID) {
		if app.GotSuperkeyUpdate {
			c.Set("event_override", "Application.create")
		}
	}

	return c.JSON(http.StatusOK, app.ToResponse())
}

func ApplicationDelete(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	// Check if the application exists before proceeding.
	applicationExists, err := applicationDB.Exists(id)
	if err != nil {
		return err
	}

	if !applicationExists {
		return util.NewErrNotFound("application")
	}

	c.Logger().Infof("Deleting Application Id %v", id)

	// Cascade delete the application.
	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	err = service.DeleteCascade(applicationDB.Tenant(), "Application", id, forwardableHeaders)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func ApplicationListAuthentications(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	appID, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auths, count, err := authDao.ListForApplication(appID, 100, 0, nil)
	if err != nil {
		return err
	}

	tenantId := authDao.Tenant()
	out := make([]interface{}, count)
	for i := 0; i < int(count); i++ {
		// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
		err := marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, &auths[i])
		if err != nil {
			return err
		}

		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), 100, 0))
}

func SourceListApplications(c echo.Context) error {
	applicationDB, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	var (
		applications []m.Application
		count        int64
	)

	id, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	applications, count, err = applicationDB.SubCollectionList(m.Source{ID: id}, limit, offset, filters)
	if err != nil {
		return err
	}

	c.Logger().Infof("tenant: %v", *applicationDB.Tenant())

	out := make([]interface{}, len(applications))
	for i := 0; i < len(applications); i++ {
		out[i] = applications[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

// ApplicationPause pauses a given application by setting its "paused_at" column to "now()".
func ApplicationPause(c echo.Context) error {
	applicationId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	applicationDao, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	err = applicationDao.Pause(applicationId)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	application, err := applicationDao.GetById(&applicationId)
	if err != nil {
		return err
	}

	// Get the Kafka headers we will need to be forwarding.
	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	// Raise the pause event for the application.
	err = service.RaiseEvent("Application.pause", application, forwardableHeaders)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ApplicationUnpause resumes a given application by setting its "paused_at" column to "NULL".
func ApplicationUnpause(c echo.Context) error {
	applicationId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	applicationDao, err := getApplicationDao(c)
	if err != nil {
		return err
	}

	err = applicationDao.Unpause(applicationId)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	application, err := applicationDao.GetById(&applicationId)
	if err != nil {
		return err
	}

	// Get the Kafka headers we will need to be forwarding.
	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	// Raise the unpause event for the application.
	err = service.RaiseEvent("Application.unpause", application, forwardableHeaders)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusNoContent, nil)
}
