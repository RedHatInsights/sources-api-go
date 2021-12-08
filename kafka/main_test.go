package kafka

import (
	"flag"
	"os"
	"testing"
)

func TestMain(t *testing.M) {
	// we need this to parse arguments otherwise there are not recognized which lead to error
	flag.Bool("createdb", false, "create the test database")
	flag.Bool("integration", false, "run unit or integration tests")
	flag.Parse()

	os.Exit(t.Run())
}
