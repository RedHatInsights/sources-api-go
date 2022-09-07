package echo

import (
	"bytes"
	"net/http"
	"testing"
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
	if err == nil {
		t.Error("No error was found when there should have been a no body error")
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
	if err == nil {
		t.Error("No error was found when there should have been a no body error")
	}

}
