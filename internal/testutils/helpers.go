package testutils

import (
	"fmt"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/redhatinsights/platform-go-middlewares/identity"
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
	return &identity.XRHID{Identity: identity.Identity{AccountNumber: accountNumber, User: identity.User{UserID: testUserId}}}
}

func SingleResourceBulkCreateRequest(nameSource, sourceTypeName, applicationTypeName, authenticationResourceType string) *model.BulkCreateRequest {
	sourceCreateRequest := model.SourceCreateRequest{Name: &nameSource}
	bulkCreateSource := model.BulkCreateSource{SourceCreateRequest: sourceCreateRequest, SourceTypeName: sourceTypeName}

	bulkCreateApplication := model.BulkCreateApplication{SourceName: nameSource, ApplicationTypeName: applicationTypeName}

	authenticationCreateRequest := model.AuthenticationCreateRequest{ResourceType: authenticationResourceType}
	bulkCreateAuthentication := model.BulkCreateAuthentication{AuthenticationCreateRequest: authenticationCreateRequest, ResourceName: applicationTypeName}

	endpointCreateRequest := model.EndpointCreateRequest{}
	bulkCreateEndpoints := model.BulkCreateEndpoint{EndpointCreateRequest: endpointCreateRequest, SourceName: nameSource}

	return &model.BulkCreateRequest{Sources: []model.BulkCreateSource{bulkCreateSource},
		Applications:    []model.BulkCreateApplication{bulkCreateApplication},
		Authentications: []model.BulkCreateAuthentication{bulkCreateAuthentication},
		Endpoints:       []model.BulkCreateEndpoint{bulkCreateEndpoints}}
}
