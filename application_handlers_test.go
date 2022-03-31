package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/fixtures"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/kafka"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

func TestSourceApplicationSubcollectionList(t *testing.T) {
	sourceID := 1
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/applications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues(fmt.Sprintf("%d", sourceID))

	err := SourceListApplications(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
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

	if len(out.Data) != 2 {
		t.Error("not enough objects passed back from DB")
	}

	SortByStringValueOnKey("id", out.Data)

	var wantData []m.Application
	for _, app := range fixtures.TestApplicationData {
		if app.SourceID == int64(sourceID) {
			wantData = append(wantData, app)
		}
	}

	if len(wantData) != len(out.Data) {
		t.Errorf("not enough objects passed back from DB, want %d, got %d", len(wantData), len(out.Data))
	}

	a1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if a1["id"] != fmt.Sprintf("%d", wantData[0].ID) {
		t.Error("ghosts infected the return")
	}

	a2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if a2["id"] != fmt.Sprintf("%d", wantData[1].ID) {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestSourceApplicationSubcollectionListNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/134793847/applications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("source_id")
	c.SetParamValues("134793847")

	notFoundSourceListApplications := ErrorHandlingContext(SourceListApplications)
	err := notFoundSourceListApplications(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestSourceApplicationSubcollectionListBadRequestInvalidSyntax(t *testing.T) {
	var param = []string{" 1 ", "1s", "1,1", "1,1", "*", " ", "?", "abc"}

	for _, p := range param {
		c, rec := request.CreateTestContext(
			http.MethodGet,
			"/api/sources/v3.1/sources/xxx/applications",
			nil,
			map[string]interface{}{
				"limit":    100,
				"offset":   0,
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		c.SetParamNames("source_id")
		c.SetParamValues(p)

		badRequestSourceListApplications := ErrorHandlingContext(SourceListApplications)
		err := badRequestSourceListApplications(c)
		if err != nil {
			t.Error(err)
		}

		testutils.BadRequestTest(t, rec)
	}
}

func TestSourceApplicationSubcollectionListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/sources/1/applications",
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

	c.SetParamNames("source_id")
	c.SetParamValues("1")

	badRequestSourceListApplications := ErrorHandlingContext(SourceListApplications)
	err := badRequestSourceListApplications(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationList(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	err := ApplicationList(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
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

	if len(out.Data) != len(fixtures.TestApplicationData) {
		t.Error("not enough objects passed back from DB")
	}

	SortByStringValueOnKey("id", out.Data)

	a1, ok := out.Data[0].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if a1["id"] != "1" {
		t.Error("ghosts infected the return")
	}

	if a1["extra"] == nil {
		t.Error("ghosts infected the return")
	}

	a2, ok := out.Data[1].(map[string]interface{})
	if !ok {
		t.Error("model did not deserialize as a source")
	}

	if a2["id"] != "2" {
		t.Error("ghosts infected the return")
	}

	if a2["extra"] == nil {
		t.Error("ghosts infected the return")
	}

	AssertLinks(t, c.Request().RequestURI, out.Links, 100, 0)
}

func TestApplicationListBadRequestInvalidFilter(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	var c, rec = request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications",
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
	badRequestApplicationList := ErrorHandlingContext(ApplicationList)
	err := badRequestApplicationList(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationGet(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications/1",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Error("Did not return 200")
	}

	var outApplication m.ApplicationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &outApplication)
	if err != nil {
		t.Error("Failed unmarshaling output")
	}

	if outApplication.Extra == nil {
		t.Error("ghosts infected the return")
	}
}

func TestApplicationGetNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications/9843762095",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9843762095")

	notFoundApplicationGet := ErrorHandlingContext(ApplicationGet)
	err := notFoundApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationGetBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/applications/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationGet := ErrorHandlingContext(ApplicationGet)
	err := badRequestApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationCreateGood(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		SourceIDRaw:          "2",
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 201 {
		t.Errorf("Wrong return code, expected %v got %v", 201, rec.Code)
	}

	app := m.ApplicationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &app)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if app.SourceID != "2" {
		t.Errorf("Wrong source ID, wanted %v got %v", "2", app.SourceID)
	}

	id, _ := strconv.ParseInt(app.ID, 10, 64)
	dao, _ := getApplicationDao(c)
	_, _ = dao.Delete(&id)
}

func TestApplicationCreateMissingSourceId(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestApplicationCreate := ErrorHandlingContext(ApplicationCreate)
	err := badRequestApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationCreateMissingApplicationTypeId(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: true}

	req := m.ApplicationCreateRequest{
		SourceIDRaw: "1",
		Extra:       nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestApplicationCreate := ErrorHandlingContext(ApplicationCreate)
	err := badRequestApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationCreateIncompatible(t *testing.T) {
	service.AppTypeDao = &dao.MockApplicationTypeDao{Compatible: false}

	req := m.ApplicationCreateRequest{
		SourceIDRaw:          "2",
		ApplicationTypeIDRaw: "1",
		Extra:                nil,
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestApplicationCreate := ErrorHandlingContext(ApplicationCreate)
	err := badRequestApplicationCreate(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationEdit(t *testing.T) {
	req := m.ApplicationEditRequest{
		Extra:                   map[string]interface{}{"thing": true},
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/applications/1",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	err := ApplicationEdit(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != 200 {
		t.Errorf("Wrong return code, expected %v got %v", 200, rec.Code)
	}

	app := m.ApplicationResponse{}
	raw, _ := io.ReadAll(rec.Body)
	err = json.Unmarshal(raw, &app)
	if err != nil {
		t.Errorf("Failed to unmarshal application from response: %v", err)
	}

	if *app.AvailabilityStatus != "available" {
		t.Errorf("Wrong availability status, wanted %v got %v", "available", *app.AvailabilityStatus)
	}
}

func TestApplicationEditNotFound(t *testing.T) {
	req := m.ApplicationEditRequest{
		Extra:                   map[string]interface{}{"thing": true},
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/applications/9764567834",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9764567834")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	notFoundApplicationEdit := ErrorHandlingContext(ApplicationEdit)
	err := notFoundApplicationEdit(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationEditBadRequest(t *testing.T) {
	req := m.ApplicationEditRequest{
		Extra:                   map[string]interface{}{"thing": true},
		AvailabilityStatus:      request.PointerToString("available"),
		AvailabilityStatusError: request.PointerToString(""),
	}

	body, _ := json.Marshal(req)

	c, rec := request.CreateTestContext(
		http.MethodPatch,
		"/api/sources/v3.1/applications/xxx",
		bytes.NewReader(body),
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")
	c.Request().Header.Add("Content-Type", "application/json;charset=utf-8")

	badRequestApplicationEdit := ErrorHandlingContext(ApplicationEdit)
	err := badRequestApplicationEdit(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

func TestApplicationDeleteNotFound(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/9843762095",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("9843762095")

	notFoundApplicationGet := ErrorHandlingContext(ApplicationDelete)
	err := notFoundApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.NotFoundTest(t, rec)
}

func TestApplicationDeleteBadRequest(t *testing.T) {
	c, rec := request.CreateTestContext(
		http.MethodDelete,
		"/api/sources/v3.1/applications/xxx",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("xxx")

	badRequestApplicationGet := ErrorHandlingContext(ApplicationDelete)
	err := badRequestApplicationGet(c)
	if err != nil {
		t.Error(err)
	}

	testutils.BadRequestTest(t, rec)
}

// TestPauseApplication tests that an application gets successfully paused.
func TestPauseApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications/1/pause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ApplicationPause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// TestResumeApplication tests that an application gets successfully resumed.
func TestResumeApplication(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications/1/unpause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	err := ApplicationUnpause(c)
	if err != nil {
		t.Error(err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}
}

// TestPauseApplicationPauseRaiseEventCheck tests that a proper "raise event" is raised when a source is paused.
func TestPauseApplicationPauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications/1/pause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = events.EventStreamProducer{Sender: MockSender{}}

	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		err := applicationEventTestHelper(t, c, "Application.pause", eventType, payload, headers)

		applicationRaiseEventCallCount++
		return err
	}

	err := ApplicationPause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are pausing a single application, therefore the "RaiseEvent" function should have been called once.
		want := 1
		got := applicationRaiseEventCallCount

		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// TestPauseApplicationPauseRaiseEventCheck tests that a proper "raise event" is raised when a source is unpaused.
func TestPauseApplicationUnpauseRaiseEventCheck(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)

	c, rec := request.CreateTestContext(
		http.MethodPost,
		"/api/sources/v3.1/applications/1/pause",
		nil,
		map[string]interface{}{
			"tenantID": int64(1),
		},
	)

	c.SetParamNames("id")
	c.SetParamValues("1")

	// We back up the producer so that we can restore it once the test has finished. This way we don't mess up with
	// other tests that may need to raise events.
	backupProducer := service.Producer
	service.Producer = events.EventStreamProducer{Sender: MockSender{}}

	var applicationRaiseEventCallCount int
	raiseEventFunc = func(eventType string, payload []byte, headers []kafka.Header) error {
		// Set up an error which will get returned. Probably will get overwritten if there are multiple errors, but
		// we don't mind since we are logging every failure. Essentially, it just to satisfy the function signature.
		err := applicationEventTestHelper(t, c, "Application.unpause", eventType, payload, headers)

		applicationRaiseEventCallCount++
		return err
	}

	err := ApplicationUnpause(c)
	if err != nil {
		t.Error(err)
	}

	{
		// We are resuming a single application, therefore the "RaiseEvent" function should have been called once.
		want := 1
		got := applicationRaiseEventCallCount

		if want != got {
			t.Errorf(`raise event was called an incorrect number of times for the application event. Want "%d", got "%d"`, want, got)
		}
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf(`want status "%d", got "%d"`, http.StatusNoContent, rec.Code)
	}

	// Restore the producer back to the original.
	service.Producer = backupProducer
}

// applicationEventTestHelper helps to test whether the received payload and the headers from the "RaiseEvent" function
// correspond to what we are expecting.
func applicationEventTestHelper(t *testing.T, c echo.Context, expectedEventType string, eventType string, payload []byte, headers []kafka.Header) error {
	if expectedEventType != eventType {
		t.Errorf(`incorrect event type when raising the event. Want "%s", got "%s"`, expectedEventType, eventType)
		return errors.New(`incorrect event type raised`)
	}

	// We have to unmarshal the payload to grab the ID and fetch the fixture from the database.
	var content map[string]interface{}
	err := json.Unmarshal(payload, &content)
	if err != nil {
		t.Errorf(`could not unmarshal the payload: %s`, err)
		return err
	}

	// For some reason the ID comes as a float64.
	tmpId, ok := content["id"].(float64)
	if !ok {
		t.Errorf(`incorrect type for the id. Want "string", got "%s"`, reflect.TypeOf(content["id"]))
		return errors.New(`incorrect type for the id`)
	}

	id := int64(tmpId)

	appDao, err := getApplicationDao(c)
	if err != nil {
		t.Errorf(`could not get the application DAO: %s`, err)
		return err
	}

	app, err := appDao.GetById(&id)
	if err != nil {
		t.Errorf(`error fetching the application: %s`, err)
		return err
	}

	{
		// Turn the application into JSON.
		want, err := json.Marshal(app.ToEvent())
		if err != nil {
			t.Errorf(`error marshalling the event: %s`, err)
			return err
		}

		got := payload
		if !bytes.Equal(want, got) {
			t.Errorf(`incorrect payload given to RaiseEvent. Want "%s", got "%s"`, want, got)
			return errors.New(`incorrect payload given to RaiseEvent`)
		}
	}

	{
		// The header should contain the expected event type as well.
		want := expectedEventType
		got := string(headers[0].Value)
		if want != got {
			t.Errorf(`incorrect header on raise event. Want "%s", got "%s"`, want, got)
			return errors.New(`incorrect header on raise event`)
		}
	}

	return nil
}
