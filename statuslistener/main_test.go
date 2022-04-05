package statuslistener

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	l "github.com/RedHatInsights/sources-api-go/logger"
)

// runningIntegration is used to skip integration tests if we're just running unit tests.
var runningIntegration = false

func TestMain(t *testing.M) {
	l.InitLogger(config)
	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		runningIntegration = true
		database.ConnectAndMigrateDB("status_listener")
		database.CreateFixtures("status_listener")
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("status_listener")
	}

	os.Exit(code)
}
