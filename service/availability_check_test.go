package service

import (
	"net/url"
	"testing"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type dummyChecker struct {
	ApplicationCounter   int
	EndpointCounter      int
	RhcConnectionCounter int
}

// validate that the dummy checker is implementing the interface properly
var _ = (availabilityChecker)(&dummyChecker{})

func (c dummyChecker) Logger() echo.Logger {
	return logger.EchoLogger{Entry: logger.Log.WithFields(logrus.Fields{})}
}

// send out a http response per application
func (c *dummyChecker) ApplicationAvailabilityCheck(source *m.Source, scheduled bool) {
	for i := 0; i < len(source.Applications); i++ {
		c.httpAvailabilityRequest(source, scheduled, &source.Applications[i], &url.URL{})
	}
}

// send out a satellite kafka message per endpoint
func (c *dummyChecker) EndpointAvailabilityCheck(source *m.Source) {
	for i := 0; i < len(source.Endpoints); i++ {
		c.publishSatelliteMessage(&kafka.Writer{}, source, &source.Endpoints[i])
	}
}

// ping RHC for each RHC Connection
func (c *dummyChecker) RhcConnectionAvailabilityCheck(source *m.Source, headers []kafka.Header) {
	for i := 0; i < len(source.SourceRhcConnections); i++ {
		c.pingRHC(source, &source.SourceRhcConnections[i].RhcConnection, headers)
	}
}

// dummy methods making sure they're getting called.
func (c *dummyChecker) httpAvailabilityRequest(source *m.Source, scheduled bool, app *m.Application, uri *url.URL) {
	c.ApplicationCounter++
}
func (c *dummyChecker) publishSatelliteMessage(writer *kafka.Writer, source *m.Source, endpoint *m.Endpoint) {
	c.EndpointCounter++
}
func (c *dummyChecker) pingRHC(source *m.Source, rhcConnection *m.RhcConnection, headers []kafka.Header) {
	c.RhcConnectionCounter++
}
func (c *dummyChecker) updateRhcStatus(source *m.Source, status string, errstr string, rhcConnection *m.RhcConnection, headers []kafka.Header) {
}

func TestScheduledApplicationAvailability(t *testing.T) {
	var acr = &dummyChecker{}
	acr.ApplicationAvailabilityCheck(&m.Source{
		// 2 applications on this source.
		Applications: []m.Application{{}, {}},
	}, true)

	if acr.ApplicationCounter != 2 {
		t.Errorf("availability check not called for both applications, got %v expected %v", acr.ApplicationCounter, 2)
	}
}

func TestUserApplicationAvailability(t *testing.T) {
	var acr = &dummyChecker{}
	acr.ApplicationAvailabilityCheck(&m.Source{
		// 2 applications on this source.
		Applications: []m.Application{{}, {}},
	}, false)

	if acr.ApplicationCounter != 2 {
		t.Errorf("availability check not called for both applications, got %v expected %v", acr.ApplicationCounter, 2)
	}
}

func TestEndpointAvailability(t *testing.T) {
	var acr = &dummyChecker{}
	acr.EndpointAvailabilityCheck(&m.Source{
		// 3 endpoints on this source.
		Endpoints: []m.Endpoint{{}, {}, {}},
	})

	if acr.EndpointCounter != 3 {
		t.Errorf("availability check not called for all endpoints, got %v expected %v", acr.EndpointCounter, 3)
	}
}

func TestRhcConnectionAvailability(t *testing.T) {
	var acr = &dummyChecker{}
	acr.RhcConnectionAvailabilityCheck(&m.Source{
		// 2 rhc connections!
		SourceRhcConnections: []m.SourceRhcConnection{{RhcConnection: m.RhcConnection{RhcId: "asdf"}}, {RhcConnection: m.RhcConnection{RhcId: "qwerty"}}},
	}, []kafka.Header{})

	if acr.RhcConnectionCounter != 2 {
		t.Errorf("availability check not called for all rhc connections, got %v expected %v", acr.RhcConnectionCounter, 2)
	}
}

func TestAllAvailability(t *testing.T) {
	var acr = &dummyChecker{}
	src := &m.Source{
		// 2 applications on this source.
		Applications: []m.Application{{}, {}, {}},
		// 3 endpoints on this source.
		Endpoints: []m.Endpoint{{}, {}, {}, {}},
		// ...and 1 rhc connection
		SourceRhcConnections: []m.SourceRhcConnection{{RhcConnection: m.RhcConnection{RhcId: "asdf"}}},
	}
	acr.ApplicationAvailabilityCheck(src, true)
	acr.EndpointAvailabilityCheck(src)
	acr.RhcConnectionAvailabilityCheck(src, []kafka.Header{})

	if acr.ApplicationCounter != 3 {
		t.Errorf("availability check not called for both applications, got %v expected %v", acr.ApplicationCounter, 3)
	}

	if acr.EndpointCounter != 4 {
		t.Errorf("availability check not called for all endpoints, got %v expected %v", acr.EndpointCounter, 4)
	}

	if acr.RhcConnectionCounter != 1 {
		t.Errorf("availability check not called for all rhc connections, got %v expected %v", acr.RhcConnectionCounter, 1)
	}
}
