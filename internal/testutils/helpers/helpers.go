package helpers

import (
	"fmt"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/util"
)

func GetSourcesCountWithAppType(appTypeId int64) int {
	var sourcesCount int
	var sourceIdList []int64

	for _, app := range fixtures.TestApplicationData {
		if app.ApplicationTypeID == appTypeId {
			sourceIdList = append(sourceIdList, app.SourceID)
		}
	}
	for _, sourceID := range sourceIdList {
		for _, src := range fixtures.TestSourceData {
			if src.ID == sourceID {
				sourcesCount++
			}
		}
	}
	return sourcesCount
}

func GetAppTypeCountWithSourceId(sourceId int64) int {
	var appTypeList []int64

	for _, app := range fixtures.TestApplicationData {
		if app.SourceID == sourceId {
			appTypeList = append(appTypeList, app.ApplicationTypeID)
		}
	}

	return len(appTypeList)
}

// SkipIfNotRunningIntegrationTests is a helper function which skips a test if the integration tests don't want to be
// run.
func SkipIfNotRunningIntegrationTests(t *testing.T) {
	if !parser.RunningIntegrationTests {
		t.Skip("Skipping integration test")
	}
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
