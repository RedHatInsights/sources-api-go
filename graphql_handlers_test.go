package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
)

const query = `
{"query":"{\n meta {\n  count\n }\n sources(\n  offset: 0\n  limit: 50\n  sort_by: { name: \"created_at\", direction: desc }\n  filter: {\n   name: \"source_type.vendor\"\n   operation: \"not_eq\"\n   value: \"Red Hat\"\n  }\n ) {\n  id\n  created_at\n  app_creation_workflow\n  source_type_id\n  name\n  imported\n  availability_status\n  source_ref\n  last_checked_at\n  updated_at\n  last_available_at\n  app_creation_workflow\n  paused_at\n  authentications {\n   authtype\n   username\n  }\n  applications {\n   application_type_id\n   id\n   availability_status_error\n   availability_status\n   paused_at\n   extra\n   authentications {\n    username\n    authtype\n   }\n  }\n  endpoints {\n   id\n   scheme\n   host\n   port\n   path\n   receptor_node\n   role\n   certificate_authority\n   verify_ssl\n   availability_status_error\n   availability_status\n   authentications {\n    authtype\n    availability_status\n    availability_status_error\n   }\n  }\n }\n}\n"}
`

func TestGraphQL(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/graphql",
		strings.NewReader(query),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Set("Content-Type", "application/json")

	err := GraphQLQuery(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Errorf("Bad response - got %v wanted %v", rec.Code, 200)
	}

	bytes, _ := io.ReadAll(rec.Body)

	var body map[string]interface{}

	err = json.Unmarshal(bytes, &body)
	if err != nil {
		t.Error(err)
	}

	if body["errors"] != nil {
		t.Errorf("errors present: %v", body["errors"])
		t.FailNow()
	}
}
