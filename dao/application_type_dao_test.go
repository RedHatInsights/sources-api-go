package dao

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestApplicationTypeSubCollectionListOffsetAndLimit tests that SubCollectionList() in application type dao returns
// correct count value and correct count of returned objects
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

// TestApplicationTypeSubscollectionDisabledApplicationTypes tests that when
// fetching the associated application types for an entity, the disabled ones
// are not returned.
func TestApplicationTypeSubscollectionDisabledApplicationTypes(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_sub_disabled_app_types")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[0].Name}

	// Get the expected application types for the given source from the
	// fixtures.
	sourceId := fixtures.TestSourceData[0].ID

	expectedAppTypes := []m.ApplicationType{}

	for _, app := range fixtures.TestApplicationData {
		if app.SourceID == sourceId {
			for _, appType := range fixtures.TestApplicationTypeData {
				if app.ApplicationTypeID == appType.Id && appType.Name != fixtures.TestApplicationTypeData[0].Name {
					expectedAppTypes = append(expectedAppTypes, appType)
				}
			}
		}
	}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)

	appTypes, gotCount, err := appTypeDao.SubCollectionList(m.Source{ID: sourceId}, 100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the application types: %s`, err)
	}

	// Verify that only the unfiltered application types have been fetched.
	if int64(len(expectedAppTypes)) != gotCount {
		t.Errorf(`incorrect count of application types, want "%d", got "%d"`, len(expectedAppTypes), gotCount)
	}

	for i, expectedAppType := range expectedAppTypes {
		if expectedAppType.Id != appTypes[i].Id {
			t.Errorf(`unexpected app type fetched from the database. Want "%v", got "%v`, expectedAppType, appTypes[i])
		}

		if expectedAppType.Name != appTypes[i].Name {
			t.Errorf(`unexpected app type fetched from the database. Want "%v", got "%v`, expectedAppType, appTypes[i])
		}
	}

	DropSchema("app_types_sub_disabled_app_types")
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

// TestApplicationTypeListDisabledApplicationTypes tests that when listing the
// application types, the disabled ones are not returned.
func TestApplicationTypeListDisabledApplicationTypes(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_list_disabled_app_types")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[0].Name}

	// Get all the expected application types.
	expectedAppTypes := []m.ApplicationType{}

	for _, appType := range fixtures.TestApplicationTypeData {
		if appType.Name != config.Get().DisabledApplicationTypes[0] {
			expectedAppTypes = append(expectedAppTypes, appType)
		}
	}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)

	appTypes, gotCount, err := appTypeDao.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the app types: %s`, err)
	}

	if int64(len(expectedAppTypes)) != gotCount {
		t.Errorf(`incorrect count of app types, want "%d", got "%d"`, len(expectedAppTypes), gotCount)
	}

	// Verify that only the unfiltered application types have been fetched.
	if int64(len(expectedAppTypes)) != gotCount {
		t.Errorf(`incorrect count of application types, want "%d", got "%d"`, len(expectedAppTypes), gotCount)
	}

	for _, expectedAppType := range expectedAppTypes {
		atIndex := slices.IndexFunc(appTypes, func(appType m.ApplicationType) bool {
			return expectedAppType.Id == appType.Id && expectedAppType.Name == appType.Name
		})

		if atIndex == -1 {
			t.Errorf(`unexpected application types fetched. Want "%v", got "%v"`, expectedAppTypes, appTypes)
		}
	}

	DropSchema("app_types_list_disabled_app_types")
}

// TestApplicationTypeGetById tests that the function under test is able to
// fetch an application by its ID.
func TestApplicationTypeGetById(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_get_by_id")

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	gotAppType, err := appTypeDao.GetById(&fixtures.TestApplicationTypeData[0].Id)

	// Verify that no error was returned.
	if err != nil {
		t.Errorf(`unexpected error when fetching an application type by its id: "%s"`, err)
	}

	// Verify that the correct application type was fetched.
	if gotAppType.Id != fixtures.TestApplicationTypeData[0].Id {
		t.Errorf(`unexpected application type fetched. Want "%v", got "%v"`, fixtures.TestApplicationTypeData[0], gotAppType)
	}

	DropSchema("app_types_get_by_id")
}

// TestApplicationTypeGetByIdDisabledApplicationTypes tests that a disabled
// application type cannot be fetched.
func TestApplicationTypeGetByIdDisabledApplicationTypes(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_get_by_id_disabled_app_types")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[0].Name}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	gotAppType, err := appTypeDao.GetById(&fixtures.TestApplicationTypeData[0].Id)

	// Verify that the expected error was returned.
	if gotAppType != nil {
		t.Error("got application type object, want nil")
	}

	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf("want not found err, got '%v'", err)
	}

	DropSchema("app_types_get_by_id_disabled_app_types")
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

// TestApplicationTypeGetByNameDisabledApplicationTypes tests that an
// application type cannot be fetched by its name if it is disabled.
func TestApplicationTypeGetByNameDisabledApplicationTypes(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_get_by_name_disabled_app_types")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[4].Name}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	gotAppType, err := appTypeDao.GetByName(fixtures.TestApplicationTypeData[4].Name)

	// Verify that the expected error was returned.
	if gotAppType != nil {
		t.Error("got application type object, want nil")
	}

	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf("want not found err, got '%v'", err)
	}

	DropSchema("app_types_get_by_name_disabled_app_types")
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

	if !errors.As(err, &util.ErrNotFound{}) {
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

	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("want bad request err, got '%v'", err)
	}

	DropSchema("app_type_by_name")
}

// TestApplicationTypeCompatibleWithSourceSourceNotFound tests that when a
// non-existent source is specified, the function under test returns an error.
func TestApplicationTypeCompatibleWithSourceSourceNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_app_type_compatible_non_existing_source")

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	err := appTypeDao.ApplicationTypeCompatibleWithSource(fixtures.TestApplicationTypeData[0].Id, 12345)

	// Verify that the expected error was returned.
	if err == nil {
		t.Error("want error, got nil")
	}

	if !strings.Contains(err.Error(), "source not found") {
		t.Errorf(`want "source not found" error, got "%s"`, err)
	}

	DropSchema("app_types_source_app_type_compatible_non_existing_source")
}

// TestApplicationTypeCompatibleWithSourceSourceNotFound tests that when a non
// compatible source and application type are specified, the function under
// test returns an error.
func TestApplicationTypeCompatibleWithSourceNotCompatible(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_app_type_not_compatible")

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	err := appTypeDao.ApplicationTypeCompatibleWithSource(fixtures.TestApplicationTypeData[4].Id, fixtures.TestSourceData[5].ID)

	// Verify that the expected error was returned.
	if err == nil {
		t.Error("want error, got nil")
	}

	if !strings.Contains(err.Error(), "record not found") {
		t.Errorf(`want "record not found" error, got "%s"`, err)
	}

	DropSchema("app_types_source_app_type_not_compatible")
}

// TestApplicationTypeCompatibleWithSource tests that when a source is
// compatible with the given application type, the function under test does not
// return an error.
func TestApplicationTypeCompatibleWithSource(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_app_type_compatible")

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	err := appTypeDao.ApplicationTypeCompatibleWithSource(fixtures.TestApplicationTypeData[5].Id, fixtures.TestSourceData[0].ID)

	// Verify that no error was returned.
	if err != nil {
		t.Errorf("want nil, got error: %s", err)
	}

	DropSchema("app_types_source_app_type_compatible")
}

// TestApplicationTypeCompatibleWithSourceDisabledApplicationType tests that
// when an application type is disabled, the function under test returns an
// error even for a source that would be compatible with the disabled
// application type.
func TestApplicationTypeCompatibleWithSourceDisabledApplicationType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_app_type_compatible_disabled_app_type")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[5].Name}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	err := appTypeDao.ApplicationTypeCompatibleWithSource(fixtures.TestApplicationTypeData[5].Id, fixtures.TestSourceData[0].ID)

	// Verify that the expected error was returned.
	if err == nil {
		t.Error("want error, got nil")
	}

	if !strings.Contains(err.Error(), "record not found") {
		t.Errorf(`want "record not found" error, got "%s"`, err)
	}

	DropSchema("app_types_source_app_type_compatible_disabled_app_type")
}

// TestApplicationTypesGetSuperkeyResultType tests that the function under test
// fetches the expected authentication type.
func TestApplicationTypesGetSuperkeyResultType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_superkey_result_type")

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	authType, err := appTypeDao.GetSuperKeyResultType(fixtures.TestApplicationTypeData[5].Id, "amazon")

	// Verify that no error was returned.
	if err != nil {
		t.Errorf("unexpected error when fetching the superkey result type: %s", err)
	}

	if authType != "arn" {
		t.Errorf(`unexpected authentication type fetched. Want "arn", got "%s"`, authType)
	}

	DropSchema("app_types_source_superkey_result_type")
}

// TestApplicationTypesGetSuperkeyResultTypeDisabledApplicationType tests that
// the function under test returns an empty authentication type if the
// given application type is disabled.
func TestApplicationTypesGetSuperkeyResultTypeDisabledApplicationType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	SwitchSchema("app_types_source_superkey_result_type_disabled_app_type")

	// Disable an application type.
	defer func() { config.Get().DisabledApplicationTypes = []string{} }()

	config.Get().DisabledApplicationTypes = []string{fixtures.TestApplicationTypeData[5].Name}

	// Call the function under test.
	appTypeDao := GetApplicationTypeDao(&fixtures.TestTenantData[0].Id)
	authType, err := appTypeDao.GetSuperKeyResultType(fixtures.TestApplicationTypeData[5].Id, "amazon")

	// Verify that no error was returned.
	if err != nil {
		t.Errorf("unexpected error when fetching the superkey result type: %s", err)
	}

	if authType != "" {
		t.Errorf(`unexpected authentication type fetched. Want "", got "%s"`, authType)
	}

	DropSchema("app_types_source_superkey_result_type_disabled_app_type")
}
