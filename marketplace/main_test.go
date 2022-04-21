package marketplace

import (
	"log"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/util"
	miniredisV2 "github.com/alicebob/miniredis/v2"
)

var miniredis *miniredisV2.Miniredis

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	_ = parser.ParseFlags()

	// Start Miniredis
	miniredis = miniredisV2.NewMiniRedis()
	err := miniredis.StartAddr("localhost:45884")
	if err != nil {
		log.Fatalf("Could not initialize Minidredis: %s", err)
	}

	// Initialize the logger to avoid nil dereference errors.
	logger.InitLogger(config.Get())

	// Initialize the encryption key to the following: aaaaaaaaaaaaaaaa
	err = os.Setenv("ENCRYPTION_KEY", "YWFhYWFhYWFhYWFhYWFhYQ")
	if err != nil {
		log.Fatalf(`error setting the "ENCRYPTION_KEY" environment variable: %s`, err)
	}
	util.InitializeEncryption()

	result := t.Run()

	// Close Miniredis
	miniredis.Close()

	os.Exit(result)
}
