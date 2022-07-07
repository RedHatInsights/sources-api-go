package service

import (
	"testing"

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

	var sourceFixture = fixtures.TestSourceData[0]
	var tenantFixture = fixtures.TestTenantData[0]

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
