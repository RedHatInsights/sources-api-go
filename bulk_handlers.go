package main

import (
	"fmt"
	"net/http"

	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
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

	tenantID, ok := c.Get(h.TENANTID).(int64)
	if !ok {
		return fmt.Errorf("failed to pull tenant from request")
	}

	xrhid, ok := c.Get(h.XRHID).(string)
	if !ok {
		c.Logger().Warnf("bad xrhid %v", c.Get(h.XRHID))
	}
	id, ok := c.Get(h.PARSED_IDENTITY).(*identity.XRHID)
	if !ok {
		c.Logger().Warnf("failed to pull identity from request")
		return fmt.Errorf("failed to pull identity from request")
	}

	user := &m.User{TenantID: tenantID}
	userID, ok := c.Get(h.USERID).(int64)

	if ok {
		user.UserID = id.Identity.User.UserID
		user.Id = userID
	}

	// TODO: Pull the identity from the context after the org_id changes are merged.
	output, err := service.BulkAssembly(req, &m.Tenant{Id: tenantID, ExternalTenant: id.Identity.AccountNumber}, user)
	if err != nil {
		return err
	}

	forwardableHeaders, err := service.ForwadableHeaders(c)
	if err != nil {
		return err
	}

	service.SendBulkMessages(output, forwardableHeaders, xrhid)

	return c.JSON(http.StatusCreated, output.ToResponse())
}
