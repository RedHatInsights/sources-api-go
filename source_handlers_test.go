package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/RedHatInsights/sources-api-go/middleware"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

var notExistingTenantId = int64(309832948930)

func TestSourceListAuthentications(t *testing.T) {
	originalSecretStore := conf.SecretStore
	tenantId := int64(1)
	sourceId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	err := SourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	var wantCount int
	for _, a := range fixtures.TestAuthenticationData {
		if a.SourceID == sourceId && a.TenantID == tenantId {
			wantCount++
		}
	}

	if out.Meta.Count != wantCount {
		t.Error("count not set correctly")
	}

	auth1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if config.IsVaultOn() {
		want := fixtures.TestAuthenticationData[0].ID
		got := auth1["id"]

		if want != got {
			t.Errorf(`the IDs from the authentication don't match. Want "%s", got "%s"'`, want, got)
		}
	} else {
		if !util.SliceContainsString([]string{"testUser", "first", "second", "third"}, auth1["username"].(string)) {
			t.Errorf("invalid username returned, not found in test tenant data.")
		}
	}

	if !config.IsVaultOn() {
		// For every returned authentication
		for _, authOut := range out.Data {
			authOutId, err := strconv.ParseInt(authOut.(map[string]interface{})["id"].(string), 10, 64)
			if err != nil {
				t.Error(err)
			}
			// find auth in fixtures and check the tenant id
			for _, authFixtures := range fixtures.TestAuthenticationData {
				if authOutId == authFixtures.DbID {
					if authFixtures.TenantID != tenantId {
						t.Errorf("expected tenant id = %d, got %d", tenantId, authFixtures.TenantID)
					}
				}
			}
		}

	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
	conf.SecretStore = originalSecretStore
}

// TestSourceListAuthenticationsEmptyList tests that empty list is returned
// when the source doesn't have authentications
func TestSourceListAuthenticationsEmptyList(t *testing.T) {
	tenantId := int64(1)
	sourceId := int64(101)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	err := SourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestSourceListAuthenticationsTenantNotExists tests that not found is returned
// for not existing tenant
func TestSourceListAuthenticationsTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := notExistingTenantId
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := notFoundSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestSourceListAuthenticationsInvalidTenant tests that not found is returned for
// valid tenant and source that not belongs to this tenant
func TestSourceListAuthenticationsInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	sourceId := int64(2)
	tenantId := int64(2)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := notFoundSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceListAuthenticationsNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/30983098439/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("30983098439")

	notFoundSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := notFoundSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceListAuthenticationsBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("xxx")

	badRequestSourceListAuthentications := ErrorHandlingContext(SourceListAuthentications)
	err := badRequestSourceListAuthentications(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceTypeSourceSubcollectionList(t *testing.T) {
	sourceTypeId := int64(1)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

	err := SourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	// How many sources with given source type is in fixtures
	// (adding new fixtures will not affect the test)
	var wantSourcesCount int
	for _, i := range fixtures.TestSourceData {
		if i.SourceTypeID == sourceTypeId && i.TenantID == tenantId {
			wantSourcesCount++
		}
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		_, ok := src.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}
	}

	err = checkAllSourcesBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestSourceTypeSourceSubcollectionListTenantNotExists tests that empty list
// is returned for existing source type and not existing tenant
func TestSourceTypeSourceSubcollectionListTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// Check existing source type with not existing tenant id
	// Expected is empty list
	sourceTypeId := int64(1)
	tenantId := notExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

	err := SourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestSourceTypeSourceSubcollectionListEmptySubcollection tests that empty list
// is returned for existing source type without existing sources in db for given tenant
func TestSourceTypeSourceSubcollectionListEmptySubcollection(t *testing.T) {
	sourceTypeId := int64(100)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceTypeId))

	err := SourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestSourceTypeSourceSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/80398409384/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues("80398409384")

	notFoundSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := notFoundSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceTypeSourceSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/xxx/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues("xxx")

	badRequestSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := badRequestSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceTypeSourceSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/source_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_type_id")
	c.SetParamValues("1")

	badRequestSourceTypeListSource := ErrorHandlingContext(SourceTypeListSource)
	err := badRequestSourceTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationTypeListSourceSubcollectionList(t *testing.T) {
	appTypeId := int64(1)
	tenantId := int64(1)
	wantSourcesCount := len(testutils.GetSourcesWithAppType(appTypeId))

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues(fmt.Sprintf("%d", appTypeId))

	err := ApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	for _, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		if s["id"] == 1 && s["name"] != "Source1" {
			t.Error("ghosts infected the return")
		}
	}

	err = checkAllSourcesBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestApplicationTypeListSourceSubcollectionListTenantNotExists tests that empty list
// is returned for existing application type and not existing tenant
func TestApplicationTypeListSourceSubcollectionListTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// Check existing application type with not existing tenant id
	appTypeId := int64(1)
	tenantId := notExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues(fmt.Sprintf("%d", appTypeId))

	err := ApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestApplicationTypeListSourceSubcollectionListEmptySubcollection tests that empty list
// is returned for existing application type without existing sources for given tenant
func TestApplicationTypeListSourceSubcollectionListEmptySubcollection(t *testing.T) {
	appTypeId := int64(100)
	tenantId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues(fmt.Sprintf("%d", appTypeId))

	err := ApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestApplicationTypeListSourceSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/398748974/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("398748974")

	notFoundApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := notFoundApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationTypeListSourceSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/xxx/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("xxx")

	badRequestApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := badRequestApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationTypeListSourceSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_types/1/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("application_type_id")
	c.SetParamValues("1")

	badRequestApplicationTypeListSource := ErrorHandlingContext(ApplicationTypeListSource)
	err := badRequestApplicationTypeListSource(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceList(t *testing.T) {
	tenantId := int64(1)
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		})

	err := SourceList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	var wantSourcesCount int
	for _, s := range fixtures.TestSourceData {
		if s.TenantID == tenantId {
			wantSourcesCount++
		}
	}

	if len(out.Data) != wantSourcesCount {
		t.Error("not enough objects passed back from DB")
	}

	SortByStringValueOnKey("name", out.Data)

	s1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if s1["name"] != "Source1" {
		t.Error("ghosts infected the return")
	}

	s2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if s2["name"] != "Source2" {
		t.Error("ghosts infected the return")
	}

	err = checkAllSourcesBelongToTenant(tenantId, out.Data)
	if err != nil {
		t.Error(err)
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

// TestSourceListTenantNotExists tests that empty list is returned for not existing tenant
func TestSourceListTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// For not existing tenant is expected that returned value
	// will be empty list and return code 200
	tenantId := notExistingTenantId

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		})

	err := SourceList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

// TestSourceListTenantWithoutSources tests that empty list is returned for existing tenant
// without related sources
func TestSourceListTenantWithoutSources(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	// For tenant without sources is expected that returned value
	// will be empty list and return code 200
	tenantId := int64(3)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": tenantId,
		})

	err := SourceList(c)
	if err != nil {
		t.Error(err)
	}

	templates.EmptySubcollectionListTest(t, c, rec)
}

func TestSourceListSatellite(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
			// this gets set during the parse middleware
			"cert-auth": true,
		})

	err := SourceList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if len(out.Data) != 0 {
		t.Error("Objects were not filtered out of request")
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		})
	badRequestSourceList := ErrorHandlingContext(SourceList)
	err := badRequestSourceList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceGet(t *testing.T) {
	tenantId := int64(1)
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	err := SourceGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outSrc m.SourceResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outSrc)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if *outSrc.Name != "Source1" {
		t.Error("ghosts infected the return")
	}

	if outSrc.ID != fmt.Sprintf("%d", sourceId) {
		t.Errorf("source with wrong ID returned, expected %d, got %s", sourceId, outSrc.ID)
	}

	// Convert ID from returned source into int64
	outSrcId, err := strconv.ParseInt(outSrc.ID, 10, 64)
	if err != nil {
		t.Error(err)
	}

	// Check in fixtures that returned source belongs to the desired tenant
	for _, src := range fixtures.TestSourceData {
		if src.ID == outSrcId {
			if src.TenantID != tenantId {
				t.Errorf("wrong tenant id, expected %d, got %d", tenantId, src.TenantID)
			}
			break
		}
	}
}

// TestSourceGetInvalidTenant tests that not found is returned for
// existing source id but with tenant that is now owner of this source
func TestSourceGetInvalidTenant(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := int64(3)
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err := notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

// TestSourceGetTenantNotExists tests that not found is returned for
// not existing tenant
func TestSourceGetTenantNotExists(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	tenantId := notExistingTenantId
	sourceId := int64(1)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1",
		nil,
		map[string]interface{}{
			"tenantID": tenantId,
		},
	)

	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", sourceId))

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err := notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/9872034520975",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9872034520975")

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err := notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestSourceGet := ErrorHandlingContext(SourceGet)
	err := badRequestSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestSourceCreateBadRequest tests that the handler responds with an 400 when an invalid JSON is received
func TestSourceCreateBadRequest(t *testing.T) {
	emptyName := ""
	requestBody := m.SourceCreateRequest{
		Name: &emptyName,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestSourceCreate := ErrorHandlingContext(SourceCreate)
	err = badRequestSourceCreate(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestSourceCreate tests that a 201 is received when a proper JSON message is received
func TestSourceCreate(t *testing.T) {
	// Test with a proper JSON
	name := "TestRequest"
	uid := "5"
	version := "10.5"
	imported := "true"
	sourceRef := "Source reference #5"
	var sourceTypeId int64 = 1

	requestBody := m.SourceCreateRequest{
		Name:                &name,
		Uid:                 &uid,
		Version:             &version,
		Imported:            &imported,
		SourceRef:           &sourceRef,
		AppCreationWorkflow: m.AccountAuth,
		AvailabilityStatus:  m.Available,
		SourceTypeIDRaw:     &sourceTypeId,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err = SourceCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Did not return 201. Body: %s", rec.Body.String())
	}

	src := m.SourceResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &src)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if src.SourceTypeId != "1" {
		t.Errorf("Wrong source ID, wanted %v got %v", "1", src.SourceTypeId)
	}

	id, _ := strconv.ParseInt(src.ID, 10, 64)
	dao, _ := getSourceDao(c)
	_, err = dao.Delete(&id)
	if err != nil {
		t.Errorf("Failed to delete source id %v", id)
	}
}

func TestSourceEdit(t *testing.T) {
	backupNotificationProducer := service.NotificationProducer
	service.NotificationProducer = &mocks.MockAvailabilityStatusNotificationProducer{}

	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               util.StringRef(newSourceName),
		AvailabilityStatus: util.StringRef("unavailable"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

	sourceEditHandlerWithNotifier := middleware.Notifier(SourceEdit)
	err := sourceEditHandlerWithNotifier(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusOK, rec.Code)
	}

	src := m.SourceResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &src)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if *src.Name != newSourceName {
		t.Errorf("Unexpected source name: expected '%s', got '%s'", newSourceName, *src.Name)
	}

	if *src.AvailabilityStatus != "unavailable" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", *src.AvailabilityStatus)
	}

	notificationProducer, ok := service.NotificationProducer.(*mocks.MockAvailabilityStatusNotificationProducer)
	if !ok {
		t.Errorf("unable to cast notification producer")
	}

	emailNotificationInfo := &m.EmailNotificationInfo{ResourceDisplayName: "Source",
		CurrentAvailabilityStatus:  "unavailable",
		PreviousAvailabilityStatus: "available",
		SourceName:                 newSourceName,
		SourceID:                   strconv.FormatInt(fixtures.TestSourceData[0].ID, 10),
		TenantID:                   strconv.FormatInt(fixtures.TestSourceData[0].TenantID, 10),
	}

	if !cmp.Equal(emailNotificationInfo, notificationProducer.EmailNotificationInfo) {
		t.Errorf("Invalid email notification data:")
		t.Errorf("Expected: %v Obtained: %v", emailNotificationInfo, notificationProducer.EmailNotificationInfo)
	}

	service.NotificationProducer = backupNotificationProducer
}

func TestSourceEditNotFound(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               util.StringRef(newSourceName),
		AvailabilityStatus: util.StringRef("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/8937498374",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("8937498374")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundSourceEdit := ErrorHandlingContext(SourceEdit)
	err := notFoundSourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceEditBadRequest(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               util.StringRef(newSourceName),
		AvailabilityStatus: util.StringRef("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/xxx",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestSourceEdit := ErrorHandlingContext(SourceEdit)
	err := badRequestSourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourceDelete(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/100",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("100")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := SourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusNoContent, rec.Code)
	}

	// Check that source doesn't exist.
	c, rec = request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/100",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("100")

	notFoundSourceGet := ErrorHandlingContext(SourceGet)
	err = notFoundSourceGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/9038049384",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9038049384")

	notFoundSourceDelete := ErrorHandlingContext(SourceDelete)
	err := notFoundSourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourceDeleteBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/sources/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestSourceDelete := ErrorHandlingContext(SourceDelete)
	err := badRequestSourceDelete(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestAvailabilityStatusCheck(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 202 {
		t.Errorf("Wrong code, got %v, expected %v", rec.Code, 202)
	}
}

func TestAvailabilityStatusCheckNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/183209745/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("183209745")

	notFoundSourceCheckAvailability := ErrorHandlingContext(SourceCheckAvailability)
	err := notFoundSourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestAvailabilityStatusCheckBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/xxx/check_availability",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("xxx")

	badRequestSourceCheckAvailability := ErrorHandlingContext(SourceCheckAvailability)
	err := badRequestSourceCheckAvailability(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTest(t *testing.T) {
	sourceId := "1"

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/"+sourceId+"/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(sourceId)

	err := SourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("want %d, got %d", http.StatusOK, rec.Code)
	}

	var out util.Collection
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshalling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if parser.RunningIntegrationTests {
		if len(out.Data) != 2 {
			t.Error("not enough objects passed back from DB")
		}
	} else {
		if len(fixtures.TestRhcConnectionData) != len(out.Data) {
			t.Error("not enough objects passed back from DB")
		}
	}

	for _, source := range out.Data {
		_, ok := source.(map[string]interface{})

		if !ok {
			t.Error("model did not deserialize as a source")
		}
	}
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/0394830498/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("0394830498")

	notFoundSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := notFoundSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestInvalidSyntax(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/xxx/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("xxx")

	badRequestSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := badRequestSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestSourcesGetRelatedRhcConnectionsTestBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/rhc_connections",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	badRequestSourcesRhcConnectionList := ErrorHandlingContext(SourcesRhcConnectionList)
	err := badRequestSourcesRhcConnectionList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

// TestPauseSourceAndItsApplications tests that the "pause source" endpoint sets all the applications and the source
// itself as paused, by modifying their "paused_at" column.
func TestPauseSourceAndItsApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/pause",
		nil,
		map[string]interface{}{
			"tenantID":      int64(1),
			"x-rh-identity": util.GeneratedXRhIdentity("1234", "1234"),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourcePause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// TestResumeSourceAndItsApplications tests that the "unpause source" endpoint sets all the applications and the source
// itself as resumed, by setting their "paused_at" column as "NULL".
func TestResumeSourceAndItsApplications(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID":      int64(1),
			"x-rh-identity": util.GeneratedXRhIdentity("1234", "1234"),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	err := SourceUnpause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// MockSender is just a mock which will allow us to control how the "RaiseEvent" function gets executed.
type MockSender struct {
}

// raiseEventFunc is an overrideable function which gets executed when the sender's "RaiseEvent" is called. This helps
// keeping the test logic inside each test.
var raiseEventFunc func(eventType string, payload []byte, headers []kafka.Header) error

// RaiseEvent is a placeholder function which simulates a call to the "RaiseEvent" function.
func (p MockSender) RaiseEvent(eventType string, payload []byte, headers []kafka.Header) error {
	return raiseEventFunc(eventType, payload, headers)
}

// TestSourcePauseRaiseEventCheck tests that a proper "raise event" is raised when a source is paused.
func TestSourcePauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID":      int64(1),
			"x-rh-identity": util.GeneratedXRhIdentity("1234", "1234"),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: MockSender{}} }

	var sourceRaiseEventCallCount int
	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		var err error

		switch eventType {
		case "Source.pause":
			err = sourceEventTestHelper(t, c, "Source.pause", payload, headers)

			sourceRaiseEventCallCount++
		case "Application.pause":
			err = applicationEventTestHelper(t, c, "Application.pause", eventType, payload, headers)

			applicationRaiseEventCallCount++
		default:
			t.Errorf(`incorrect event type when raising the event. Want "Source.pause" or "Application.pause", got "%s"`, eventType)
			err = errors.New(`incorrect event type raised`)
		}

		return err
	}

	err := SourcePause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are pausing a single source, therefore there should only have been 1 call to the "RaiseEvent" function.
		want := 1
		got := sourceRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the source event. Want "%d", got "%d"`, want, got)
		}
	}

	{
		// The source has 2 related application in the fixtures, so the "RaiseEvent" function should have been called
		// twice.
		want := 2
		got := applicationRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// TestSourceUnpauseRaiseEventCheck tests that a proper "raise event" is raised when a source is unpaused.
func TestSourceUnpauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/sources/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID":      int64(1),
			"x-rh-identity": util.GeneratedXRhIdentity("1234", "1234"),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: MockSender{}} }

	var sourceRaiseEventCallCount int
	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		var err error

		switch eventType {
		case "Source.unpause":
			err = sourceEventTestHelper(t, c, "Source.unpause", payload, headers)

			sourceRaiseEventCallCount++
		case "Application.unpause":
			err = applicationEventTestHelper(t, c, "Application.unpause", eventType, payload, headers)

			applicationRaiseEventCallCount++
		default:
			t.Errorf(`incorrect event type when raising the event. Want "Source.pause" or "Application.pause", got "%s"`, eventType)
			err = errors.New(`incorrect event type raised`)
		}

		return err
	}

	err := SourceUnpause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are resuming a single source, therefore there should only have been 1 call to the "RaiseEvent" function.
		want := 1
		got := sourceRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the source event. Want "%d", got "%d"`, want, got)
		}
	}

	{
		// The source has 2 related application in the fixtures, so the "RaiseEvent" function should have been called
		// twice.
		want := 2
		got := applicationRaiseEventCallCount
		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// sourceEventTestHelper helps to test whether the received payload and the headers from the "RaiseEvent" function
// correspond to what we are expecting.
func sourceEventTestHelper(t *testing.T, c echo.Context, expectedEventType string, payload []byte, headers []kafka.Header) error {
	sourceDao, err := getSourceDao(c)
	if err != nil {
		t.Errorf(`could not get the source dao: %s`, err)
		return err
	}

	// Grab the source from the fixtures.
	expectedSource, err := sourceDao.GetById(&fixtures.TestSourceData[0].ID)
	if err != nil {
		t.Errorf(`could not fetch the source: %s`, err)
		return err
	}

	{
		// Turn the source into JSON.
		want, err := json.Marshal(expectedSource.ToEvent())
		if err != nil {
			t.Errorf(`error marshalling the event: %s`, err)
			return err
		}

		got := payload
		if !bytes.Equal(want, got) {
			t.Errorf(`incorrect payload received on raise event.Want "%s", got "%s"`, want, got)
			return err
		}
	}

	{
		var h kafka.Header
		for _, header := range headers {
			if header.Key == "event_type" {
				h = header
				break
			}
		}
		// The header should contain the expected event type as well.
		want := expectedEventType
		got := string(h.Value)
		if want != got {
			t.Errorf(`incorrect header on raise event. Want "%s", got "%s"`, want, got)
			return errors.New(`incorrect header on raise event`)
		}
	}

	return nil
}

// TestSourceEditPausedIntegration tests that a "bad request" response is returned when a paused source is tried to be
// updated when the payload has not allowed fields. Sets the first application of the fixtures as paused and then it
// unpauses it back once the test is finished.
func TestSourceEditPausedIntegration(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               util.StringRef(newSourceName),
		AvailabilityStatus: util.StringRef("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	// Make sure we are using the "NoUnknownFieldsBinder".
	backupBinder := c.Echo().Binder
	c.Echo().Binder = &NoUnknownFieldsBinder{}

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	// Modify the source so that the underlying code identifies it as "paused".
	err := dao.DB.Model(m.Source{}).Where("id = ?", 1).UpdateColumn("paused_at", time.Now()).Error
	if err != nil {
		t.Error(err)
	}

	badRequestSourceEdit := ErrorHandlingContext(SourceEdit)
	err = badRequestSourceEdit(c)
	if err != nil {
		t.Error(err)
	}

	// Revert the changes so other tests don't have any problems.
	err = dao.DB.Model(m.Source{}).Where("id = ?", 1).UpdateColumn("paused_at", nil).Error
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusBadRequest, rec.Code)
	}

	want := "name"
	got := rec.Body.String()
	if !strings.Contains(got, want) {
		t.Errorf(`unexpected body returned. Want "%s" contained in what we got "%s"`, want, got)
	}

	// Restore the binder to not affect any other tests.
	c.Echo().Binder = backupBinder
}

// TestSourceEditPausedUnit tests that a "bad request" response is returned when a paused source is tried to be updated
// when the payload has not allowed fields. Runs on unit tests by swapping the mock source's DAO to one that simulates
// that the endpoints are paused.
func TestSourceEditPausedUnit(t *testing.T) {
	newSourceName := "New source name"
	req := m.SourceEditRequest{
		Name:               util.StringRef(newSourceName),
		AvailabilityStatus: util.StringRef("available"),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/sources/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	// Get the specific SourceDao mock which simulates that the sources are paused.
	backupDao := getSourceDao
	getSourceDao = func(c echo.Context) (dao.SourceDao, error) {
		return &dao.MockSourceDao{Sources: fixtures.TestSourceData}, nil
	}

	// Set the fixture source as "paused".
	pausedAt := time.Now()
	fixtures.TestSourceData[0].PausedAt = &pausedAt

	// Make sure we don't accept the "Name" field we set up above.
	backupBinder := c.Echo().Binder
	c.Echo().Binder = &NoUnknownFieldsBinder{}

	badRequestSourceEdit := ErrorHandlingContext(SourceEdit)
	err := badRequestSourceEdit(c)

	// Revert the fixture endpoint to its default value.
	fixtures.TestSourceData[0].PausedAt = nil
	// Go back to the previous DAO mock.
	getSourceDao = backupDao

	if err != nil {
		t.Error(err)
	}

	got, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Errorf(`error reading the response: %s`, err)
	}

	want := []byte("name")
	if !bytes.Contains(got, want) {
		t.Errorf(`unexpected error received. Want "%s", got "%s"`, want, err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Wrong return code, expected %v got %v", http.StatusBadRequest, rec.Code)
	}

	// Restore the binder to not affect any other tests.
	c.Echo().Binder = backupBinder
}

// HELPERS:

// checkAllSourcesBelongToTenant checks that all returned sources belongs to given tenant
func checkAllSourcesBelongToTenant(tenantId int64, sources []interface{}) error {
	// For every returned source
	for _, srcOut := range sources {
		srcOutId, err := strconv.ParseInt(srcOut.(map[string]interface{})["id"].(string), 10, 64)
		if err != nil {
			return err
		}
		// find source in fixtures and check the tenant id
		for _, src := range fixtures.TestSourceData {
			if srcOutId == src.ID {
				if src.TenantID != tenantId {
					return fmt.Errorf("expected tenant id = %d, got %d", tenantId, src.TenantID)
				}
			}
		}
	}
	return nil
}
