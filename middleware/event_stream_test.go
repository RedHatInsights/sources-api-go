package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/sources-api-go/internal/events"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/mocks"
	"github.com/RedHatInsights/sources-api-go/internal/testutils/request"
	h "github.com/RedHatInsights/sources-api-go/middleware/headers"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/labstack/echo/v4"
)

var raiseMiddleware = RaiseEvent

type fakeEvent struct {
	raised bool
}

type fakeEventEvent struct {
	Raised bool `json:"raised"`
}

func (f *fakeEvent) ToEvent() interface{} {
	return fakeEventEvent{Raised: f.raised}
}

func TestRaiseEvent(t *testing.T) {
	s := mocks.MockSender{}
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &s} }
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("event_type", "Thing.create")
		c.Set("resource", &fakeEvent{raised: true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	// sleep in order for goroutine to run
	time.Sleep(50 * time.Millisecond)

	if s.Hit != 1 {
		t.Errorf("Wrong number of Hits to raise event, got %v expected %v", s.Hit, 1)
	}
}

func TestRaiseEventWithHeaders(t *testing.T) {
	s := mocks.MockSender{}
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &s} }
	c, rec := request.CreateTestContext(http.MethodGet, "/", nil, map[string]interface{}{
		h.PskKey: "1234",
		"x-rh-identity":    util.GeneratedXRhIdentity("1234", "1234"),
	})

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("event_type", "Thing.create")
		c.Set("resource", &fakeEvent{raised: true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	// sleep in order for goroutine to run
	time.Sleep(50 * time.Millisecond)

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	if s.Hit != 1 {
		t.Errorf("Wrong number of Hits to raise event, got %v expected %v", s.Hit, 1)
	}

	if len(s.Headers) != 4 {
		t.Errorf("Headers not set properly from RaiseEvent")
	}

	expected := []string{"event_type", "x-rh-identity", "x-rh-sources-account-number", "x-rh-sources-org-id"}
	for _, h := range s.Headers {
		if !util.SliceContainsString(expected, h.Key) {
			t.Errorf("Got bad header: [%v: %v]", h.Key, h.Value)
		}
	}
}

func TestRaiseEventBody(t *testing.T) {
	s := mocks.MockSender{}
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &s} }
	c, rec := request.EmptyTestContext()

	f := raiseMiddleware(func(c echo.Context) error {
		c.Set("event_type", "Thing.create")
		c.Set("resource", &fakeEvent{raised: true})
		return c.NoContent(http.StatusNoContent)
	})

	err := f(c)
	if err != nil {
		t.Errorf("Got an error when none would have been expected: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("Wrong return code, expected %v got %v", 204, rec.Code)
	}

	// sleep in order for goroutine to run
	time.Sleep(50 * time.Millisecond)

	if s.Hit != 1 {
		t.Errorf("Wrong number of Hits to raise event, got %v expected %v", s.Hit, 1)
	}

	if s.Body != `{"raised":true}` {
		t.Errorf("Raised bad Body %v", s.Body)
	}
}

func TestNoRaiseEvent(t *testing.T) {
	s := mocks.MockSender{}
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &s} }
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

	// sleep in order for goroutine to run
	time.Sleep(50 * time.Millisecond)

	if s.Hit != 0 {
		t.Errorf("Wrong number of Hits to raise event, got %v expected %v", s.Hit, 0)
	}
}

func TestSkipOnContext(t *testing.T) {
	s := mocks.MockSender{}
	service.Producer = func() events.Sender { return events.EventStreamProducer{Sender: &s} }
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

	// sleep in order for goroutine to run
	time.Sleep(50 * time.Millisecond)

	if s.Hit != 0 {
		t.Errorf("Wrong number of Hits to raise event, got %v expected %v", s.Hit, 0)
	}
}
