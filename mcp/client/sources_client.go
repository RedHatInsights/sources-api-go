package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type SourcesClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Logger
}

// NewSourcesClient creates a new HTTP client for the sources-api-go service
func NewSourcesClient(baseURL string, log *logrus.Logger) *SourcesClient {
	return &SourcesClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

// ListSources retrieves a list of sources with optional filtering
func (c *SourcesClient) ListSources(ctx context.Context, xrhIdentity string, filter map[string]interface{}, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/sources", c.baseURL)

	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))

	for k, v := range filter {
		params.Add(fmt.Sprintf("filter[%s]", k), fmt.Sprintf("%v", v))
	}

	fullURL := endpoint
	if len(params) > 0 {
		fullURL = endpoint + "?" + params.Encode()
	}

	return c.doRequest(ctx, "GET", fullURL, xrhIdentity, nil)
}

// GetSource retrieves a single source by ID
func (c *SourcesClient) GetSource(ctx context.Context, xrhIdentity string, id string) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/sources/%s", c.baseURL, id)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// ListApplications retrieves a list of applications
func (c *SourcesClient) ListApplications(ctx context.Context, xrhIdentity string, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/applications?limit=%d", c.baseURL, limit)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// GetApplication retrieves a single application by ID
func (c *SourcesClient) GetApplication(ctx context.Context, xrhIdentity string, id string) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/applications/%s", c.baseURL, id)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// ListEndpoints retrieves a list of endpoints
func (c *SourcesClient) ListEndpoints(ctx context.Context, xrhIdentity string, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/endpoints?limit=%d", c.baseURL, limit)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// ListApplicationTypes retrieves a list of application types (metadata)
func (c *SourcesClient) ListApplicationTypes(ctx context.Context, xrhIdentity string) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/application_types", c.baseURL)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// ListSourceTypes retrieves a list of source types (metadata)
func (c *SourcesClient) ListSourceTypes(ctx context.Context, xrhIdentity string) (interface{}, error) {
	endpoint := fmt.Sprintf("%s/api/sources/v3.1/source_types", c.baseURL)
	return c.doRequest(ctx, "GET", endpoint, xrhIdentity, nil)
}

// doRequest executes an HTTP request with the x-rh-identity header
func (c *SourcesClient) doRequest(ctx context.Context, method, url, xrhIdentity string, body io.Reader) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-rh-identity", xrhIdentity)
	req.Header.Set("Content-Type", "application/json")

	c.log.Debugf("Making %s request to %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
