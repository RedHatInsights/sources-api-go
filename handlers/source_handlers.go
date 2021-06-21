package handlers

import (
	"fmt"
	"github.com/lindgrenj6/sources-api-go/middleware"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/lindgrenj6/sources-api-go/dao"
	m "github.com/lindgrenj6/sources-api-go/model"
	"github.com/lindgrenj6/sources-api-go/util"
)

func getSourceDao(c echo.Context) (dao.SourceDao, error) {
	var tenantID int64
	var ok bool
	if tenantID, ok = c.Get("tenantID").(int64); !ok {
		return nil, c.JSON(http.StatusForbidden, util.ErrorDoc("no tenant id", "401"))
	}

	return &dao.SourceDaoImpl{TenantID: &tenantID}, nil
}

func SourceList(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	filters := c.Get("filters").([]middleware.Filter)
	fmt.Printf("%#v\n", filters)
	sources, err := sourcesDB.List(
		c.Get("limit").(int),
		c.Get("offset").(int),
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("Bad Request", "400"))
	}
	c.Logger().Infof("tenant: %v", *sourcesDB.Tenant())

	out := make([]interface{}, len(sources))
	for i, s := range sources {
		out[i] = *s.ToResponse()
	}

	count, err := sourcesDB.Count()
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc("Error getting count", "400"))
	}
	return c.JSON(http.StatusOK, util.CollectionResponse(out, count, 100, 0))
}

func SourceGet(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	c.Logger().Infof("Getting Source ID %v", id)

	s, err := sourcesDB.GetById(&id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceCreate(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	input := &m.SourceCreateRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	source := &m.Source{
		Name:                input.Name,
		Uid:                 input.Uid,
		Version:             input.Version,
		Imported:            input.Imported,
		SourceRef:           input.SourceRef,
		AppCreationWorkflow: input.AppCreationWorkflow,
		AvailabilityStatus: m.AvailabilityStatus{
			AvailabilityStatus: input.AvailabilityStatus,
		},
		SourceTypeId: input.SourceTypeId,
		Tenancy:      m.Tenancy{TenantId: sourcesDB.Tenant()},
	}

	id, err := sourcesDB.Create(source)
	if err != nil {
		return err
	}

	s, err := sourcesDB.GetById(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceEdit(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	input := &m.SourceEditRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	s, err := sourcesDB.GetById(&id)
	if err != nil {
		return err
	}

	s.UpdateFromRequest(input)
	err = sourcesDB.Update(s)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceDelete(c echo.Context) (err error) {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return err
	}

	c.Logger().Infof("Deleting Source ID %v", id)

	err = sourcesDB.Delete(&id)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}
