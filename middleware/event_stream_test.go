package middleware

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var raiseMiddleware = raiseEvent("Thingy.create")

type mockSender struct {
	hit     int
	headers []kafka.Header
	body    string
}

func (m *mockSender) RaiseEvent(_ string, b []byte, headers []kafka.Header) error {
	m.headers = headers
	m.body = string(b)
	m.hit++
	return nil
}

func TestRaiseEvent(t *testing.T) {
	s := mockSender{}
	producer = events.EventStreamProducer{Sender: &s}
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("resource", map[string]interface{}{"raised": true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.hit != 1 {
		t.Errorf("Wrong number of hits to raise event, got %v expected %v", s.hit, 1)
	}
}

func TestRaiseEventWithHeaders(t *testing.T) {
	s := mockSender{}
	producer = events.EventStreamProducer{Sender: &s}
	c, rec := request.CreateTestContext(http.MethodGet, "/", nil, map[string]interface{}{
		"psk-account":   "1234",
		"x-rh-identity": "asdfasdf",
	})

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("resource", map[string]interface{}{"raised": true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.hit != 1 {
		t.Errorf("Wrong number of hits to raise event, got %v expected %v", s.hit, 1)
	}

	if len(s.headers) != 3 {
		t.Errorf("Headers not set properly from RaiseEvent")
	}

	expected := []string{"event_type", "x-rh-identity", "x-rh-sources-account-number"}
	for _, h := range s.headers {
		if !util.SliceContainsString(expected, h.Key) {
			t.Errorf("Got bad header: [%v: %v]", h.Key, h.Value)
		}
	}
}

func TestRaiseEventBody(t *testing.T) {
	s := mockSender{}
	producer = events.EventStreamProducer{Sender: &s}
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("resource", map[string]interface{}{"raised": true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.hit != 1 {
		t.Errorf("Wrong number of hits to raise event, got %v expected %v", s.hit, 1)
	}

	if s.body != `{"raised":true}` {
		t.Errorf("Raised bad body %v", s.body)
	}
}

func TestNoRaiseEvent(t *testing.T) {
	s := mockSender{}
	producer = events.EventStreamProducer{Sender: &s}
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.hit != 0 {
		t.Errorf("Wrong number of hits to raise event, got %v expected %v", s.hit, 0)
	}
}

func TestSkipOnContext(t *testing.T) {
	s := mockSender{}
	producer = events.EventStreamProducer{Sender: &s}
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("skip_raise", struct{}{})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.hit != 0 {
		t.Errorf("Wrong number of hits to raise event, got %v expected %v", s.hit, 0)
	}
}
