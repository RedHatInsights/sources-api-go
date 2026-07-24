package templates

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v5"
)

func EmptySubcollectionListTest(t *testing.T, c echo.Context, rec *httptest.ResponseRecorder) {
	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out util.Collection

	err := json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if out.Meta.Limit != 100 {
		t.Error("limit not set correctly")
	}

	if out.Meta.Offset != 0 {
		t.Error("offset not set correctly")
	}

	if out.Meta.Count != 0 {
		t.Error("count not set correctly")
	}

	if len(out.Data) != 0 {
		t.Errorf("expected 0 objects passed back from DB, got %d", len(out.Data))
	}

	testutils.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}
