package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
)

func TestOpenApiReturn(t *testing.T) {
	raw, err := os.ReadFile("public/openapi-3-v3.1.json")
	if err != nil {
		t.Errorf("Failed to read openapi file: %v", err)
	}

	rawFile := make(map[string]interface{})
	err = json.Unmarshal(raw, &rawFile)
	if err != nil {
		t.Errorf("Failed to marshal json: %v", err)
	}

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/openapi.json",
		nil,
		map[string]interface{}{},
	)

	err = PublicOpenApi("v3.1")(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Errorf("Did not return 200")
	}

	output, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Error(err)
	}

	out := make(map[string]interface{})
	err = json.Unmarshal(output, &out)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(out, rawFile) {
		t.Errorf("Endpoint did not return the same file as on disk")
	}
}
