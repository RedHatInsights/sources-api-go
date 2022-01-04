package marketplace

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = testutils.ParseFlags()

	os.Exit(t.Run())
}
