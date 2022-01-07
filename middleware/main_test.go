package middleware

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var (
	xrhid         string
	emptyIdentity = identity.XRHID{Identity: identity.Identity{AccountNumber: "12345"}}
)

func TestMain(t *testing.M) {
	_ = parser.ParseFlags()

	// for header parsing test
	rawId, _ := json.Marshal(emptyIdentity)
	xrhid = string(base64.StdEncoding.EncodeToString(rawId))

	logger.InitLogger(conf)
	e = echo.New()
	code := t.Run()
	os.Exit(code)
}
