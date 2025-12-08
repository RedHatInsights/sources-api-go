package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestMetaDataSubcollectionListWithOffsetAndLimit tests that SubCollectionList() in meta data dao
// returns correct count value and correct count of returned objects
func TestMetaDataSubcollectionListWithOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	metaDataDao := GetMetaDataDao()

	appTypeId := fixtures.TestApplicationTypeData[0].Id
	// How many meta data with given application type id is in fixtures
	var wantCount int64

	for _, appMetaData := range fixtures.TestMetaDataData {
		if appMetaData.ApplicationTypeID == appTypeId {
			wantCount++
		}
	}

	for _, d := range fixtures.TestDataOffsetLimit {
		metaDatas, gotCount, err := metaDataDao.SubCollectionList(m.ApplicationType{Id: appTypeId}, d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the meta datas: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of meta datas, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(metaDatas)

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

	DropSchema("offset_limit")
}

// TestMetaDataListOffsetAndLimit tests that List() in meta data dao returns correct count value
// and correct count of returned objects
func TestMetaDataListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	metaDataDao := GetMetaDataDao()
	wantCount := int64(len(fixtures.TestMetaDataData))

	for _, d := range fixtures.TestDataOffsetLimit {
		metaDatas, gotCount, err := metaDataDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the meta datas: %s`, err)
		}

		// Check of count value returned form List()
		if wantCount != gotCount {
			t.Errorf(`incorrect count of meta datas, want "%d", got "%d"`, wantCount, gotCount)
		}

		// Check of object's count returned from List()
		got := len(metaDatas)

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

	DropSchema("offset_limit")
}
