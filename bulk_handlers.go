package main

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
)

func BulkCreate(c echo.Context) error {
	req := m.BulkCreateRequest{}
	err := c.Bind(&req)
	if err != nil {
		return err
	}

	tenantID, ok := c.Get("tenantID").(int64)
	if !ok {
		return fmt.Errorf("failed to pull tenant from request")
	}

	output, err := service.ParseBulkCreateRequest(req, &tenantID)
	if err != nil {
		return err
	}

	err = dao.GetSourceDao(&tenantID).Create(&output.Sources[0])
	if err != nil {
		return err
	}

	err = service.LinkUpAuthentications(output, &tenantID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, output.ToResponse())
}
