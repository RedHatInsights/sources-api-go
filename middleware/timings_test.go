package middleware

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/labstack/echo/v4"
)

var sleepyFunc echo.HandlerFunc = func(c echo.Context) error {
	time.Sleep(50 * time.Millisecond)
	return c.NoContent(http.StatusOK)
}

func TestTiming(t *testing.T) {
	c, _ := request.EmptyTestContext()

	err := Timing(sleepyFunc)(c)
	if err != nil {
		t.Error(err)
	}

	duration := c.Response().Header().Get("X-Request-Duration-Ms")
	if duration == "" {
		t.Errorf("Failed to pull Duration from request")
	}

	timing, err := strconv.ParseInt(duration, 10, 64)
	if err != nil {
		t.Error(err)
	}

	if timing < 50 {
		t.Errorf("Timing was not more than 50ms as expected: %vms", timing)
	}
}
