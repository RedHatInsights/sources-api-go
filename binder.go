package main

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

// struct containing our custom "bind" logic
type NoUnknownFieldsBinder struct{}

/*
	The single "Bind" method implements the echo.Binder interface.

	For our custom binder we want to not allow any fields that aren't declared
	on the `*CreateRequest` structs. This is easily achieved by switching on the
	`DisallowUnknownFields` Decoder setting for unmarshaling json.
*/
func (binder *NoUnknownFieldsBinder) Bind(i interface{}, c echo.Context) error {
	// if there is no body on the request - return early
	if c.Request().Body == http.NoBody || c.Request().Body == nil {
		return util.NewErrBadRequest("no body")
	}
	defer c.Request().Body.Close()

	// create a new decoder for the request body
	dec := json.NewDecoder(c.Request().Body)
	// return an error if any non-declared fields are included in the request
	dec.DisallowUnknownFields()

	err := dec.Decode(i)
	if err != nil {
		c.Logger().Warnf("Failed to decode request: %v", err)
		return util.NewErrBadRequest(err)
	}

	return nil
}
