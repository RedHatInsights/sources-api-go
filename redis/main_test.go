package redis

import (
	"log"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
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

	result := t.Run()

	// Close Miniredis
	miniredis.Close()

	os.Exit(result)
}
