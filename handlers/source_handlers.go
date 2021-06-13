package handlers

import (
	"net/http"
	"strconv"

	"github.com/lindgrenj6/sources-api-go/db"
	"github.com/lindgrenj6/sources-api-go/model"

	"github.com/labstack/echo/v4"
)

func SourceList(c echo.Context) error {
	var sources []model.Source
	result := db.DB.Find(&sources)
	c.Logger().Infof("count: %v, error %v", result.RowsAffected, result.Error)

	out := make([]model.SourceResponse, len(sources))
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

	var s model.Source
	db.DB.First(&s, id)

	return c.JSON(http.StatusOK, s.ToResponse())
}

func SourceCreate(c echo.Context) error {
	input := &model.SourceCreateRequest{}
	if err := c.Bind(input); err != nil {
		return err
	}

	source := &model.Source{
		Name:                input.Name,
		Uid:                 input.Uid,
		Version:             input.Version,
		Imported:            input.Imported,
		SourceRef:           input.SourceRef,
		AppCreationWorkflow: input.AppCreationWorkflow,
		AvailabilityStatus: model.AvailabilityStatus{
			AvailabilityStatus:      input.AvailabilityStatus,
			AvailabilityStatusError: input.AvailabilityStatusError,
		},
		SourceTypeId: input.SourceTypeId,
	}

	db.DB.Create(source)

	return c.JSON(http.StatusOK, source.ToResponse())
}

func SourceEdit(c echo.Context) error {
	return c.String(http.StatusOK, "hello")
}

func SourceDelete(c echo.Context) (err error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return
	}
	c.Logger().Infof("Deleting Source ID %v", id)
	result := db.DB.Delete(&model.Source{Id: int64(id)})

	if result.RowsAffected != 0 {
		return c.NoContent(http.StatusNoContent)
	} else {
		return c.NoContent(http.StatusNotFound)
	}

}
