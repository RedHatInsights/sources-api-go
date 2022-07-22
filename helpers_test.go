package main

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

// TestGetFilters tests that the function returns correctly filters from request
func TestGetFilters(t *testing.T) {
	want := []util.Filter{
		{Name: "wrongName", Value: []string{"wrongValue"}},
	}

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  want,
			"tenantID": int64(1),
		},
	)

	got, err := getFilters(c)

	if len(got) != 1 {
		t.Error("ghosts infected the return")
	}

	if got[0].Name != want[0].Name {
		t.Errorf(`incorrect filter name. Want "%s", got "%s`, want[0].Name, got[0].Name)
	}

	if len(got[0].Value) != 1 {
		t.Error("ghosts infected the return")
	}

	if got[0].Value[0] != want[0].Value[0] {
		t.Errorf(`incorrect value name. Want "%s", got "%s`, want[0].Value[0], got[0].Value[0])
	}

	if err != nil {
		t.Errorf(`want nil err, got "%s"`, err)
	}
}

// TestGetFiltersBadFilters tests that the function returns error when filters
// in request has invalid structure
func TestGetFiltersBadFilters(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"filters":  "blabla",
			"tenantID": int64(1),
		},
	)

	got, err := getFilters(c)

	if got != nil {
		t.Error("ghosts infected the return")
	}

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err.Error() != "failed to pull filters from request" {
		t.Errorf("wrong error message")
	}
}

// TestGetFiltersMissingFilters tests that the function returns error when request
// doesn't contain filters
func TestGetFiltersMissingFilters(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   0,
			"tenantID": int64(1),
		},
	)

	got, err := getFilters(c)

	if got != nil {
		t.Error("ghosts infected the return")
	}

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err.Error() != "failed to pull filters from request" {
		t.Errorf("wrong error message")
	}
}

// TestGetLimit tests that the function returns correctly limit from request
func TestGetLimit(t *testing.T) {
	want := 100

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    want,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	got, _, err := getLimitAndOffset(c)

	if got != want {
		t.Errorf(`want "%d" error, got "%d"`, want, got)
	}

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}
}

// TestGetLimitNegativeLimit tests that the function returns correctly negative limit
// from request
func TestGetLimitNegativeLimit(t *testing.T) {
	want := -100

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    want,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	got, _, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestGetLimitBadLimit tests that the function returns error when limit
// is not number
func TestGetLimitBadLimit(t *testing.T) {
	badLimit := "xxx"

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    badLimit,
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	got, _, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestGetLimitMissingLimit tests that the function returns error when limit
// is missing
func TestGetLimitMissingLimit(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"offset":   0,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	got, _, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestGetOffset tests that the function returns correctly offset from request
func TestGetOffset(t *testing.T) {
	want := 10

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   want,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	_, got, err := getLimitAndOffset(c)

	if got != want {
		t.Errorf(`want "%d" error, got "%d"`, want, got)
	}

	if err != nil {
		t.Errorf(`want nil error, got "%s"`, err)
	}
}

// TestGetOffsetNegativeOffset tests that the function returns correctly negative
// offset from request
func TestGetOffsetNegativeOffset(t *testing.T) {
	want := -10

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   want,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	_, got, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestGetOffsetBadOffset tests that the function returns error when offset
// is not number
func TestGetOffsetBadOffset(t *testing.T) {
	want := "xxx"

	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"offset":   want,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	_, got, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestGetOffsetMissingOffset tests that the function returns error when offset
// is missing
func TestGetOffsetMissingOffset(t *testing.T) {
	c, _ := request.CreateTestContext(
		http.MethodGet,
		"/api/sources/v3.1/whatever",
		nil,
		map[string]interface{}{
			"limit":    100,
			"filters":  []util.Filter{},
			"tenantID": int64(1),
		},
	)

	_, got, err := getLimitAndOffset(c)

	if got != 0 {
		t.Errorf(`want 0, got "%d"`, got)
	}

	if !errors.Is(err, util.ErrBadRequestEmpty) {
		t.Error("ghosts infected the return")
	}
}

// TestSetEventStreamResourceForApplication tests if function creates in echo context
// correctly the key "event_type" and if key "resource" is correct type
func TestSetEventStreamResourceForApplication(t *testing.T) {
	testData := map[string]string{
		http.MethodPost:   ".create",
		http.MethodPatch:  ".update",
		http.MethodDelete: ".destroy",
	}

	for httpMethod, event := range testData {
		c, _ := request.CreateTestContext(
			httpMethod,
			"/api/sources/v3.1/whatever",
			nil,
			map[string]interface{}{
				"limit":    100,
				"offset":   0,
				"filters":  []util.Filter{},
				"tenantID": int64(1),
			},
		)

		model := &m.Application{}

		setEventStreamResource(c, model)

		if c.Get("event_type") != "Application"+event {
			t.Error("ghosts infected the return")
		}

		if _, ok := c.Get("resource").(m.Event); !ok {
			t.Errorf("expected ApplicationEvent type, got %v", reflect.TypeOf(c.Get("resource")))
		}
	}
}
