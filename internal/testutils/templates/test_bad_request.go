package templates

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

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
