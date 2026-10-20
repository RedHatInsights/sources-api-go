package jobs

import (
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/logger"
)

func TestMain(t *testing.M) {
	logger.InitLogger(config.Get())

	flags := parser.ParseFlags()

	if flags.CreateDb {
		database.CreateTestDB()
	} else if flags.Integration {
		database.ConnectAndMigrateDB("jobs")
		database.CreateFixtures("jobs")
	}

	code := t.Run()

	if flags.Integration {
		database.DropSchema("jobs")
	}

	os.Exit(code)
}
