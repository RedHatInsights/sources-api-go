package handlers

import (
	"net/http"
	"strconv"

	"github.com/lindgrenj6/sources-api-go/dao"

	m "github.com/lindgrenj6/sources-api-go/model"

	"github.com/labstack/echo/v4"
)

func SourceList(c echo.Context) error {
	sources, count, err := dao.SourceList(100, 0)
	if err != nil {
		return err
	}
	c.Logger().Infof("count: %v, error %v", count, err)

	out := make([]m.SourceResponse, len(sources))
	for i, s := range sources {
		out[i] = *s.ToResponse()
	}

	return c.JSON(http.StatusOK, out)
}

func SourceGet(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}

	c.Logger().Infof("Getting Source ID %v", id)

	s, err := dao.SourceGet(int64(id))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceCreate(c echo.Context) error {
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
	}

	id, err := dao.SourceCreate(source)
	if err != nil {
		return err
	}

	s, err := dao.SourceGet(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceEdit(c echo.Context) error {
	input := &m.SourceEditRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}

	s, err := dao.SourceGet(int64(id))
	if err != nil {
		return err
	}

	s.UpdateFromRequest(input)
	err = dao.SourceUpdate(s)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceDelete(c echo.Context) (err error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}

	c.Logger().Infof("Deleting Source ID %v", id)

	err = dao.SourceDelete(int64(id))
	if err != nil {
		return c.NoContent(http.StatusNoContent)
	} else {
		return c.NoContent(http.StatusNotFound)
	}
}
