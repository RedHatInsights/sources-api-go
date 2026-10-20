package util

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	l "github.com/RedHatInsights/sources-api-go/logger"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = parser.ParseFlags()

	// Initialize the logger to avoid nil pointer dereferences.
	l.InitLogger(config.Get())

	os.Exit(t.Run())
}
