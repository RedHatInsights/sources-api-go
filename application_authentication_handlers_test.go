package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/helpers"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/parser"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/templates"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestApplicationAuthenticationList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	err := ApplicationAuthenticationList(c)
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

	if len(out.Data) != len(fixtures.TestApplicationAuthenticationData) {
		t.Error("not enough objects passed back from DB")
	}

	appAuth, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if appAuth["id"] != "1" {
		t.Error("ghosts infected the return")
	}

	if appAuth["application_id"] != "1" {
		t.Error("ghosts infected the return")
	}

	authID := strconv.Itoa(int(fixtures.TestAuthenticationData[0].DbID))
	if appAuth["authentication_id"].(string) != authID {
		t.Error("ghosts infected the return")
	}

	helpers.AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationAuthenticationListBadRequestInvalidFilter(t *testing.T) {
	helpers.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications",
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

	badRequestApplicationAuthenticationList := ErrorHandlingContext(ApplicationAuthenticationList)
	err := badRequestApplicationAuthenticationList(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationListWithOffsetAndLimit(t *testing.T) {
	helpers.SkipIfNotRunningIntegrationTests(t)

	testData := templates.TestDataForOffsetLimitTest
	wantAppAuthCount := len(fixtures.TestApplicationAuthenticationData)

	for _, i := range testData {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/application_authentications",
			nil,
			map[string]interface{}{
				"limit":    i["limit"],
				"offset":   i["offset"],
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		err := ApplicationAuthenticationList(c)
		if err != nil {
			t.Error(err)
		}

		path := c.Request().RequestURI
		templates.WithOffsetAndLimitTest(t, path, rec, wantAppAuthCount, i["limit"], i["offset"])
	}
}

func TestApplicationAuthenticationGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var out m.ApplicationAuthenticationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &out)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}
	authID := strconv.Itoa(int(fixtures.TestAuthenticationData[0].DbID))
	if out.AuthenticationID != authID {
		t.Error("ghosts infected the return")
	}

	if out.ApplicationID != "1" {
		t.Error("ghosts infected the return")
	}
}

func TestApplicationAuthenticationGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/13094830948",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("13094830948")

	notFoundApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := notFoundApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.NotFoundTest(t, rec)
}

func TestApplicationAuthenticationGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/application_authentications/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationGet)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationCreate(t *testing.T) {
	if parser.RunningIntegrationTests {
		t.Skip("Test not supported when using db backend")
	}

	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    7,
		AuthenticationIDRaw: 7,
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	err := ApplicationAuthenticationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 201 {
		t.Errorf("Wrong response code, got %v wanted %v", rec.Code, 201)
	}
}

func TestApplicationAuthenticationCreateBadAppId(t *testing.T) {
	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    "abcd",
		AuthenticationIDRaw: 7,
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationCreate)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationCreateBadAuthId(t *testing.T) {
	input := m.ApplicationAuthenticationCreateRequest{
		ApplicationIDRaw:    7,
		AuthenticationIDRaw: "abcd",
	}

	body, _ := json.Marshal(&input)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/application_authentications",
		bytes.NewBuffer(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.Request().Header.Add("Content-Type", "application/json")

	badRequestApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationCreate)
	err := badRequestApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}

	templates.BadRequestTest(t, rec)
}

func TestApplicationAuthenticationDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/application_authentications/1234523452542",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)
	c.SetParamNames("id")
	c.SetParamValues("1234523452542")

	notFoundApplicationAuthenticationGet := ErrorHandlingContext(ApplicationAuthenticationDelete)
	err := notFoundApplicationAuthenticationGet(c)
	if err != nil {
		t.Error(err)
	}
	templates.NotFoundTest(t, rec)
}
