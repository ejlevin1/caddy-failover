package api_registrar

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestApiServingHandler_ServeHTTP(t *testing.T) {
	// Reset and setup test data
	Reset()
	ResetPaths()
	defer func() {
		Reset()
		ResetPaths()
	}()

	// Register a test API spec
	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
			Endpoints: []CaddyModuleApiEndpoint{
				{
					Method:  "GET",
					Path:    "/test",
					Summary: "Test endpoint",
					Responses: map[int]ResponseDef{
						200: {Description: "Success"},
					},
				},
			},
		}
	})

	// Register the API at a path (simulating what registration handler does)
	RegisterApiPath("test_api", &ApiConfig{
		Path:    "/api",
		Enabled: true,
	})

	// Create serving handler
	handler := &ApiServingHandler{
		Format: "openapi-v3.0",
	}

	// Provision the handler
	ctx := caddy.Context{}
	if err := handler.Provision(ctx); err != nil {
		t.Fatalf("Failed to provision handler: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		path           string
		format         string
		expectedStatus int
		checkContent   func(t *testing.T, body []byte)
	}{
		{
			name:           "GET OpenAPI v3.0 JSON",
			method:         "GET",
			path:           "/api/openapi.json",
			format:         "openapi-v3.0",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body []byte) {
				var doc map[string]interface{}
				if err := json.Unmarshal(body, &doc); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if doc["openapi"] != "3.0.3" {
					t.Errorf("Expected OpenAPI version 3.0.3, got %v", doc["openapi"])
				}
				if doc["info"] == nil {
					t.Error("Missing info section")
				}
				if doc["paths"] == nil {
					t.Error("Missing paths section")
				}
			},
		},
		{
			name:           "GET OpenAPI v3.1 JSON",
			method:         "GET",
			path:           "/api/openapi-3.1.json",
			format:         "openapi-v3.1",
			expectedStatus: http.StatusOK,
			checkContent: func(t *testing.T, body []byte) {
				var doc map[string]interface{}
				if err := json.Unmarshal(body, &doc); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if doc["openapi"] != "3.1.0" {
					t.Errorf("Expected OpenAPI version 3.1.0, got %v", doc["openapi"])
				}
			},
		},
		{
			name:           "POST should pass through",
			method:         "POST",
			path:           "/api/openapi.json",
			format:         "openapi-v3.0",
			expectedStatus: http.StatusOK, // Will call the mock next handler
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new handler for each test with the specified format
			testHandler := &ApiServingHandler{
				Format: tt.format,
			}
			if err := testHandler.Provision(ctx); err != nil {
				t.Fatalf("Failed to provision handler: %v", err)
			}

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Create a mock next handler
			nextCalled := false
			next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
				return nil
			})

			// Serve the request
			err := testHandler.ServeHTTP(w, req, next)
			if err != nil {
				t.Fatalf("ServeHTTP returned error: %v", err)
			}

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content if provided
			if tt.checkContent != nil && !nextCalled {
				tt.checkContent(t, w.Body.Bytes())
			}

			// For non-GET requests, ensure next handler was called
			if tt.method != "GET" && !nextCalled {
				t.Error("Expected next handler to be called for non-GET request")
			}
		})
	}
}

func TestApiServingHandler_Formats(t *testing.T) {
	// Reset and setup test data
	Reset()
	ResetPaths()
	defer func() {
		Reset()
		ResetPaths()
	}()

	// Register a minimal test API
	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
		}
	})

	RegisterApiPath("test_api", &ApiConfig{
		Path:    "/api",
		Enabled: true,
	})

	tests := []struct {
		name          string
		format        string
		expectedType  string
		expectedError bool
	}{
		{
			name:         "OpenAPI v3.0",
			format:       "openapi-v3.0",
			expectedType: "application/json",
		},
		{
			name:         "OpenAPI v3.1",
			format:       "openapi-v3.1",
			expectedType: "application/json",
		},
		{
			name:         "Swagger UI",
			format:       "swagger-ui",
			expectedType: "text/html; charset=utf-8",
		},
		{
			name:         "Redoc",
			format:       "redoc",
			expectedType: "text/html; charset=utf-8",
		},
		{
			name:          "Invalid format",
			format:        "invalid-format",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ApiServingHandler{
				Format: tt.format,
			}

			ctx := caddy.Context{}
			if err := handler.Provision(ctx); err != nil {
				t.Fatalf("Failed to provision handler: %v", err)
			}

			req := httptest.NewRequest("GET", "/api/docs", nil)
			w := httptest.NewRecorder()

			next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				// Don't write anything - let the handler set the status
				return nil
			})

			err := handler.ServeHTTP(w, req, next)

			if tt.expectedError {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected bad request for invalid format, got %d (body: %s)", w.Code, w.Body.String())
				}
			} else {
				if err != nil {
					t.Fatalf("ServeHTTP returned error: %v", err)
				}
				if w.Header().Get("Content-Type") != tt.expectedType {
					t.Errorf("Expected content type %s, got %s", tt.expectedType, w.Header().Get("Content-Type"))
				}
			}
		})
	}
}

func TestApiServingHandler_ServerURL(t *testing.T) {
	// Reset and setup test data
	Reset()
	ResetPaths()
	defer func() {
		Reset()
		ResetPaths()
	}()

	// Register a minimal test API
	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
			Endpoints: []CaddyModuleApiEndpoint{
				{
					Method:  "GET",
					Path:    "/test",
					Summary: "Test endpoint",
				},
			},
		}
	})

	RegisterApiPath("test_api", &ApiConfig{
		Path:    "/api",
		Enabled: true,
	})

	tests := []struct {
		name              string
		configuredURL     string
		requestHost       string
		requestTLS        bool
		xForwardedProto   string
		expectedServerURL string
	}{
		{
			name:              "Auto-detect HTTP",
			configuredURL:     "",
			requestHost:       "example.com",
			requestTLS:        false,
			expectedServerURL: "http://example.com",
		},
		{
			name:              "Auto-detect HTTPS",
			configuredURL:     "",
			requestHost:       "example.com",
			requestTLS:        true,
			expectedServerURL: "https://example.com",
		},
		{
			name:              "X-Forwarded-Proto override",
			configuredURL:     "",
			requestHost:       "example.com",
			requestTLS:        false,
			xForwardedProto:   "https",
			expectedServerURL: "https://example.com",
		},
		{
			name:              "Configured URL",
			configuredURL:     "https://api.example.com",
			requestHost:       "localhost",
			expectedServerURL: "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ApiServingHandler{
				Format:    "openapi-v3.0",
				ServerURL: tt.configuredURL,
			}

			ctx := caddy.Context{}
			if err := handler.Provision(ctx); err != nil {
				t.Fatalf("Failed to provision handler: %v", err)
			}

			req := httptest.NewRequest("GET", "/api/openapi.json", nil)
			req.Host = tt.requestHost
			if tt.requestTLS {
				// Simulate TLS by setting a non-nil TLS field
				// We don't need actual TLS details, just non-nil to indicate HTTPS
				req.TLS = &tls.ConnectionState{}
			}
			if tt.xForwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.xForwardedProto)
			}

			w := httptest.NewRecorder()
			next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				return nil
			})

			err := handler.ServeHTTP(w, req, next)
			if err != nil {
				t.Fatalf("ServeHTTP returned error: %v", err)
			}

			// Parse the response to check the server URL
			var doc map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			servers, ok := doc["servers"].([]interface{})
			if !ok || len(servers) == 0 {
				t.Fatal("No servers found in OpenAPI spec")
			}

			server := servers[0].(map[string]interface{})
			if server["url"] != tt.expectedServerURL {
				t.Errorf("Expected server URL %s, got %s", tt.expectedServerURL, server["url"])
			}
		})
	}
}
