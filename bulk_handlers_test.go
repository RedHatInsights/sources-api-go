package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func TestBulkCreateMissingSourceType(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	nameSource := "test"
	bulkCreateSource := m.BulkCreateSource{SourceCreateRequest: m.SourceCreateRequest{Name: &nameSource}}
	requestBody := m.BulkCreateRequest{Sources: []m.BulkCreateSource{bulkCreateSource}}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Error("Could not marshal JSON")
	}

	c, _ := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/bulkc_reate",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")
	c.Set("identity", &identity.XRHID{Identity: identity.Identity{AccountNumber: fixtures.TestTenantData[0].ExternalTenant}})

	err = BulkCreate(c)
	if err.Error() != "no source type present, need either [source_type_name] or [source_type_id]" {
		t.Error(err)
	}
}
