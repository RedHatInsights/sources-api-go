package main

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/marketplace"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

var getAuthenticationDao func(c echo.Context) (dao.AuthenticationDao, error)

func getAuthenticationDaoWithTenant(c echo.Context) (dao.AuthenticationDao, error) {
	tenantId, err := getTenantFromEchoContext(c)

	if err != nil {
		return nil, err
	}

	return dao.GetAuthenticationDao(&tenantId), nil
}

func AuthenticationList(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
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

	authentications, count, err := authDao.List(limit, offset, filters)
	if err != nil {
		return err
	}

	tenantId := authDao.Tenant()
	out := make([]interface{}, 0, len(authentications))
	for _, auth := range authentications {
		// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
		err := marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, &auth)
		if err != nil {
			return err
		}

		out = append(out, *auth.ToResponse())
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func AuthenticationGet(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	auth, err := authDao.GetById(c.Param("uid"))
	if err != nil {
		return err
	}

	// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
	tenantId := authDao.Tenant()
	err = marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, auth)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, auth.ToResponse())
}

func AuthenticationCreate(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	createRequest := m.AuthenticationCreateRequest{}
	err = c.Bind(&createRequest)
	if err != nil {
		return err
	}

	err = service.ValidateAuthenticationCreationRequest(&createRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	var extra map[string]interface{}
	var extraDb datatypes.JSON

	if config.IsVaultOn() {
		extra = createRequest.Extra
	} else {
		extraDb, err = json.Marshal(createRequest.Extra)

		if err != nil {
			return util.NewErrBadRequest(`invalid JSON given in "extra" field`)
		}
	}

	auth := &m.Authentication{
		Name:         createRequest.Name,
		AuthType:     createRequest.AuthType,
		Username:     createRequest.Username,
		Password:     createRequest.Password,
		Extra:        extra,
		ExtraDb:      extraDb,
		ResourceType: createRequest.ResourceType,
		ResourceID:   createRequest.ResourceID,
	}
	err = authDao.Create(auth)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
	tenantId := authDao.Tenant()
	err = marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, auth)
	if err != nil {
		return err
	}

	accountNumber, err := getAccountNumberFromEchoContext(c)
	if err != nil {
		c.Logger().Warn(err)
	}

	auth.Tenant = m.Tenant{Id: auth.TenantID, ExternalTenant: accountNumber}
	setEventStreamResource(c, auth)
	return c.JSON(http.StatusCreated, auth.ToResponse())
}

func AuthenticationEdit(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	updateRequest := &m.AuthenticationEditRequest{}
	err = c.Bind(updateRequest)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	auth, err := authDao.GetById(c.Param("uid"))
	if err != nil {
		return err
	}

	previousStatus := ""
	if auth.AvailabilityStatus != nil {
		previousStatus = *auth.AvailabilityStatus
	}
	err = auth.UpdateFromRequest(updateRequest)
	if err != nil {
		return util.NewErrBadRequest(`invalid JSON given in "extra" field`)
	}

	err = authDao.Update(auth)
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	sourceDao := dao.GetSourceDao(authDao.Tenant())
	source, err := sourceDao.GetById(&auth.SourceID)
	if err != nil {
		return err
	}
	auth.Source = *source

	// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
	tenantId := authDao.Tenant()
	err = marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, auth)
	if err != nil {
		return err
	}

	setNotificationForAvailabilityStatus(c, previousStatus, auth)
	setEventStreamResource(c, auth)
	return c.JSON(http.StatusOK, auth.ToResponse())
}

func AuthenticationDelete(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	auth, err := authDao.Delete(c.Param("uid"))
	if err != nil {
		return err
	}

	// Set the marketplace token —if the auth is of the marketplace type— for the authentication.
	tenantId := authDao.Tenant()
	err = marketplace.SetMarketplaceTokenAuthExtraField(*tenantId, auth)
	if err != nil {
		return err
	}

	setEventStreamResource(c, auth)
	return c.NoContent(http.StatusNoContent)
}
