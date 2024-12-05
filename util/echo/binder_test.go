package echo

import (
	"bytes"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/util"
)

type TestStruct struct {
	YesIamAField bool `json:"good_field"`
}

func TestGoodPayload(t *testing.T) {
	c, _ := CreateTestContext(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"good_field": true}`),
		nil,
	)
	err := c.Bind(&TestStruct{})
	if err != nil {
		t.Error(err)
	}
}

func TestBadPayload(t *testing.T) {
	c, _ := CreateTestContext(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"good_field": true, "oops": "yes"}`),
		nil,
	)

	err := c.Bind(&TestStruct{})
	if err == nil {
		t.Error("No error was found when there should have been an extra field error")
	}

}

func TestNilBody(t *testing.T) {
	c, _ := CreateTestContext(
		http.MethodPost,
		"/",
		nil,
		nil,
	)

	err := c.Bind(&TestStruct{})

	// Check that returned err is Bad request with "no body" message
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("Expected Bad request err, got %s", err)
	} else if !strings.Contains(err.Error(), "no body") {
		t.Errorf("Expected that err message contains 'no body' but got '%s'", err)
	}

}

func TestNoBody(t *testing.T) {
	c, _ := CreateTestContext(
		http.MethodPost,
		"/",
		http.NoBody,
		nil,
	)
	err := c.Bind(&TestStruct{})

	// Check that returned err is Bad request with "no body" message
	if !errors.As(err, &util.ErrBadRequest{}) {
		t.Errorf("Expected Bad request err, got %s", err)
	} else if !strings.Contains(err.Error(), "no body") {
		t.Errorf("Expected that err message contains 'no body' but got '%s'", err)
	}

}

// TestEmptyJsonBody tests that when a valid, empty JSON object is given, a "bad request" error is returned.
func TestEmptyJsonBody(t *testing.T) {
	testValues := []*bytes.Buffer{
		bytes.NewBufferString("{}"),
		bytes.NewBufferString("{     }"),
		bytes.NewBufferString("[]"),
		bytes.NewBufferString("[     ]"),
		bytes.NewBufferString("null"),
	}

	for _, tv := range testValues {
		c, _ := CreateTestContext(
			http.MethodPost,
			"/",
			tv,
			nil,
		)

		err := c.Bind(&TestStruct{})
		if err == nil {
			t.Error("No error was found when there should have been a no body error")
		}

		if !errors.As(err, &util.ErrBadRequest{}) {
			t.Errorf(`bad request error expected when passing it an empty JSON body, got "%s"`, reflect.TypeOf(err))
		}
	}
}
