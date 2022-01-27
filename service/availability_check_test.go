package service

import (
	"testing"

	m "github.com/RedHatInsights/sources-api-go/model"
)

type dummyChecker struct {
	ApplicationCounter int
	EndpointCounter    int
}

func (c *dummyChecker) ApplicationAvailabilityCheck(source *m.Source) {
	for i := 0; i < len(source.Applications); i++ {
		c.ApplicationCounter++
	}
}

func (c *dummyChecker) EndpointAvailabilityCheck(source *m.Source) {
	for i := 0; i < len(source.Endpoints); i++ {
		c.EndpointCounter++
	}
}

func TestApplicationAvailability(t *testing.T) {
	d := &dummyChecker{}
	ac = d

	RequestAvailabilityCheck(&m.Source{
		// 2 applications on this source.
		Applications: []m.Application{{}, {}},
	})

	if d.ApplicationCounter != 2 {
		t.Errorf("availability check not called for both applications, got %v expected %v", d.ApplicationCounter, 2)
	}
}

func TestEndpointAvailability(t *testing.T) {
	d := &dummyChecker{}
	ac = d

	RequestAvailabilityCheck(&m.Source{
		// 3 endpoints on this source.
		Endpoints: []m.Endpoint{{}, {}, {}},
	})

	if d.EndpointCounter != 3 {
		t.Errorf("availability check not called for all endpoints, got %v expected %v", d.EndpointCounter, 3)
	}
}

func TestBothAvailability(t *testing.T) {
	d := &dummyChecker{}
	ac = d

	RequestAvailabilityCheck(&m.Source{
		// 2 applications on this source.
		Applications: []m.Application{{}, {}, {}},
		// 3 endpoints on this source.
		Endpoints: []m.Endpoint{{}, {}, {}, {}},
	})

	if d.ApplicationCounter != 3 {
		t.Errorf("availability check not called for both applications, got %v expected %v", d.ApplicationCounter, 3)
	}

	if d.EndpointCounter != 4 {
		t.Errorf("availability check not called for all endpoints, got %v expected %v", d.EndpointCounter, 4)
	}
}
