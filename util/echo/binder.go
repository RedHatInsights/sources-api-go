package echo

import (
	"encoding/json"
	"net/http"
	"reflect"

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
	// Close the request's body after we're done with it.
	defer c.Request().Body.Close()

	// Check if request contains a body
	if c.Request().Body == http.NoBody {
		return util.NewErrBadRequest("no body")
	}

	// is the struct passed in initially empty? then it's pretty easy to tell if
	// no fields were plucked from the body.
	isInitiallyEmpty := reflect.ValueOf(i).Elem().IsZero()

	// create a new decoder for the request body
	dec := json.NewDecoder(c.Request().Body)
	// return an error if any non-declared fields are included in the request
	dec.DisallowUnknownFields()

	err := dec.Decode(i)
	if err != nil {
		c.Logger().Warnf("Failed to decode request: %v", err)
		return util.NewErrBadRequest(err)
	}

	// ...is the struct still empty after unmarshaling? if so then we had an
	// empty JSON body.
	isEmptyAfterDecoding := reflect.ValueOf(i).Elem().IsZero()

	if isInitiallyEmpty && isEmptyAfterDecoding {
		return util.NewErrBadRequest("empty body")
	}

	return nil
}
