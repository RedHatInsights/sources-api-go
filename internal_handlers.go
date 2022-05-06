package main

import (
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

	// The DAO doesn't need a tenant set, since the queries won't be filtered by that tenant
	sourcesDB := dao.GetSourceDao(nil)
	sources, count, err := sourcesDB.ListInternal(limit, offset, filters)

	if err != nil {
		return err
	}

	out := make([]interface{}, len(sources))
	for i := 0; i < len(sources); i++ {
		out[i] = sources[i].ToInternalResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
