package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestApplicationTypeSubCollectionListOffsetAndLimit tests that SubCollectionList() in application type dao returns
//  correct count value and correct count of returned objects
func TestApplicationTypeSubCollectionListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("offset_limit")

	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	sourceId := int64(1)

	var appTypeList []int64
	for _, app := range fixtures.TestApplicationData {
		if app.SourceID == sourceId {
			appTypeList = append(appTypeList, app.ApplicationTypeID)
		}
	}
	wantCount := int64(len(appTypeList))

	for _, d := range fixtures.TestDataOffsetLimit {
		appTypes, gotCount, err := appTypeDao.SubCollectionList(m.Source{ID: sourceId}, d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the application types: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of application types, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(appTypes)
		want := int(wantCount) - d.Offset
		if want < 0 {
			want = 0
		}

		if want > d.Limit {
			want = d.Limit
		}
		if got != want {
			t.Errorf(`objects passed back from DB: want "%v", got "%v"`, want, got)
		}
	}
	DoneWithFixtures("offset_limit")
}

// TestApplicationTypeListOffsetAndLimit tests that List() in application type dao returns correct
// count value and correct count of returned objects
func TestApplicationTypeListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	CreateFixtures("offset_limit")

	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	wantCount := int64(len(fixtures.TestApplicationTypeData))

	for _, d := range fixtures.TestDataOffsetLimit {
		appTypes, gotCount, err := appTypeDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the app types: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of app types, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(appTypes)
		want := int(wantCount) - d.Offset
		if want < 0 {
			want = 0
		}

		if want > d.Limit {
			want = d.Limit
		}
		if got != want {
			t.Errorf(`objects passed back from DB: want "%v", got "%v"`, want, got)
		}
	}
	DoneWithFixtures("offset_limit")
}
