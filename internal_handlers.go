package main

import (
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

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
	sourcesDB := &dao.SourceDaoImpl{}
	sources, count, err := sourcesDB.ListInternal(limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	out := make([]interface{}, len(sources))
	for i := 0; i < len(sources); i++ {
		out[i] = sources[i].ToInternalResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}
