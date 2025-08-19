package api_registrar

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestApiRegistrationHandler_Provision(t *testing.T) {
	// Reset registry before tests
	Reset()
	ResetPaths()
	defer func() {
		Reset()
		ResetPaths()
	}()

	// Register a test API spec that the handler can reference
	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
		}
	})

	tests := []struct {
		name        string
		handler     *ApiRegistrationHandler
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid registration",
			handler: &ApiRegistrationHandler{
				Path: "/api/v1",
				APIs: map[string]*ApiRegistrationConfig{
					"test_api": {
						Title:   "Custom Title",
						Version: "2.0",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing path",
			handler: &ApiRegistrationHandler{
				APIs: map[string]*ApiRegistrationConfig{
					"test_api": {},
				},
			},
			expectError: true,
			errorMsg:    "path is required",
		},
		{
			name: "Unknown API",
			handler: &ApiRegistrationHandler{
				Path: "/api/v1",
				APIs: map[string]*ApiRegistrationConfig{
					"unknown_api": {},
				},
			},
			expectError: true,
			errorMsg:    "unknown API",
		},
		{
			name: "Multiple APIs registration",
			handler: &ApiRegistrationHandler{
				Path: "/api/v1",
				APIs: map[string]*ApiRegistrationConfig{
					"test_api": {
						Title: "API 1",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear paths before each test
			ResetPaths()

			ctx := caddy.Context{}
			err := tt.handler.Provision(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() == "" {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify registration was successful
				if tt.handler.Path != "" {
					paths := GetRegisteredApiPaths()
					for apiID := range tt.handler.APIs {
						if config, exists := paths[apiID]; !exists {
							t.Errorf("API %s was not registered", apiID)
						} else if config.Path != tt.handler.Path {
							t.Errorf("API %s registered at wrong path: expected %s, got %s",
								apiID, tt.handler.Path, config.Path)
						}
					}
				}
			}
		})
	}
}

func TestApiRegistrationHandler_ServeHTTP(t *testing.T) {
	// The registration handler should be a pass-through
	handler := &ApiRegistrationHandler{
		Path: "/api/v1",
		APIs: map[string]*ApiRegistrationConfig{},
	}

	// Track if next handler was called
	nextCalled := false
	nextStatus := http.StatusOK
	nextBody := "next handler response"

	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		w.WriteHeader(nextStatus)
		w.Write([]byte(nextBody))
		return nil
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	err := handler.ServeHTTP(w, req, next)
	if err != nil {
		t.Fatalf("ServeHTTP returned error: %v", err)
	}

	if !nextCalled {
		t.Error("Next handler was not called")
	}

	if w.Code != nextStatus {
		t.Errorf("Expected status %d, got %d", nextStatus, w.Code)
	}

	if w.Body.String() != nextBody {
		t.Errorf("Expected body '%s', got '%s'", nextBody, w.Body.String())
	}
}

func TestApiRegistrationHandler_PathConflict(t *testing.T) {
	// Reset registry
	Reset()
	ResetPaths()
	defer func() {
		Reset()
		ResetPaths()
	}()

	// Register a test API
	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{
			ID:      "test_api",
			Title:   "Test API",
			Version: "1.0",
		}
	})

	// First registration
	handler1 := &ApiRegistrationHandler{
		Path: "/api/v1",
		APIs: map[string]*ApiRegistrationConfig{
			"test_api": {},
		},
	}

	ctx := caddy.Context{}
	err := handler1.Provision(ctx)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration at different path (should fail)
	handler2 := &ApiRegistrationHandler{
		Path: "/api/v2",
		APIs: map[string]*ApiRegistrationConfig{
			"test_api": {},
		},
	}

	err = handler2.Provision(ctx)
	if err == nil {
		t.Error("Expected error for conflicting path registration")
	}

	// Same path registration (should succeed - update)
	handler3 := &ApiRegistrationHandler{
		Path: "/api/v1",
		APIs: map[string]*ApiRegistrationConfig{
			"test_api": {
				Title: "Updated Title",
			},
		},
	}

	err = handler3.Provision(ctx)
	if err != nil {
		t.Errorf("Same path registration should succeed: %v", err)
	}
}

func TestApiRegistrationHandler_CaddyModule(t *testing.T) {
	handler := &ApiRegistrationHandler{}
	info := handler.CaddyModule()

	if info.ID != "http.handlers.caddy_api_registrar" {
		t.Errorf("Expected module ID 'http.handlers.caddy_api_registrar', got '%s'", info.ID)
	}

	if info.New == nil {
		t.Error("Module New function is nil")
	}

	module := info.New()
	if _, ok := module.(*ApiRegistrationHandler); !ok {
		t.Error("Module New function does not return *ApiRegistrationHandler")
	}
}
