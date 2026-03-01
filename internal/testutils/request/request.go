package request

import (
	"io"
	"net/http"
	"net/http/httptest"

	echoUtils "github.com/RedHatInsights/sources-api-go/util/echo"
	"github.com/labstack/echo/v5"
)

var echoInstance = echo.New()

// CreateTestContext sets up a new echo context with the parameters given, and returns the context itself and the
// response recorder.
func CreateTestContext(method string, path string, body io.Reader, context map[string]interface{}) (echo.Context, *httptest.ResponseRecorder) {
	echoInstance.Binder = &echoUtils.NoUnknownFieldsBinder{}
	request := httptest.NewRequest(method, path, body)
	recorder := httptest.NewRecorder()
	echoContext := echoInstance.NewContext(request, recorder)

	// Passing the headers in the context instead of in a new variable is very unorthodox, but this avoids having to
	// refactor all the functions that depend on this, which would clutter any PR that needs to modify it.
	if headersMap, ok := context["headers"]; ok {
		if headers, typeOk := headersMap.(map[string]string); typeOk {
			for key, value := range headers {
				request.Header.Add(key, value)
			}
		}

		delete(context, "headers")
	}

	for k, v := range context {
		echoContext.Set(k, v)
	}

	return echoContext, recorder
}

// EmptyTestContext returns an empty http context - for when we don't need much
// other than the recorder + context
func EmptyTestContext() (echo.Context, *httptest.ResponseRecorder) {
	return CreateTestContext(http.MethodGet, "/", nil, nil)
}
