package templates

import (
	"encoding/json"
	"fmt"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/helpers"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

var (
	TestDataForOffsetLimitTest = []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100}}
)

func WithOffsetAndLimitTest(t *testing.T, path string, rec *httptest.ResponseRecorder, count int, limit int, offset int) {
	if rec.Code != http.StatusOK {
		t.Error("Did not return 200")
	}

	var out util.Collection
	err := json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != limit {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != offset {
		t.Error("offset not set correctly")
	}

	if out.Meta.Count != count {
		t.Errorf("count not set correctly")
	}

	// Check if count of returned objects is equal to test data
	// taking into account offset and limit.
	gotObjects := len(out.Data)
	wantObjects := count - offset
	if wantObjects < 0 {
		wantObjects = 0
	}

	if wantObjects > limit {
		wantObjects = limit
	}
	if gotObjects != wantObjects {
		t.Errorf("objects passed back from DB: wantObjects'%v', got '%v'", wantObjects, gotObjects)
	}

	helpers.AssertLinks(t, path, out.Links, limit, offset)
}

func NotFoundTest(t *testing.T, rec *httptest.ResponseRecorder) {
	if rec.Code != 404 {
		t.Error(fmt.Sprintf("Wrong return code: expected 404, got %d", rec.Code))
	}

	var out util.ErrorDocument
	err := json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if len(out.Errors) == 0 {
		t.Error("Error message is empty")
	}

	for _, src := range out.Errors {
		if !strings.HasSuffix(src.Detail, "not found") {
			t.Error(fmt.Sprintf("Wrong error message: expected suffix 'not found' in '%s'", src.Detail))
		}
		if src.Status != "404" {
			t.Error(fmt.Sprintf("Wrong error status: expected 404, got %s", src.Status))
		}
	}
}

func BadRequestTest(t *testing.T, rec *httptest.ResponseRecorder) {
	if rec.Code != 400 {
		t.Error(fmt.Sprintf("Wrong return code: expected 400, got %d", rec.Code))
	}

	var out util.ErrorDocument
	err := json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if len(out.Errors) == 0 {
		t.Error("Error message is empty")
	}

	for _, src := range out.Errors {
		if !strings.HasPrefix(src.Detail, "bad request") {
			t.Error(fmt.Sprintf("Wrong error message: expected prefix 'bad request' in '%s'", src.Detail))
		}
		if src.Status != "400" {
			t.Error(fmt.Sprintf("Wrong error status: expected 400, got %s", src.Status))
		}
	}
}
