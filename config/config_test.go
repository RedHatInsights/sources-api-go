package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

func TestMain(t *testing.M) {
	// We need to parse the flags as otherwise we face the "flag provided but not defined: -integration" error.
	_ = parser.ParseFlags()

	t.Run()
}

// TestFindDependentApplication tests that the function under test reports an error when the specified application is
// not found in Clowder's configuration, and that the endpoint is returned when it is.
func TestFindDependentApplication(t *testing.T) {
	// Declare the test endpoints as if we had read them from Clowder.
	testEndpoints := []clowder.DependencyEndpoint{
		{
			App: "authorization",
		},
		{
			App: "cache",
		},
		{
			App: "message-bus",
		},
	}

	// Define a slice of test cases.
	testCases := []struct {
		ApplicationName string
		ExpectError     bool
	}{
		{
			ApplicationName: "made-up-application",
			ExpectError:     true,
		},
		{
			ApplicationName: "auth",
			ExpectError:     true,
		},
		{
			ApplicationName: "AuTh",
			ExpectError:     true,
		},
		{
			ApplicationName: "authorization",
			ExpectError:     false,
		},
		{
			ApplicationName: "cache",
			ExpectError:     false,
		},
		{
			ApplicationName: "message-bus",
			ExpectError:     false,
		},
		{
			ApplicationName: "AUTHORIZATION",
			ExpectError:     false,
		},
		{
			ApplicationName: "authoriZation",
			ExpectError:     false,
		},
		{
			ApplicationName: "AuThOrIzaTiOn",
			ExpectError:     false,
		},
	}

	for _, tc := range testCases {
		endpoint, err := findDependentApplication(tc.ApplicationName, testEndpoints)

		if tc.ExpectError {
			if err == nil {
				t.Errorf(`the function under test should have returned an error, none was returned for application "%s"`, tc.ApplicationName)
				continue
			}

			wantErrorMsg := fmt.Sprintf(`unable to find application "%s" in the endpoints section of the cdappconfig.json file`, tc.ApplicationName)
			if err.Error() != wantErrorMsg {
				t.Errorf(`the function under test returned an unexpected error. Want "%s", got "%s"`, wantErrorMsg, err.Error())
				continue
			}
		} else {
			if err != nil {
				t.Errorf(`the function under test should have not returned an error when testing it with the application "%s", but the following one was returned: %s`, tc.ApplicationName, err)
				continue
			}

			if !strings.EqualFold(endpoint.App, tc.ApplicationName) {
				t.Errorf(`unexpected application was found by the function under test. Want "%s", got "%s"`, tc.ApplicationName, endpoint.App)
				continue
			}
		}
	}
}
