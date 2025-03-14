package main

import (
	"fmt"
	"github.com/RedHatInsights/sources-api-go/middleware/headers"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// InternalAuthenticationGet fetches one authentication and returns it with the password exposed. Internal use only.
func InternalAuthenticationGet(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	auth, err := authDao.GetById(c.Param("uuid"))
	if err != nil {
		return err
	}

	exposeEncryptedAttribute := c.QueryParam("expose_encrypted_attribute[]")
	if exposeEncryptedAttribute == "password" {
		return c.JSON(http.StatusOK, auth.ToInternalResponse())
	}

	return c.JSON(http.StatusOK, auth.ToResponse())
}

// InternalSourceList lists all the sources in a compact format —since the client that will use it,
// "sources-monitor-go" only requires a small set of fields—.
func InternalSourceList(c echo.Context) error {
	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	// Skip Sources which do not have associated applications or RHC Connections. Useful for the Sources Monitor since
	// it prevents returning Sources that do not need to have availability checks performed for them. More information
	// here: https://issues.redhat.com/browse/RHCLOUD-38735.
	var skipEmptySources = false
	if skip := c.Request().Header.Get(headers.SkipEmptySources); skip != "" {
		skipEmptySources = skip == "true"
	}

	// The DAO doesn't need a tenant set, since the queries won't be filtered by that tenant
	sourcesDB := dao.GetSourceDao(nil)
	sources, count, err := sourcesDB.ListInternal(limit, offset, filters, skipEmptySources)

	if err != nil {
		return err
	}

	out := make([]interface{}, len(sources))
	for i := 0; i < len(sources); i++ {
		out[i] = sources[i].ToInternalResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

// GetUntranslatedTenants returns all the tenants from the database that have just an "external_tenant" set up.
func GetUntranslatedTenants(c echo.Context) error {
	tenantsDao := dao.GetTenantDao()

	tenants, err := tenantsDao.GetUntranslatedTenants()
	if err != nil {
		return fmt.Errorf("unable to fetch the untranslated tenants: %w", err)
	}

	out := make([]interface{}, len(tenants))
	for i := 0; i < len(tenants); i++ {
		out[i] = tenants[i]
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), len(tenants), 0, 0))
}

// TranslateTenants attempts to translate the
func TranslateTenants(c echo.Context) error {
	tenantsDao := dao.GetTenantDao()

	translatableTenants, translatedTenants, untranslatedTenants, translationResults, err := tenantsDao.TranslateTenants()
	if err != nil {
		return fmt.Errorf("unable to translate the EBS account numbers to orgIds: %w", err)
	}

	response := map[string]interface{}{
		"total_translatable_tenants_count": translatableTenants,
		"translation_results":              translationResults,
		"translated_tenants":               translatedTenants,
		"untranslated_tenants":             untranslatedTenants,
	}

	return c.JSON(http.StatusOK, response)
}

func InternalSecretGet(c echo.Context) error {
	secretDao, err := getSecretDao(c)
	if err != nil {
		return err
	}

	paramID, err := util.InterfaceToInt64(c.Param("id"))
	if err != nil {
		return util.NewErrBadRequest(err)
	}

	secret, err := secretDao.GetById(&paramID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, secret.ToInternalSecretResponse())
}
