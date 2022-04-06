package templates

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/helpers"
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
