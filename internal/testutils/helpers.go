package testutils

import (
	"fmt"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
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

func GetSourcesWithAppType(appTypeId int64) []model.Source {
	var sourceIds = make(map[int64]struct{})

	// Find applications with given application type and get
	// list of unique source IDs
	for _, app := range fixtures.TestApplicationData {
		if app.ApplicationTypeID == appTypeId {
			_, ok := sourceIds[app.SourceID]
			if !ok {
				sourceIds[app.SourceID] = struct{}{}
			}
		}
	}

	// Find sources for source IDs
	var sources []model.Source

	for id := range sourceIds {
		for _, src := range fixtures.TestSourceData {
			if id == src.ID {
				sources = append(sources, src)
				break
			}
		}
	}

	return sources
}

func AssertLinks(t *testing.T, path string, links util.Links, limit int, offset int) {
	expectedFirstLink := fmt.Sprintf("%s?limit=%d&offset=%d", path, limit, offset)

	expectedLastLink := fmt.Sprintf("%s?limit=%d&offset=%d", path, limit, limit+offset)
	if links.First != expectedFirstLink {
		t.Error("first link is not correct for " + path)
	}

	if links.Last != expectedLastLink {
		t.Error("last link is not correct for " + path)
	}
}

func IdentityHeaderForUser(testUserId string) *identity.XRHID {
	accountNumber := fixtures.TestTenantData[0].ExternalTenant
	orgID := fixtures.TestTenantData[0].OrgID

	return &identity.XRHID{Identity: identity.Identity{OrgID: orgID, AccountNumber: accountNumber, User: &identity.User{UserID: testUserId}}}
}

func SingleResourceBulkCreateRequest(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType string) *model.BulkCreateRequest {
	return SingleResourceBulkCreateRequestWithStatus(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType, model.Available)
}

// SingleResourceBulkCreateRequestWithStatus similar to the other function, but
// it allows specifying the availability status for the resources.
func SingleResourceBulkCreateRequestWithStatus(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType, availablityStatus string) *model.BulkCreateRequest {
	// Set up the source.
	sourceCreateRequest := model.SourceCreateRequest{Name: &nameSource, AvailabilityStatus: availablityStatus}
	bulkCreateSource := model.BulkCreateSource{SourceCreateRequest: sourceCreateRequest, SourceTypeName: sourceTypeName}

	// Set up the application.
	bulkCreateApplication := model.BulkCreateApplication{SourceName: nameSource, ApplicationTypeName: applicationTypeName}

	// Set up the authentication.
	authenticationCreateRequest := model.AuthenticationCreateRequest{ResourceType: authenticationResourceType}
	bulkCreateAuthentication := model.BulkCreateAuthentication{AuthenticationCreateRequest: authenticationCreateRequest, ResourceName: applicationTypeName}

	// set up the endpoint.
	endpointCreateRequest := model.EndpointCreateRequest{AvailabilityStatus: availablityStatus}
	bulkCreateEndpoints := model.BulkCreateEndpoint{EndpointCreateRequest: endpointCreateRequest, SourceName: nameSource}

	return &model.BulkCreateRequest{Sources: []model.BulkCreateSource{bulkCreateSource},
		Applications:    []model.BulkCreateApplication{bulkCreateApplication},
		Authentications: []model.BulkCreateAuthentication{bulkCreateAuthentication},
		Endpoints:       []model.BulkCreateEndpoint{bulkCreateEndpoints}}
}
