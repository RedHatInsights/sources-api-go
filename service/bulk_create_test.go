package service

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/database"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TestParseEndpointsTestRegression is a regression test for RHCLOUD-19931. It tests that the "parseEndpoints" function
// correctly maps all the fields —including the one that was failing, the source ID— from the request to an endpoint
// struct.
func TestParseEndpointsTestRegression(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare all the fixtures to simulate a valid Bulk Create request.
	var endpointFixture = fixtures.TestEndpointData[0]

	endpointFixture.Role = util.StringRef("endpoint-role")
	endpointFixture.ReceptorNode = util.StringRef("endpoint-receptor-node")

	var verifySsl = true

	endpointFixture.VerifySsl = &verifySsl
	endpointFixture.CertificateAuthority = util.StringRef("endpoint-ca")

	var (
		sourceFixture = fixtures.TestSourceData[0]
		tenantFixture = fixtures.TestTenantData[0]
	)

	var reqEndpoints = []model.BulkCreateEndpoint{
		{
			EndpointCreateRequest: model.EndpointCreateRequest{
				Default:              false,
				ReceptorNode:         endpointFixture.ReceptorNode,
				Role:                 *endpointFixture.Role,
				Scheme:               endpointFixture.Scheme,
				Host:                 *endpointFixture.Host,
				Port:                 endpointFixture.Port,
				Path:                 *endpointFixture.Path,
				VerifySsl:            endpointFixture.VerifySsl,
				CertificateAuthority: endpointFixture.CertificateAuthority,
				AvailabilityStatus:   endpointFixture.AvailabilityStatus,
				SourceIDRaw:          sourceFixture.ID,
			},
		},
	}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{sourceFixture},
	}

	// Call the function under test.
	endpoints, err := parseEndpoints(reqEndpoints, &bulkCreateOutput, &tenantFixture)
	if err != nil {
		t.Errorf(`unexpected error when parsing the endpoints from bulk create: %s`, err)
	}

	if len(endpoints) != 1 {
		t.Errorf(`unexpected number of endpoints received. Want "%d", got "%d"`, 1, len(endpoints))
	}

	resultEndpoint := endpoints[0]

	{
		want := *endpointFixture.Scheme
		got := *resultEndpoint.Scheme

		if want != got {
			t.Errorf(`wrong endpoint scheme parsed. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := *endpointFixture.Host
		got := *resultEndpoint.Host

		if want != got {
			t.Errorf(`wrong endpoint host parsed. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := *endpointFixture.Path
		got := *resultEndpoint.Path

		if want != got {
			t.Errorf(`wrong endpoint path parsed. Want "%s", got "%s"`, want, got)
		}
	}

	{
		want := *endpointFixture.Port
		got := *resultEndpoint.Port

		if want != got {
			t.Errorf(`wrong endpoint port parsed. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := *endpointFixture.VerifySsl
		got := *resultEndpoint.VerifySsl

		if want != got {
			t.Errorf(`wrong endpoint verify ssl parsed. Want "%t", got "%t"`, want, got)
		}
	}

	{
		want := tenantFixture
		got := resultEndpoint.Tenant

		if want != got {
			t.Errorf(`wrong endpoint tenant parsed. Want "%v", got "%v"`, want, got)
		}
	}

	{
		want := endpointFixture.TenantID
		got := resultEndpoint.TenantID

		if want != got {
			t.Errorf(`wrong endpoint tenant ID parsed. Want "%d", got "%d"`, want, got)
		}
	}

	{
		want := endpointFixture.SourceID
		got := resultEndpoint.SourceID

		if want != got {
			t.Errorf(`wrong endpoint source id parsed. Want "%d", got "%d"`, want, got)
		}
	}
}

// TestParseSourcesWithSourceTypeId tests that correct output is returned for valid inputs
// and with source type id
func TestParseSourcesWithSourceTypeId(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeFixture := fixtures.TestSourceTypeData[0]
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name:            util.StringRef(sourceName),
				SourceTypeIDRaw: sourceTypeFixture.Id,
			},
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if err != nil {
		t.Errorf(`unexpected error when parsing the sources from bulk create: %s`, err)
	}

	// Check the results
	if len(sources) != 1 {
		t.Errorf("expected 1 source returned from parseSources() but got %d", len(sources))
	}

	sourceOut := sources[0]

	if sourceOut.AvailabilityStatus != model.InProgress {
		t.Errorf("expected availability status 'in_progress', got %s", sourceOut.AvailabilityStatus)
	}

	if sourceOut.SourceTypeID != sourceTypeFixture.Id {
		t.Errorf("expected source type id %d, got %d", sourceTypeFixture.Id, sourceOut.SourceTypeID)
	}

	if sourceOut.Name != sourceName {
		t.Errorf("expected source name %s, got %s", sourceName, sourceOut.Name)
	}

	if sourceOut.TenantID != tenant.Id {
		t.Errorf("expected tenant id %d, got %d", tenant.Id, sourceOut.TenantID)
	}

	if sourceOut.UserID != nil {
		t.Errorf("expected user id = nil, got %d", sourceOut.UserID)
	}
}

// TestParseSourcesWithSourceTypeName tests that correct output is returned for valid inputs
// and with source type name
func TestParseSourcesWithSourceTypeName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeFixture := fixtures.TestSourceTypeData[1]
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name: util.StringRef(sourceName),
			},
			SourceTypeName: sourceTypeFixture.Name,
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if err != nil {
		t.Errorf(`unexpected error when parsing the sources from bulk create: %s`, err)
	}

	// Check the results
	if len(sources) != 1 {
		t.Errorf("expected 1 source returned from parseSources() but got %d", len(sources))
	}

	sourceOut := sources[0]

	if sourceOut.AvailabilityStatus != model.InProgress {
		t.Errorf("expected availability status 'in_progress', got %s", sourceOut.AvailabilityStatus)
	}

	if sourceOut.SourceTypeID != sourceTypeFixture.Id {
		t.Errorf("expected source type id %d, got %d", sourceTypeFixture.Id, sourceOut.SourceTypeID)
	}

	if sourceOut.Name != sourceName {
		t.Errorf("expected source name %s, got %s", sourceName, sourceOut.Name)
	}

	if sourceOut.TenantID != tenant.Id {
		t.Errorf("expected tenant id %d, got %d", tenant.Id, sourceOut.TenantID)
	}

	if sourceOut.UserID != nil {
		t.Errorf("expected user id = nil, got %d", sourceOut.UserID)
	}
}

// TestParseSourcesBadRequestInvalidSourceTypeId tests that bad request is returned
// for invalid source type id
func TestParseSourcesBadRequestInvalidSourceTypeId(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeId := "wrong id"
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name:            util.StringRef(sourceName),
				SourceTypeIDRaw: sourceTypeId,
			},
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources and check the results
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("expected bad request error, got <%s>", err)
	}

	if sources != nil {
		t.Error("ghost infected the return")
	}
}

// TestParseSourcesNotFoundInvalidSourceTypeId tests that not found is returned
// for not existing source type id
func TestParseSourcesNotFoundInvalidSourceTypeId(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeId := "1000"
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name:            util.StringRef(sourceName),
				SourceTypeIDRaw: sourceTypeId,
			},
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources and check the results
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf("expected not found error, got <%s>", err)
	}

	if sources != nil {
		t.Error("ghost infected the return")
	}
}

// TestParseSourcesBadRequestInvalidSourceTypeName tests that bad request is returned
// for invalid source type name
func TestParseSourcesBadRequestInvalidSourceTypeName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeName := "invalid name"
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name: util.StringRef(sourceName),
			},
			SourceTypeName: sourceTypeName,
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources and check the results
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("expected bad request error, got <%s>", err)
	}

	if sources != nil {
		t.Error("ghost infected the return")
	}
}

// TestParseSourcesBadRequestMissingSourceType tests that bad request is returned
// when source type id or name is missing in the request
func TestParseSourcesBadRequestMissingSourceType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name: util.StringRef(sourceName),
			},
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources and check the results
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("expected bad request error, got <%s>", err)
	}

	if sources != nil {
		t.Error("ghost infected the return")
	}
}

// TestParseSourcesBadRequestValidationFails tests that bad request is returned
// when validation of source create request fails
func TestParseSourcesBadRequestValidationFails(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeName := "google"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				AvailabilityStatus: model.Available,
			},
			SourceTypeName: sourceTypeName,
		},
	}

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	// Parse the sources and check the results
	var err error

	sources, err := parseSources(reqSources, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("expected bad request error, got <%s>", err)
	}

	if sources != nil {
		t.Error("ghost infected the return")
	}
}

// TestParseSourcesWithSourceOwnership tests that user id is correctly set
// when ownership is present for the source
func TestParseSourcesWithSourceOwnership(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	sourceTypeName := "bitbucket"
	applicationTypeName := "app-studio"
	sourceName := "Source for TestParseSources()"

	var reqSources = []model.BulkCreateSource{
		{
			SourceCreateRequest: model.SourceCreateRequest{
				Name: util.StringRef(sourceName),
			},
			SourceTypeName: sourceTypeName,
		},
	}

	userID := "test_user"
	tenant := fixtures.TestTenantData[0]
	userDao := dao.GetUserDao(&tenant.Id)

	user, err := userDao.FindOrCreate(userID)
	if err != nil {
		t.Errorf(`Error getting or creating the tenant. Want nil error, got "%s"`, err)
	}

	userResource := model.UserResource{
		User:                  user,
		SourceNames:           []string{sourceName},
		ApplicationTypesNames: []string{applicationTypeName},
	}

	// Parse the sources
	sources, err := parseSources(reqSources, &tenant, &userResource)
	if err != nil {
		t.Errorf(`unexpected error when parsing the sources from bulk create: %s`, err)
	}

	// Check the results
	if len(sources) != 1 {
		t.Errorf("expected 1 source returned from parseSources() but got %d", len(sources))
	}

	if *sources[0].UserID != user.Id {
		t.Errorf("expected user id %d, got %d", user.Id, *sources[0].UserID)
	}
}

// TestParseApplications tests that correct output is returned for valid inputs
func TestParseApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	sourceType := fixtures.TestSourceTypeData[0]
	source.SourceType = sourceType

	appType := fixtures.TestApplicationTypeData[5]
	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	// 2 combinations for bulk create application
	// 		1) with app type id raw
	// 		2) with app type name

	reqApplicationsAppTypeId := []model.BulkCreateApplication{
		{
			ApplicationCreateRequest: model.ApplicationCreateRequest{
				ApplicationTypeIDRaw: appType.Id,
			},
			SourceName: source.Name,
		},
	}

	reqApplicationsAppTypeName := []model.BulkCreateApplication{
		{
			ApplicationTypeName: appType.Name,
			SourceName:          source.Name,
		},
	}

	reqApplicationsList := [][]model.BulkCreateApplication{
		reqApplicationsAppTypeId,
		reqApplicationsAppTypeName,
	}

	// Parse the applications
	for _, reqApplications := range reqApplicationsList {
		apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
		if err != nil {
			t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
		}

		// Check the results
		if len(apps) != 1 {
			t.Errorf("expected 1 application returned from parseApplications() but got %d", len(apps))
		}

		appOut := apps[0]

		if appOut.SourceID != source.ID {
			t.Errorf("expected source id %d, got %d", source.ID, appOut.SourceID)
		}

		if appOut.ApplicationTypeID != appType.Id {
			t.Errorf("expected application type id %d, got %d", appType.Id, appOut.ApplicationTypeID)
		}

		if appOut.TenantID != tenant.Id {
			t.Errorf("expected tenant id %d, got %d", tenant.Id, appOut.TenantID)
		}

		if appOut.UserID != nil {
			t.Errorf("expected user id = nil, got %d", appOut.UserID)
		}
	}
}

// TestParseApplicationsBadRequestApplicationNotLinked tests situation when all of the applications
// did not get linked up
func TestParseApplicationsBadRequestApplicationNotLinked(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	anotherSource := fixtures.TestSourceData[1]

	appType := fixtures.TestApplicationTypeData[5]
	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{anotherSource},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			ApplicationTypeName: appType.Name,
			SourceName:          source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestParseApplicationsBadRequestWithoutAppType tests that bad request is returned
// when application type id or name is missing in the request
func TestParseApplicationsBadRequestWithoutAppType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			SourceName: source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestParseApplicationsBadRequestInvalidAppTypeId tests that bad request is returned
// for invalid application type raw id in the request
func TestParseApplicationsBadRequestInvalidAppTypeId(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	appTypeIdRaw := "amazon"

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			ApplicationCreateRequest: model.ApplicationCreateRequest{
				ApplicationTypeIDRaw: appTypeIdRaw,
			},
			SourceName: source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestParseApplicationsAppTypeIdNotFound tests that not found is returned
// for not existing application type id
func TestParseApplicationsAppTypeIdNotFound(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	appTypeIdRaw := "1000"

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			ApplicationCreateRequest: model.ApplicationCreateRequest{
				ApplicationTypeIDRaw: appTypeIdRaw,
			},
			SourceName: source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrNotFound{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestParseApplicationsBadRequestInvalidAppTypeName tests that bad request is returned
// for invalid application type name in the request
func TestParseApplicationsBadRequestInvalidAppTypeName(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	appTypeName := "not existing app type name"

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			ApplicationTypeName: appTypeName,
			SourceName:          source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestParseApplicationsBadRequestNotCompatible tests that bad request is returned
// for not compatible app type with source type
func TestParseApplicationsBadRequestNotCompatible(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	// Prepare test data
	source := fixtures.TestSourceData[0]
	appType := fixtures.TestApplicationTypeData[1]

	tenant := fixtures.TestTenantData[0]
	userResource := model.UserResource{}

	bulkCreateOutput := model.BulkCreateOutput{
		Sources: []model.Source{source},
	}

	reqApplications := []model.BulkCreateApplication{
		{
			ApplicationTypeName: appType.Name,
			SourceName:          source.Name,
		},
	}

	// Parse the applications
	apps, err := parseApplications(reqApplications, &bulkCreateOutput, &tenant, &userResource)
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf(`unexpected error when parsing the applications from bulk create: %s`, err)
	}

	// Check the results
	if apps != nil {
		t.Errorf("expected nil returned from parseApplications() but got %d applications", len(apps))
	}
}

// TestShouldSourceApplicationSetAvailable tests that the function under test
// correctly identifies which source-application pairs should be set as
// "available" by default.
func TestShouldSourceApplicationSetAvailable(t *testing.T) {
	testCases := []struct {
		Source         model.Source
		Application    model.Application
		ExpectedResult bool
	}{
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "amazon"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/image-builder"}},
			ExpectedResult: true,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "google"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/image-builder"}},
			ExpectedResult: false,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "azure"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/image-builder"}},
			ExpectedResult: false,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "amazon"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/cloud-meter"}},
			ExpectedResult: false,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "google"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/cloud-meter"}},
			ExpectedResult: true,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "azure"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/cloud-meter"}},
			ExpectedResult: false,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "amazon"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/another-app"}},
			ExpectedResult: false,
		},
		{
			Source:         model.Source{SourceType: model.SourceType{Name: "google"}},
			Application:    model.Application{ApplicationType: model.ApplicationType{Name: "/insights/platform/another-app"}},
			ExpectedResult: false,
		},
	}

	for _, testCase := range testCases {
		want := testCase.ExpectedResult
		got := shouldSourceApplicationSetAvailable(testCase.Source, testCase.Application)

		if want != got {
			t.Errorf(`incorrectly identified source type "%s" and application type "%s" pair. Want "%t", got "%t"`, testCase.Source.SourceType.Name, testCase.Application.ApplicationType.Name, want, got)
		}
	}
}

// TestSetCorrespondingResourcesAsAvailable tests that the function under test
// overrides the availability status for the relevant source-application pairs.
//
// It sets up three sources with a "Cost Management" and an "Image Builder"
// application, and three authentications each: one for the source, and one for
// each application.
//
// Two of the sources and all its dependant elements should get the
// availability status overridden, and the last one should keep its original
// availability status left untouched.
func TestSetCorrespondingResourcesAsAvailable(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	database.SwitchSchema("test_bulk_create_corresponding_resouces_available")

	defer func() {
		database.DropSchema("test_bulk_create_corresponding_resouces_available")
		database.SwitchSchema("service")
	}()

	// Get the application types.
	cloudMeterAppType := fixtures.TestApplicationTypeData[6]
	imageBuilderAppType := fixtures.TestApplicationTypeData[7]

	// Get the sources.
	amazonSource := fixtures.TestSourceData[6]
	googleSource := fixtures.TestSourceData[7]
	bitBucketSource := fixtures.TestSourceData[8]

	// Create the applications and the authentications for the Amazon
	// source.
	amazonSourceApps := []model.Application{
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           amazonSource.ID,
			ApplicationType:    cloudMeterAppType,
			ApplicationTypeID:  cloudMeterAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           amazonSource.ID,
			ApplicationType:    imageBuilderAppType,
			ApplicationTypeID:  imageBuilderAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the applications so that we obtain an ID for them, and so that
	// we can associate them to the authentications.
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&amazonSourceApps).Error
		if err != nil {
			return fmt.Errorf("failed to create applications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create applications in the database: %s`, err)
	}

	amazonAuthentications := []model.Authentication{
		{
			ID:                 "amazon-source-auth",
			AuthType:           "basic",
			ResourceType:       "Source",
			ResourceID:         amazonSource.ID,
			SourceID:           amazonSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "cloud-meter-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         amazonSourceApps[0].ID, // Cloud-Meter.
			SourceID:           amazonSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "image-builder-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         amazonSourceApps[1].ID, // Image-Builder.
			SourceID:           amazonSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the authentications as well.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&amazonAuthentications).Error
		if err != nil {
			return fmt.Errorf("failed to create authentications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create authentications in the database: %s`, err)
	}

	// Create the applications and the authentications for the Google
	// source.
	googleSourceApps := []model.Application{
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           googleSource.ID,
			ApplicationType:    cloudMeterAppType,
			ApplicationTypeID:  cloudMeterAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           googleSource.ID,
			ApplicationType:    imageBuilderAppType,
			ApplicationTypeID:  imageBuilderAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the applications so that we obtain an ID for them, and so that
	// we can associate them to the authentications.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&googleSourceApps).Error
		if err != nil {
			return fmt.Errorf("failed to create applications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create applications in the database: %s`, err)
	}

	googleAuthentications := []model.Authentication{
		{
			ID:                 "google-source-auth",
			AuthType:           "basic",
			ResourceType:       "Source",
			ResourceID:         googleSource.ID,
			SourceID:           googleSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "cloud-meter-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         googleSourceApps[0].ID, // Cloud-Meter.
			SourceID:           googleSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "image-builder-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         googleSourceApps[1].ID, // Image-Builder.
			SourceID:           googleSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the authentications as well.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&googleAuthentications).Error
		if err != nil {
			return fmt.Errorf("failed to create authentications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create authentications in the database: %s`, err)
	}

	// Create the applications and the authentications for the BitBucket
	// source.
	bitBucketSourceApps := []model.Application{
		// Source 2 applications (google): 2 should satisfy, 1 should not
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           bitBucketSource.ID,
			ApplicationType:    cloudMeterAppType,
			ApplicationTypeID:  cloudMeterAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			AvailabilityStatus: model.InProgress,
			SourceID:           bitBucketSource.ID,
			ApplicationType:    imageBuilderAppType,
			ApplicationTypeID:  imageBuilderAppType.Id,
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the applications so that we obtain an ID for them, and so that
	// we can associate them to the authentications.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&bitBucketSourceApps).Error
		if err != nil {
			return fmt.Errorf("failed to create applications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create applications in the database: %s`, err)
	}

	bitBucketAuthentications := []model.Authentication{
		{
			ID:                 "bitbucket-source-auth",
			AuthType:           "basic",
			ResourceType:       "Source",
			ResourceID:         bitBucketSource.ID,
			SourceID:           bitBucketSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "cloud-meter-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         bitBucketSourceApps[0].ID, // Cloud-Meter.
			SourceID:           bitBucketSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
		{
			ID:                 "image-builder-auth",
			AuthType:           "basic",
			ResourceType:       "Application",
			ResourceID:         bitBucketSourceApps[1].ID, // Image-Builder.
			SourceID:           bitBucketSource.ID,
			AvailabilityStatus: util.StringRef(model.InProgress),
			TenantID:           fixtures.TestTenantData[0].Id,
		},
	}

	// Create the authentications as well.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Omit(clause.Associations).Create(&bitBucketAuthentications).Error
		if err != nil {
			return fmt.Errorf("failed to create authentications in database: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unable to create authentications in the database: %s`, err)
	}

	// Prepare the slices for the function under test.
	sources := []model.Source{amazonSource, googleSource, bitBucketSource}

	applications := []model.Application{}
	applications = append(applications, amazonSourceApps...)
	applications = append(applications, googleSourceApps...)
	applications = append(applications, bitBucketSourceApps...)

	authentications := []model.Authentication{}
	authentications = append(authentications, amazonAuthentications...)
	authentications = append(authentications, googleAuthentications...)
	authentications = append(authentications, bitBucketAuthentications...)

	// Call the function under test.
	err = dao.DB.Transaction(func(tx *gorm.DB) error {
		err = setCorrespondingResourcesAsAvailable(tx, sources, applications, authentications)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Errorf(`unexpected error when calling the function under test: %s`, err)
	}

	// Verify that the Amazon source is available.
	if model.Available != sources[0].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source availability status. Want "%s", got "%s"`, model.Available, sources[0].AvailabilityStatus)
	}

	// Verify that the Amazon source's Cost Management application is still in
	// progress.
	if model.InProgress != applications[0].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source's Cost Management availability status. Want "%s", got "%s"`, model.InProgress, applications[0].AvailabilityStatus)
	}

	// Verify that the Amazon source's Image Builder application is available.
	if model.Available != applications[1].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source's Image Builder availability status. Want "%s", got "%s"`, model.InProgress, applications[1].AvailabilityStatus)
	}

	// Verify that the Amazon source's authentication is available, that the
	// Cost Management's authentication is still in progress, and that Image
	// Builder's authentication is available.
	if model.Available != *authentications[0].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source's authentication's status. Want "%s", got "%s"`, model.Available, *authentications[0].AvailabilityStatus)
	}

	if model.InProgress != *authentications[1].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source's Cost Management application's authentication's availability status. Want "%s", got "%s"`, model.InProgress, *authentications[1].AvailabilityStatus)
	}

	if model.Available != *authentications[2].AvailabilityStatus {
		t.Errorf(`Unexpected Amazon source's Image Builder application's authentication's availability status. Want "%s", got "%s"`, model.Available, *authentications[2].AvailabilityStatus)
	}

	// Verify that the Google source is available.
	if model.Available != sources[1].AvailabilityStatus {
		t.Errorf(`Unexpected Google source availability status. Want "%s", got "%s"`, model.Available, sources[1].AvailabilityStatus)
	}

	// Verify that the Google source's Cloud Meter application is available.
	if model.Available != applications[2].AvailabilityStatus {
		t.Errorf(`Unexpected Google source's Cloud Meter availability status. Want "%s", got "%s"`, model.Available, applications[2].AvailabilityStatus)
	}

	// Verify that the Google source's Image Builder application is still in progress.
	if model.InProgress != applications[3].AvailabilityStatus {
		t.Errorf(`Unexpected Google source's Image Builder availability status. Want "%s", got "%s"`, model.InProgress, applications[3].AvailabilityStatus)
	}

	// Verify that the Google source's authentication is available, that the
	// Cloud Meter's authentication is available, and that Image Builder's
	// authentication is still in progress.
	if model.Available != *authentications[3].AvailabilityStatus {
		t.Errorf(`Unexpected Google source's authentication's status. Want "%s", got "%s"`, model.Available, *authentications[3].AvailabilityStatus)
	}

	if model.Available != *authentications[4].AvailabilityStatus {
		t.Errorf(`Unexpected Google source's Cloud Meter application's authentication's availability status. Want "%s", got "%s"`, model.Available, *authentications[4].AvailabilityStatus)
	}

	if model.InProgress != *authentications[5].AvailabilityStatus {
		t.Errorf(`Unexpected Google source's Image Builder application's authentication's availability status. Want "%s", got "%s"`, model.InProgress, *authentications[5].AvailabilityStatus)
	}

	// Verify that the BitBucket source is still in progress.
	if model.InProgress != sources[2].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source availability status. Want "%s", got "%s"`, model.InProgress, sources[2].AvailabilityStatus)
	}

	// Verify that the BitBucket source's Cloud Meter application is still in
	// progress.
	if model.InProgress != applications[4].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source's Cloud Meter availability status. Want "%s", got "%s"`, model.InProgress, applications[4].AvailabilityStatus)
	}

	// Verify that the BitBucket source's Image Builder application is still in
	// progress.
	if model.InProgress != applications[5].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source's Image Builder availability status. Want "%s", got "%s"`, model.InProgress, applications[5].AvailabilityStatus)
	}

	// Verify that the BitBucket source's authentication is still in progress,
	// that the Cloud Meter's authentication is still in progress, and that
	// Image Builder's authentication is still in progress.
	if model.InProgress != *authentications[6].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source's authentication's status. Want "%s", got "%s"`, model.InProgress, *authentications[6].AvailabilityStatus)
	}

	if model.InProgress != *authentications[7].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source's Cloud Meter application's authentication's availability status. Want "%s", got "%s"`, model.InProgress, *authentications[7].AvailabilityStatus)
	}

	if model.InProgress != *authentications[8].AvailabilityStatus {
		t.Errorf(`Unexpected BitBucket source's Image Builder application's authentication's availability status. Want "%s", got "%s"`, model.InProgress, *authentications[8].AvailabilityStatus)
	}

	databaseChecks := struct {
		ExpectedAvailableSources          []model.Source
		ExpectedInProgressSources         []model.Source
		ExpectedAvailableApplications     []model.Application
		ExpectedInProgressApplications    []model.Application
		ExpectedAvailableAuthentications  []model.Authentication
		ExpectedInProgressAuthentications []model.Authentication
	}{
		ExpectedAvailableSources: []model.Source{
			sources[0], // Amazon.
			sources[1], // Google.
		},
		ExpectedInProgressSources: []model.Source{
			sources[2], // BitBucket.
		},
		ExpectedAvailableApplications: []model.Application{
			applications[1], // Amazon — Image Builder.
			applications[2], // Google — Cloud Meter.
		},
		ExpectedInProgressApplications: []model.Application{
			applications[0], // Amazon — Cost Management.
			applications[3], // Google — Image Builder.
			applications[4], // BitBucket — Cost Management.
			applications[5], // BitBucket — Image Builder.
		},
		ExpectedAvailableAuthentications: []model.Authentication{
			authentications[0], // Amazon source authentication.
			authentications[2], // Amazon Image Builder authentication.
			authentications[3], // Google source authentication.
			authentications[4], // Google Cloud Meter authentication.
		},
		ExpectedInProgressAuthentications: []model.Authentication{
			authentications[1], // Amazon Cost Management authentication.
			authentications[5], // Google Image Builder authentication.
			authentications[6], // BitBucket source authentication.
			authentications[7], // BitBucket Cost Management authentication.
			authentications[8], // BitBucket Image Builder authentication.
		},
	}

	// Get the DAO.
	tenantID := fixtures.TestTenantData[0].Id
	requestParams := dao.RequestParams{TenantID: &tenantID}

	sourceDao := dao.GetSourceDao(&requestParams)

	// Check available sources.
	for _, expectedSource := range databaseChecks.ExpectedAvailableSources {
		dbSource, err := sourceDao.GetById(&expectedSource.ID)
		if err != nil {
			t.Errorf(`Error fetching source %d from database: %s`, expectedSource.ID, err)
		}

		if dbSource.AvailabilityStatus != model.Available {
			t.Errorf(`Source %d availability status not persisted to database. Want "%s", got "%s"`, expectedSource.ID, model.Available, dbSource.AvailabilityStatus)
		}
	}

	// Check in-progress sources.
	for _, expectedSource := range databaseChecks.ExpectedInProgressSources {
		dbSource, err := sourceDao.GetById(&expectedSource.ID)
		if err != nil {
			t.Errorf(`Error fetching source %d from database: %s`, expectedSource.ID, err)
		}

		if dbSource.AvailabilityStatus != model.InProgress {
			t.Errorf(`Source %d availability status not persisted to database. Want "%s", got "%s"`, expectedSource.ID, model.InProgress, dbSource.AvailabilityStatus)
		}
	}

	// Verify applications in database
	applicationDao := dao.GetApplicationDao(&requestParams)

	// Check available applications.
	for _, expectedApp := range databaseChecks.ExpectedAvailableApplications {
		dbApp, err := applicationDao.GetById(&expectedApp.ID)
		if err != nil {
			t.Errorf(`Error fetching application %d from database: %s`, expectedApp.ID, err)
		}

		if dbApp.AvailabilityStatus != model.Available {
			t.Errorf(`Application %d availability status not persisted to database. Want "%s", got "%s"`, expectedApp.ID, model.Available, dbApp.AvailabilityStatus)
		}
	}

	// Check in-progress applications.
	for _, expectedApp := range databaseChecks.ExpectedInProgressApplications {
		dbApp, err := applicationDao.GetById(&expectedApp.ID)
		if err != nil {
			t.Errorf(`Error fetching application %d from database: %s`, expectedApp.ID, err)
		}

		if dbApp.AvailabilityStatus != model.InProgress {
			t.Errorf(`Application %d availability status not persisted to database. Want "%s", got "%s"`, expectedApp.ID, model.InProgress, dbApp.AvailabilityStatus)
		}
	}

	// Verify authentications in database.
	authenticationDao := dao.GetAuthenticationDao(&requestParams)

	// Check available authentications.
	for _, expectedAuth := range databaseChecks.ExpectedAvailableAuthentications {
		dbAuth, err := authenticationDao.GetById(strconv.FormatInt(expectedAuth.DbID, 10))
		if err != nil {
			t.Errorf(`Error fetching authentication %d from database: %s`, expectedAuth.DbID, err)
		}

		if *dbAuth.AvailabilityStatus != model.Available {
			t.Errorf(`Authentication %d availability status not persisted to database. Want "%s", got "%s"`, expectedAuth.DbID, model.Available, *dbAuth.AvailabilityStatus)
		}
	}

	// Check in-progress authentications.
	for _, expectedAuth := range databaseChecks.ExpectedInProgressAuthentications {
		dbAuth, err := authenticationDao.GetById(strconv.FormatInt(expectedAuth.DbID, 10))
		if err != nil {
			t.Errorf(`Error fetching authentication %d from database: %s`, expectedAuth.DbID, err)
		}

		if *dbAuth.AvailabilityStatus != model.InProgress {
			t.Errorf(`Authentication %d availability status not persisted to database. Want "%s", got "%s"`, expectedAuth.DbID, model.InProgress, *dbAuth.AvailabilityStatus)
		}
	}
}
