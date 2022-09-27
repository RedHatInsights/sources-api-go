package dao

import (
	"errors"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestApplicationTypeSubCollectionListOffsetAndLimit tests that SubCollectionList() in application type dao returns
//
//	correct count value and correct count of returned objects
func TestApplicationTypeSubCollectionListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

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
	DropSchema("offset_limit")
}

// TestApplicationTypeListOffsetAndLimit tests that List() in application type dao returns correct
// count value and correct count of returned objects
func TestApplicationTypeListOffsetAndLimit(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("offset_limit")

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
	DropSchema("offset_limit")
}

func TestApplicationTypeGetByName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_type_by_name")
	wantAppType := fixtures.TestApplicationTypeData[1]

	tenantId := int64(1)
	appTypeDao := GetApplicationTypeDao(&tenantId)
	gotAppType, err := appTypeDao.GetByName(wantAppType.Name)
	if err != nil {
		t.Error(err)
	}

	if gotAppType.Name != wantAppType.Name {
		t.Errorf("want source type name '%s', got source type name '%s'", wantAppType.Name, gotAppType.Name)
	}

	DropSchema("app_type_by_name")
}

func TestApplicationTypeGetByNameNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_type_by_name")
	wantAppType := m.ApplicationType{Name: "not existing name"}

	tenantId := int64(1)
	appTypeDao := GetApplicationTypeDao(&tenantId)

	gotAppType, err := appTypeDao.GetByName(wantAppType.Name)
	if gotAppType != nil {
		t.Error("got application type object, want nil")
	}

	if !errors.Is(err, util.ErrNotFoundEmpty) {
		t.Errorf("want not found err, got '%v'", err)
	}

	DropSchema("app_type_by_name")
}
func TestApplicationTypeGetByNameBadRequest(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_type_by_name")
	wantAppType := m.ApplicationType{Name: "app type"}
	tenantId := int64(1)
	appTypeDao := GetApplicationTypeDao(&tenantId)

	gotAppType, err := appTypeDao.GetByName(wantAppType.Name)
	if gotAppType != nil {
		t.Error("got application type object, want nil")
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Errorf("want bad request err, got '%v'", err)
	}

	DropSchema("app_type_by_name")
}
