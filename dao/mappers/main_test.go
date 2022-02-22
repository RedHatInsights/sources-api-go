package mappers

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
)

func TestMain(t *testing.M) {
	_ = parser.ParseFlags()

	code := t.Run()

	os.Exit(code)
}
