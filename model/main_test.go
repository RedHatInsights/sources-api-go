package model

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
)

func TestMain(t *testing.M) {
	_ = parser.ParseFlags()
	os.Exit(t.Run())
}
