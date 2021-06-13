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

	return c.JSON(http.StatusOK, sources)
}

func SourceGet(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}
	c.Logger().Infof("Getting Source ID %v", id)
	var s model.Source
	db.DB.First(s, id)

	return c.JSON(http.StatusOK, s)
}

func SourceCreate(c echo.Context) error {
	var src model.Source
	if err := c.Bind(src); err != nil {
		return err
	}

	db.DB.Create(src)

	return c.JSON(http.StatusOK, src)
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
