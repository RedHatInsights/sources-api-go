package echo

import (
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/labstack/echo/v4"
)

var echoInstance = echo.New()

//duplicating helper function CreateTestContext() from internal/testutils/request/request.go in order to
// effectively test and importing this to other packages to prevent import cycle
func CreateTestContext(method string, path string, body io.Reader, context map[string]interface{}) (echo.Context, *httptest.ResponseRecorder) {
	echoInstance.Binder = &NoUnknownFieldsBinder{}
	request := httptest.NewRequest(method, path, body)
	recorder := httptest.NewRecorder()
	echoContext := echoInstance.NewContext(request, recorder)
	for k, v := range context {
		echoContext.Set(k, v)
	}

	return echoContext, recorder
}

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = parser.ParseFlags()

	// Initialize the logger to avoid nil pointer dereferences.
	l.InitLogger(config.Get())

	os.Exit(t.Run())
}
