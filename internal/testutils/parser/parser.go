package parser

import "flag"

// RunningIntegrationTests indicates whether the tests will be run as "unit" or "integration" tests.
var RunningIntegrationTests = false

// Flags holds the flags to control what the tests should be doing.
type Flags struct {
	CreateDb    bool
	Integration bool
}

// ParseFlags parses the flags for the "go test" command. The "-createdb" indicates that a test database should be
// created, whilst the "-integration" flag indicates that integration tests should be run along with the unit tests.
func ParseFlags() Flags {
	createDb := flag.Bool("createdb", false, "create the test database")
	integration := flag.Bool("integration", false, "run unit or integration tests")

	flag.Parse()

	// Set the "integration tests" variable
	RunningIntegrationTests = *integration

	return Flags{
		CreateDb:    *createDb,
		Integration: *integration,
	}
}
