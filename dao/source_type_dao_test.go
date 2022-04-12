package dao

import (
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestSourceTypeListOffsetAndLimit tests that List() in source type dao returns correct count value
// and correct count of returned objects
func TestSourceTypeListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

	sourceTypeDao := GetSourceTypeDao()
	wantCount := int64(len(fixtures.TestSourceTypeData))

	for _, d := range fixtures.TestDataOffsetLimit {
		sourceTypes, gotCount, err := sourceTypeDao.List(d.Limit, d.Offset, []util.Filter{})
		if err != nil {
			t.Errorf(`unexpected error when listing the source types: %s`, err)
		}

		if wantCount != gotCount {
			t.Errorf(`incorrect count of source types, want "%d", got "%d"`, wantCount, gotCount)
		}

		got := len(sourceTypes)
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
