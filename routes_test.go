package main

import (
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestInternalBasepathsRegistered verifies that both the legacy
// (/internal/v{1,2}.0/...) and new standard (/internal/sources/v{1,2}.0/...)
// basepaths are registered for every internal endpoint.  This catches typos
// in basepath strings, missing routes, or accidental route-registration
// regressions.
func TestInternalBasepathsRegistered(t *testing.T) {
	e := echo.New()
	setupRoutes(e, nil, nil)

	// Collect all registered internal routes keyed by basepath style.
	legacyRoutes := make(map[string]bool) // "/internal/v2.0/sources" → true
	newFmtRoutes := make(map[string]bool) // "/internal/sources/v2.0/sources" → true

	for _, route := range e.Routes() {
		if strings.HasPrefix(route.Path, "/internal/sources/") {
			// New format: strip "/internal/sources" prefix to get "/v2.0/..."
			suffix := strings.TrimPrefix(route.Path, "/internal/sources")
			newFmtRoutes[route.Method+" "+suffix] = true
		} else if strings.HasPrefix(route.Path, "/internal/") {
			// Legacy format: strip "/internal" prefix to get "/v2.0/..."
			suffix := strings.TrimPrefix(route.Path, "/internal")
			legacyRoutes[route.Method+" "+suffix] = true
		}
	}

	if len(legacyRoutes) == 0 {
		t.Fatal("no legacy internal routes found — route registration broken")
	}

	if len(newFmtRoutes) == 0 {
		t.Fatal("no new-format internal routes found — route registration broken")
	}

	// Every legacy route must have a new-format counterpart and vice versa.
	for route := range legacyRoutes {
		if !newFmtRoutes[route] {
			t.Errorf("legacy route %s has no new-format counterpart", route)
		}
	}

	for route := range newFmtRoutes {
		if !legacyRoutes[route] {
			t.Errorf("new-format route %s has no legacy counterpart", route)
		}
	}
}

// TestInternalBasepathsRouteCount verifies the expected number of internal
// routes per basepath.  If someone adds a route to only one basepath, this
// test catches the mismatch.
func TestInternalBasepathsRouteCount(t *testing.T) {
	e := echo.New()
	setupRoutes(e, nil, nil)

	var legacyCount, newFmtCount int

	for _, route := range e.Routes() {
		switch {
		case strings.HasPrefix(route.Path, "/internal/sources/"):
			newFmtCount++
		case strings.HasPrefix(route.Path, "/internal/"):
			legacyCount++
		}
	}

	if legacyCount != newFmtCount {
		t.Errorf("route count mismatch: legacy basepath has %d routes, new basepath has %d routes",
			legacyCount, newFmtCount)
	}
}
