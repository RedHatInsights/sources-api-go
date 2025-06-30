package service

import (
	"errors"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
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
