package middleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v5"
)

func TestError(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	explosion := HandleErrors(func(echo.Context) error { return fmt.Errorf("boom!") })

	err := explosion(c)
	if err != nil {
		t.Error("caught an error when there should not have been one")
	}

	if rec.Code != 500 {
		t.Errorf("%v was returned instead of %v", rec.Code, 500)
	}

	body, _ := io.ReadAll(rec.Body)

	if !strings.Contains(string(body), "Internal Server Error:") {
		t.Errorf("malformed body: %s", body)
	}
}

func TestNoError(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/",
		nil,
		map[string]interface{}{},
	)

	goodRequest := HandleErrors(func(echo.Context) error { return nil })

	err := goodRequest(c)
	if err != nil {
		t.Error("caught an error when there should not have been one")
	}

	if rec.Code != 200 {
		t.Errorf("%v was returned instead of %v", rec.Code, 200)
	}
}
