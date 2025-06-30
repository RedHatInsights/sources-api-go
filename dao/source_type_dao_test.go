package dao

import (
	"errors"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
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

func TestSourceTypeGetByName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("source_type_by_name")

	wantSourceType := fixtures.TestSourceTypeData[1]

	sourceTypeDao := GetSourceTypeDao()

	gotSourceType, err := sourceTypeDao.GetByName(wantSourceType.Name)
	if err != nil {
		t.Error(err)
	}

	if gotSourceType.Name != wantSourceType.Name {
		t.Errorf("want source type name '%s', got source type name '%s'", wantSourceType.Name, gotSourceType.Name)
	}

	DropSchema("source_type_by_name")
}

func TestSourceTypeGetByNameNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("source_type_by_name")

	wantSourceType := m.SourceType{Name: "not existing name"}

	sourceTypeDao := GetSourceTypeDao()

	gotSourceType, err := sourceTypeDao.GetByName(wantSourceType.Name)
	if gotSourceType != nil {
		t.Error("got source type object, want nil")
	}

	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf("want not found err, got '%v'", err)
	}

	DropSchema("source_type_by_name")
}

func TestSourceTypeGetByNameBadRequest(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("source_type_by_name")

	wantSourceType := m.SourceType{Name: "amazon"}

	sourceTypeDao := GetSourceTypeDao()

	gotSourceType, err := sourceTypeDao.GetByName(wantSourceType.Name)
	if gotSourceType != nil {
		t.Error("got source type object, want nil")
	}

	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("want bad request err, got '%v'", err)
	}

	DropSchema("source_type_by_name")
}
