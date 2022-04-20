package testutils

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
)

var conf = config.Get()

// SkipIfNotRunningIntegrationTests is a helper function which skips a test if the integration tests don't want to be
// run.
func SkipIfNotRunningIntegrationTests(t *testing.T) {
	if !parser.RunningIntegrationTests {
		t.Skip("Skipping integration test")
	}
}

func SkipIfNotSecretStoreDatabase(t *testing.T) {
	if conf.SecretStore == "vault" {
		t.Skip("Skipping test")
	}
}
