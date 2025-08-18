package api_registrar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestApiRegistrarHandler_ServeHTTP(t *testing.T) {
	// Reset and setup test data
	Reset()
	defer Reset()

	// Register a test API
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

	// Configure the API
	ConfigureApi("test_api", &ApiConfig{
		Path:    "/api",
		Enabled: true,
	})

	// Create handler
	handler := &ApiRegistrarHandler{
		Format: "openapi-v3.0",
	}

	// Provision the handler
	ctx := caddy.Context{}
	if err := handler.Provision(ctx); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	// Test GET request
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()

	err := handler.ServeHTTP(w, req, nil)
	if err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Check cache header
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=300" {
		t.Errorf("Expected Cache-Control 'public, max-age=300', got '%s'", cacheControl)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify OpenAPI structure
	if result["openapi"] == nil {
		t.Error("OpenAPI version not found in response")
	}
	if result["info"] == nil {
		t.Error("Info section not found in response")
	}
	if result["paths"] == nil {
		t.Error("Paths section not found in response")
	}

	// Verify the test endpoint is included
	paths, ok := result["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("Paths is not a map")
	}
	if paths["/api/test"] == nil {
		t.Error("Expected path '/api/test' not found")
	}
}

func TestApiRegistrarHandler_NonGETRequest(t *testing.T) {
	handler := &ApiRegistrarHandler{
		Format: "openapi-v3.0",
	}

	// Create a mock next handler
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		return nil
	})

	// Test POST request (should pass through to next handler)
	req := httptest.NewRequest("POST", "/openapi.json", nil)
	w := httptest.NewRecorder()

	err := handler.ServeHTTP(w, req, next)
	if err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}

	if !nextCalled {
		t.Error("Expected next handler to be called for non-GET request")
	}
}

func TestApiRegistrarHandler_UnsupportedFormat(t *testing.T) {
	handler := &ApiRegistrarHandler{
		Format: "invalid-format",
	}

	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()

	// Note: GetFormatter returns a default formatter for unknown formats
	// so this test checks that it handles the request properly
	err := handler.ServeHTTP(w, req, nil)
	if err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}

	// Should still return valid response (defaults to OpenAPI 3.0)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestApiRegistrarHandler_DefaultFormat(t *testing.T) {
	handler := &ApiRegistrarHandler{
		// Format not specified, should default to openapi-v3.0
	}

	ctx := caddy.Context{}
	if err := handler.Provision(ctx); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	if handler.Format != "openapi-v3.0" {
		t.Errorf("Expected default format 'openapi-v3.0', got '%s'", handler.Format)
	}
}

func TestApiRegistrarHandler_EmptyRegistry(t *testing.T) {
	// Reset registry to ensure it's empty
	Reset()
	defer Reset()

	handler := &ApiRegistrarHandler{
		Format: "openapi-v3.0",
	}

	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()

	err := handler.ServeHTTP(w, req, nil)
	if err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}

	// Should still return valid response with empty paths
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify empty paths
	paths, ok := result["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("Paths is not a map")
	}
	if len(paths) != 0 {
		t.Errorf("Expected empty paths, got %d paths", len(paths))
	}
}

func TestApiRegistrarHandler_DynamicSpecURL(t *testing.T) {
	// Test that Swagger UI gets correct spec URL based on request path
	Reset()
	defer Reset()

	handler := &ApiRegistrarHandler{
		Format: "swagger-ui",
	}

	ctx := caddy.Context{}
	if err := handler.Provision(ctx); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	tests := []struct {
		name        string
		requestPath string
		expectedURL string
	}{
		{
			name:        "API docs path with trailing slash",
			requestPath: "/api/docs/",
			expectedURL: "/api/docs/openapi.json",
		},
		{
			name:        "Caddy OpenAPI path with trailing slash",
			requestPath: "/caddy/openapi/",
			expectedURL: "/caddy/openapi/openapi.json",
		},
		{
			name:        "Custom path with trailing slash",
			requestPath: "/custom/api/",
			expectedURL: "/custom/api/openapi.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			err := handler.ServeHTTP(w, req, nil)
			if err != nil {
				t.Fatalf("ServeHTTP() error = %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Check that the response contains the expected spec URL
			body := w.Body.String()
			if !contains(body, tt.expectedURL) {
				t.Errorf("Expected response to contain spec URL %s, but it didn't", tt.expectedURL)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
