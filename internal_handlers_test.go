package main

import (
	"encoding/json"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSourceListInternal(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/internal/v2.0/sources",
		nil,
		map[string]interface{}{
			"limit":   100,
			"offset":  0,
			"filters": []util.Filter{},
		})

	err := InternalSourceList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
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

	if len(out.Data) != len(fixtures.TestSourceData) {
		t.Error("not enough objects passed back from DB")
	}

	for i, src := range out.Data {
		s, ok := src.(map[string]interface{})
		if !ok {
			t.Error("model did not deserialize as a source")
		}

		// Parse the source
		responseSourceId, err := util.InterfaceToInt64(s["id"])
		if err != nil {
			t.Errorf("could not parse id from response: %s", err)
		}

		responseExternalTenant := s["tenant"].(string)
		responseAvailabilityStatus := s["availability_status"].(string)

		// Check that the expected source data and the received data are the same
		if want := fixtures.TestSourceData[i].ID; want != responseSourceId {
			t.Errorf("Ids don't match. Want %d, got %d", want, responseSourceId)
		}

		if want := fixtures.TestTenantData[0].ExternalTenant; want != responseExternalTenant {
			t.Errorf("Tenants don't match. Want %#v, got %#v", want, responseExternalTenant)
		}

		if want := fixtures.TestSourceData[i].AvailabilityStatus; want != responseAvailabilityStatus {
			t.Errorf("Availability statuses don't match. Want %s, got %s", want, responseAvailabilityStatus)
		}
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceListInternalWithOffsetAndLimit(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	testData := []map[string]int{
		{"limit": 10, "offset": 0},
		{"limit": 10, "offset": 1},
		{"limit": 10, "offset": 100},
		{"limit": 1, "offset": 0},
		{"limit": 1, "offset": 1},
		{"limit": 1, "offset": 100},
	}

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/internal/v2.0/sources",
			nil,
			map[string]interface{}{
				"limit":   i["limit"],
				"offset":  i["offset"],
				"filters": []util.Filter{},
			})

		err := InternalSourceList(c)
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

		if out.Meta.Limit != i["limit"] {
			t.Error("limit not set correctly")
		}

		if out.Meta.Offset != i["offset"] {
			t.Error("offset not set correctly")
		}

		if out.Meta.Count != len(fixtures.TestSourceData) {
			t.Errorf("count not set correctly")
		}

		// Check if count of returned objects is equal to test data
		// taking into account offset and limit.
		got := len(out.Data)
		want := len(fixtures.TestSourceData) - i["offset"]
		if want < 0 {
			want = 0
		}

		if want > i["limit"] {
			want = i["limit"]
		}
		if got != want {
			t.Errorf("objects passed back from DB: want'%v', got '%v'", want, got)
		}

		AssertLinks(t, c.Request().RequestURI, out.Links, i["limit"], i["offset"])
	}
}

func TestSourceListInternalBadRequestInvalidFilter(t *testing.T) {
	templates.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/internal/v2.0/sources",
		nil,
		map[string]interface{}{
			"limit":  100,
			"offset": 0,
			"filters": []util.Filter{
				{Name: "wrongName", Value: []string{"wrongValue"}},
			},
		},
	)

	badRequestInternalSourceList := ErrorHandlingContext(InternalSourceList)
	err := badRequestInternalSourceList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}
