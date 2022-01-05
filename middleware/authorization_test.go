package middleware

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v4"
)

var checkPSKOrElse204 = PermissionCheck(echo.HandlerFunc(func(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}))

func TestPSKMatches(t *testing.T) {
	PSKS = []string{"1234"}
	if pskMatches("1234") != true {
		t.Errorf("psk didn't match when it should have")
	}

	if pskMatches("12345") == true {
		t.Errorf("psk matched when it should not have")
	}
}

func TestGoodPSK(t *testing.T) {
	PSKS = []string{"1234"}
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{"psk": "1234"},
	)

	err := checkPSKOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 204 {
		t.Errorf("%v was returned instead of %v", rec.Code, 200)
	}
}

func TestBadPSK(t *testing.T) {
	PSKS = []string{"abcdef"}
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{"psk": "1234"},
	)

	err := checkPSKOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 401 {
		t.Errorf("%v was returned instead of %v", rec.Code, 401)
	}
}

func TestNoPSK(t *testing.T) {
	PSKS = []string{"abcdef"}
	c, rec := request.CreateTestContext(
		"POST",
		"/",
		nil,
		map[string]interface{}{},
	)

	err := checkPSKOrElse204(c)
	if err != nil {
		t.Errorf("caught an error when there should not have been one")
	}

	if rec.Code != 401 {
		t.Errorf("%v was returned instead of %v", rec.Code, 401)
	}
}
