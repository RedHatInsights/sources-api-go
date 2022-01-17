package dao

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = parser.ParseFlags()

	os.Exit(t.Run())
}
