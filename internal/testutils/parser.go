package testutils

import "flag"

// ParseFlags parses the flags for the "go test" command. The "-createdb" indicates that a test database should be
// created, whilst the "-integration" flag indicates that integration tests should be run along with the unit tests.
// It returns a (bool, bool) which represents the status of ("-createdb", "-integration") accordingly.
func ParseFlags() (bool, bool) {
	createDb := flag.Bool("createdb", false, "create the test database")
	integration := flag.Bool("integration", false, "run unit or integration tests")

	flag.Parse()

	return *createDb, *integration
}
