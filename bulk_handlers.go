package main

import (
	"fmt"
	"net/http"

	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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

	xrhid, ok := c.Get("x-rh-identity").(string)
	if !ok {
		c.Logger().Warnf("bad xrhid %v", c.Get("x-rh-identity"))
	}
	id, ok := c.Get("identity").(identity.XRHID)
	if !ok {
		c.Logger().Warnf("failed to pull identity from request")
		return fmt.Errorf("failed to pull identity from request")
	}

	// TODO: Pull the identity from the context after the org_id changes are merged.
	output, err := service.BulkAssembly(req, &m.Tenant{Id: tenantID, ExternalTenant: id.Identity.AccountNumber})
	if err != nil {
		return err
	}

	service.SendBulkMessages(output, service.ForwadableHeaders(c), xrhid)

	return c.JSON(http.StatusCreated, output.ToResponse())
}
