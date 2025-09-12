package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDiscoveryURL(t *testing.T) {
	tests := []struct {
		name        string
		issuer      string
		wantURL     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid HTTPS issuer",
			issuer:  "https://example.com",
			wantURL: "https://example.com/.well-known/openid-configuration",
			wantErr: false,
		},
		{
			name:    "valid HTTPS issuer with trailing slash",
			issuer:  "https://example.com/",
			wantURL: "https://example.com/.well-known/openid-configuration",
			wantErr: false,
		},
		{
			name:    "valid HTTPS issuer with path",
			issuer:  "https://example.com/auth",
			wantURL: "https://example.com/auth/.well-known/openid-configuration",
			wantErr: false,
		},
		{
			name:        "empty issuer",
			issuer:      "",
			wantErr:     true,
			errContains: "JWT issuer not configured",
		},
		{
			name:        "invalid URL",
			issuer:      "not-a-url",
			wantErr:     true,
			errContains: "OIDC discovery URL must be HTTPS",
		},
		{
			name:        "HTTP issuer (not HTTPS)",
			issuer:      "http://example.com",
			wantErr:     true,
			errContains: "OIDC discovery URL must be HTTPS",
		},
		{
			name:        "FTP scheme",
			issuer:      "ftp://example.com",
			wantErr:     true,
			errContains: "OIDC discovery URL must be HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildDiscoveryURL(tt.issuer)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURL, url)
			}
		})
	}
}

func TestBuildDiscoveryURL_LocalhostHTTP(t *testing.T) {
	// Set test environment
	originalGoEnv := os.Getenv("GO_ENV")

	defer func() {
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	os.Setenv("GO_ENV", "test")

	tests := []struct {
		name    string
		issuer  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "localhost HTTP in test environment",
			issuer:  "http://localhost:8080",
			wantURL: "http://localhost:8080/.well-known/openid-configuration",
			wantErr: false,
		},
		{
			name:    "127.0.0.1 HTTP in test environment",
			issuer:  "http://127.0.0.1:8080",
			wantURL: "http://127.0.0.1:8080/.well-known/openid-configuration",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildDiscoveryURL(tt.issuer)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, url)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURL, url)
			}
		})
	}
}

func TestPerformDiscovery(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		expectedIssuer string
		wantJWKSURL    string
		wantErr        bool
		errContains    string
	}{
		{
			name: "valid discovery document",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					doc := map[string]interface{}{
						"issuer":   "https://example.com",
						"jwks_uri": "https://example.com/.well-known/jwks.json",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedIssuer: "https://example.com",
			wantJWKSURL:    "https://example.com/.well-known/jwks.json",
			wantErr:        false,
		},
		{
			name: "discovery document with charset",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					doc := map[string]interface{}{
						"issuer":   "https://example.com",
						"jwks_uri": "https://example.com/.well-known/jwks.json",
					}
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedIssuer: "https://example.com",
			wantJWKSURL:    "https://example.com/.well-known/jwks.json",
			wantErr:        false,
		},
		{
			name: "404 not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"error": "not found"}`))
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery endpoint returned status 404",
		},
		{
			name: "500 internal server error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal error"}`))
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery endpoint returned status 500",
		},
		{
			name: "wrong content type",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.Write([]byte(`<html><body>Not JSON</body></html>`))
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery endpoint returned invalid Content-Type: text/html",
		},
		{
			name: "missing content type",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"issuer": "https://example.com", "jwks_uri": "https://example.com/.well-known/jwks.json"}`))
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery endpoint returned invalid Content-Type:",
		},
		{
			name: "invalid JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"issuer": "https://example.com", "jwks_uri": invalid json`))
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "failed to parse OIDC discovery document",
		},
		{
			name: "issuer mismatch",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					doc := map[string]interface{}{
						"issuer":   "https://different.com",
						"jwks_uri": "https://different.com/.well-known/jwks.json",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery document issuer mismatch: expected https://example.com, got https://different.com",
		},
		{
			name: "missing jwks_uri",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					doc := map[string]interface{}{
						"issuer": "https://example.com",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery document missing jwks_uri field",
		},
		{
			name: "empty jwks_uri",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					doc := map[string]interface{}{
						"issuer":   "https://example.com",
						"jwks_uri": "",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(doc)
				}))
			},
			expectedIssuer: "https://example.com",
			wantErr:        true,
			errContains:    "OIDC discovery document missing jwks_uri field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			ctx := context.Background()
			jwksURL, err := performDiscovery(ctx, server.URL, tt.expectedIssuer)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, jwksURL)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantJWKSURL, jwksURL)
			}
		})
	}
}

func TestDiscoverJWKSURL(t *testing.T) {
	tests := []struct {
		name        string
		jwtIssuer   string
		setupServer func(issuerURL string) *httptest.Server
		wantJWKSURL string
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful discovery",
			jwtIssuer: "", // Will be set to server URL
			setupServer: func(issuerURL string) *httptest.Server {
				var server *httptest.Server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/.well-known/openid-configuration" {
						doc := map[string]interface{}{
							"issuer":   server.URL,
							"jwks_uri": server.URL + "/.well-known/jwks.json",
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(doc)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				return server
			},
			wantJWKSURL: "", // Will be set to server URL + "/.well-known/jwks.json"
			wantErr:     false,
		},
		{
			name:      "missing JWT issuer",
			jwtIssuer: "",
			setupServer: func(issuerURL string) *httptest.Server {
				return nil // No server needed for this test
			},
			wantErr:     true,
			errContains: "OIDC discovery URL validation failed",
		},
		{
			name:      "HTTP issuer (not HTTPS)",
			jwtIssuer: "http://example.com",
			setupServer: func(issuerURL string) *httptest.Server {
				return nil // No server needed for this test
			},
			wantErr:     true,
			errContains: "OIDC discovery URL validation failed",
		},
		{
			name:      "discovery endpoint failure",
			jwtIssuer: "", // Will be set to server URL
			setupServer: func(issuerURL string) *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "server error"}`))
				}))
				return server
			},
			wantErr:     true,
			errContains: "JWKS URL discovery failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			originalIssuer := os.Getenv("JWT_ISSUER")

			defer func() {
				if originalIssuer != "" {
					os.Setenv("JWT_ISSUER", originalIssuer)
				} else {
					os.Unsetenv("JWT_ISSUER")
				}

				config.Reset()
			}()

			var server *httptest.Server

			if tt.setupServer != nil {
				if tt.jwtIssuer == "" && tt.name == "successful discovery" {
					// For successful discovery test, create server first
					server = tt.setupServer("")
					defer server.Close()

					tt.jwtIssuer = server.URL
					tt.wantJWKSURL = server.URL + "/.well-known/jwks.json"
				} else if tt.jwtIssuer == "" && tt.name == "discovery endpoint failure" {
					// For failure test, create server first
					server = tt.setupServer("")
					defer server.Close()

					tt.jwtIssuer = server.URL
				} else {
					server = tt.setupServer(tt.jwtIssuer)
					if server != nil {
						defer server.Close()
					}
				}
			}

			if tt.jwtIssuer != "" {
				os.Setenv("JWT_ISSUER", tt.jwtIssuer)
			} else {
				os.Unsetenv("JWT_ISSUER")
			}

			config.Reset()

			ctx := context.Background()
			jwksURL, err := discoverJWKSURL(ctx)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, jwksURL)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantJWKSURL, jwksURL)
			}
		})
	}
}

func TestDiscoverJWKSURL_Localhost(t *testing.T) {
	// Set test environment to allow localhost HTTP
	originalGoEnv := os.Getenv("GO_ENV")

	defer func() {
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	os.Setenv("GO_ENV", "test")

	// Create test server
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			doc := map[string]interface{}{
				"issuer":   server.URL,
				"jwks_uri": server.URL + "/.well-known/jwks.json",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Set up environment
	originalIssuer := os.Getenv("JWT_ISSUER")

	defer func() {
		if originalIssuer != "" {
			os.Setenv("JWT_ISSUER", originalIssuer)
		} else {
			os.Unsetenv("JWT_ISSUER")
		}

		config.Reset()
	}()

	os.Setenv("JWT_ISSUER", server.URL)
	config.Reset()

	ctx := context.Background()
	jwksURL, err := discoverJWKSURL(ctx)

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/.well-known/jwks.json", jwksURL)
}
