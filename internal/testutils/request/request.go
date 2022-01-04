package request

import (
	"io"
	"net/http/httptest"

	"github.com/labstack/echo/v4"
)

var echoInstance = echo.New()

// CreateTestContext sets up a new echo context with the parameters given, and returns the context itself and the
// response recorder.
func CreateTestContext(method string, path string, body io.Reader, context map[string]interface{}) (echo.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, path, body)
	recorder := httptest.NewRecorder()
	echoContext := echoInstance.NewContext(request, recorder)

	for k, v := range context {
		echoContext.Set(k, v)
	}

	return echoContext, recorder
}
