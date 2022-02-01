package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// function that defines how we get the dao - default implementation below.
var getSourceDao func(c echo.Context) (dao.SourceDao, error)

func getSourceDaoWithTenant(c echo.Context) (dao.SourceDao, error) {
	var tenantID int64
	var ok bool

	tenantVal := c.Get("tenantID")
	if tenantID, ok = tenantVal.(int64); !ok {
		return nil, fmt.Errorf("failed to pull tenant from request")
	}

	return &dao.SourceDaoImpl{TenantID: &tenantID}, nil
}

func SourceList(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
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
		sources []m.Source
		count   int64
	)

	// When listing sources via cert-auth we want to lock them down to only the
	// satellite source type.
	if c.Get("cert-auth") != nil {
		satelliteId := strconv.Itoa(int(dao.Static.GetSourceTypeId("satellite")))
		filters = append(filters, util.Filter{Name: "source_type_id", Value: []string{satelliteId}})
	}

	sources, count, err = sourcesDB.List(limit, offset, filters)

	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}
	c.Logger().Infof("tenant: %v", *sourcesDB.Tenant())

	out := make([]interface{}, len(sources))
	for i, s := range sources {
		out[i] = *s.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func SourceGet(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
	if err != nil {
		return err
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	c.Logger().Infof("Getting Source Id %v", id)

	s, err := sourcesDB.GetById(&id)

	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
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

	err = service.ValidateSourceCreationRequest(sourcesDB, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(fmt.Sprintf("Validation failed: %s", err.Error()), "400"))
	}

	source := &m.Source{
		Name:                *input.Name,
		Uid:                 input.Uid,
		Version:             input.Version,
		Imported:            input.Imported,
		SourceRef:           input.SourceRef,
		AppCreationWorkflow: input.AppCreationWorkflow,
		AvailabilityStatus: m.AvailabilityStatus{
			AvailabilityStatus: input.AvailabilityStatus,
		},
		SourceTypeID: *input.SourceTypeID,
	}

	err = sourcesDB.Create(source)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, source.ToResponse())
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

	c.Logger().Infof("Deleting Source Id %v", id)

	err = sourcesDB.Delete(&id)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

func SourceListAuthentications(c echo.Context) error {
	authDao, err := getAuthenticationDao(c)
	if err != nil {
		return err
	}

	sourceID, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	auths, count, err := authDao.ListForSource(sourceID, 100, 0, nil)
	if err != nil {
		return c.JSON(http.StatusNotFound, util.ErrorDoc(err.Error(), "404"))
	}

	out := make([]interface{}, count)
	for i := 0; i < int(count); i++ {
		out[i] = auths[i].ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), 100, 0))
}

func SourceTypeListSource(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
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
		sources []m.Source
		count   int64
	)

	id, err := strconv.ParseInt(c.Param("source_type_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	sources, count, err = sourcesDB.SubCollectionList(m.SourceType{Id: id}, limit, offset, filters)

	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	c.Logger().Infof("tenant: %v", *sourcesDB.Tenant())

	out := make([]interface{}, len(sources))
	for i, s := range sources {
		out[i] = *s.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func ApplicationTypeListSource(c echo.Context) error {
	sourcesDB, err := getSourceDao(c)
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
		sources []m.Source
		count   int64
	)

	id, err := strconv.ParseInt(c.Param("application_type_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	sources, count, err = sourcesDB.SubCollectionList(m.ApplicationType{Id: id}, limit, offset, filters)

	if err != nil {
		if errors.Is(err, util.ErrNotFoundEmpty) {
			return err
		}
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	c.Logger().Infof("tenant: %v", *sourcesDB.Tenant())

	out := make([]interface{}, len(sources))
	for i, s := range sources {
		out[i] = *s.ToResponse()
	}

	return c.JSON(http.StatusOK, util.CollectionResponse(out, c.Request(), int(count), limit, offset))
}

func SourceCheckAvailability(c echo.Context) error {
	sourceDao, err := getSourceDao(c)
	if err != nil {
		return err
	}

	sourceID, err := strconv.ParseInt(c.Param("source_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, util.ErrorDoc(err.Error(), "400"))
	}

	src, err := sourceDao.GetByIdWithPreload(&sourceID,
		"SourceType",
		"Applications",
		"Applications.ApplicationType",
		"Endpoints",
		"Tenant",
	)
	if err != nil {
		return c.JSON(http.StatusNotFound, util.ErrorDoc(err.Error(), "404"))
	}

	// do it async!
	go func() { service.RequestAvailabilityCheck(src) }()

	return c.JSON(http.StatusAccepted, map[string]interface{}{})
}

func InternalSourceList(c echo.Context) error {
	filters, err := getFilters(c)
	if err != nil {
		return err
	}

	limit, offset, err := getLimitAndOffset(c)
	if err != nil {
		return err
	}

	// When listing sources via cert-auth we want to lock them down to only the
	// satellite source type.
	if c.Get("cert-auth") != nil {
		satelliteId := strconv.Itoa(int(dao.Static.GetSourceTypeId("satellite")))
		filters = append(filters, util.Filter{Name: "source_type_id", Value: []string{satelliteId}})
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
