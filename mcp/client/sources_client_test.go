package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/sources-api-go/mcp/client"
	"github.com/sirupsen/logrus"
)

func TestListSources(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/sources" {
			t.Errorf("Expected path /api/sources/v3.1/sources, got %s", r.URL.Path)
		}
		if r.Header.Get("x-rh-identity") != "test-identity" {
			t.Errorf("Expected x-rh-identity header 'test-identity', got '%s'", r.Header.Get("x-rh-identity"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "1", "name": "test-source"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.ListSources(context.Background(), "test-identity", nil, 100)
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListSourcesWithFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("limit") != "50" {
			t.Errorf("Expected limit=50, got %s", query.Get("limit"))
		}
		if query.Get("filter[name]") != "test" {
			t.Errorf("Expected filter[name]=test, got %s", query.Get("filter[name]"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	filter := map[string]interface{}{
		"name": "test",
	}
	_, err := c.ListSources(context.Background(), "test-identity", filter, 50)
	if err != nil {
		t.Fatalf("ListSources with filter failed: %v", err)
	}
}

func TestGetSource(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/sources/123" {
			t.Errorf("Expected path /api/sources/v3.1/sources/123, got %s", r.URL.Path)
		}
		if r.Header.Get("x-rh-identity") != "test-identity" {
			t.Errorf("Expected x-rh-identity header 'test-identity', got '%s'", r.Header.Get("x-rh-identity"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "123", "name": "test-source"}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.GetSource(context.Background(), "test-identity", "123")
	if err != nil {
		t.Fatalf("GetSource failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetSourceNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors": [{"detail": "Source not found"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	_, err := c.GetSource(context.Background(), "test-identity", "999")
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
}

func TestListApplications(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/applications" {
			t.Errorf("Expected path /api/sources/v3.1/applications, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "1", "source_id": "10"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.ListApplications(context.Background(), "test-identity", 100)
	if err != nil {
		t.Fatalf("ListApplications failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestGetApplication(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/applications/456" {
			t.Errorf("Expected path /api/sources/v3.1/applications/456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "456", "source_id": "123"}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.GetApplication(context.Background(), "test-identity", "456")
	if err != nil {
		t.Fatalf("GetApplication failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListEndpoints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/endpoints" {
			t.Errorf("Expected path /api/sources/v3.1/endpoints, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "1", "host": "example.com"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.ListEndpoints(context.Background(), "test-identity", 100)
	if err != nil {
		t.Fatalf("ListEndpoints failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListApplicationTypes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/application_types" {
			t.Errorf("Expected path /api/sources/v3.1/application_types, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "1", "name": "/insights/platform/cost-management"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.ListApplicationTypes(context.Background(), "test-identity")
	if err != nil {
		t.Fatalf("ListApplicationTypes failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestListSourceTypes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sources/v3.1/source_types" {
			t.Errorf("Expected path /api/sources/v3.1/source_types, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [{"id": "1", "name": "amazon", "product_name": "Amazon Web Services"}]}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	result, err := c.ListSourceTypes(context.Background(), "test-identity")
	if err != nil {
		t.Fatalf("ListSourceTypes failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	_, err := c.ListSources(context.Background(), "test-identity", nil, 100)
	if err == nil {
		t.Fatal("Expected error for 500 response")
	}
}

func TestUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer ts.Close()

	log := logrus.New()
	c := client.NewSourcesClient(ts.URL, log)

	_, err := c.ListSources(context.Background(), "test-identity", nil, 100)
	if err == nil {
		t.Fatal("Expected error for 401 response")
	}
}
