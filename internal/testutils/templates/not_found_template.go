package templates

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

func NotFoundTest(t *testing.T, rec *httptest.ResponseRecorder) {
	if rec.Code != 404 {
		t.Errorf("Wrong return code: expected 404, got %d", rec.Code)
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
			t.Errorf("Wrong error message: expected suffix 'not found' in '%s'", src.Detail)
		}

		if src.Status != "404" {
			t.Errorf("Wrong error status: expected 404, got %s", src.Status)
		}
	}
}
