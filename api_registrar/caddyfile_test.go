package api_registrar

import (
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func TestParseApiRegistration(t *testing.T) {
	tests := []struct {
		name        string
		caddyfile   string
		expectError bool
		checkFunc   func(t *testing.T, handler *ApiRegistrationHandler)
	}{
		{
			name: "Valid registration with path",
			caddyfile: `
				caddy_api_registrar {
					path /api/v1
					test_api {
						title "Test API"
						version "1.0"
						description "Test description"
					}
				}
			`,
			expectError: false,
			checkFunc: func(t *testing.T, handler *ApiRegistrationHandler) {
				if handler.Path != "/api/v1" {
					t.Errorf("Expected path /api/v1, got %s", handler.Path)
				}
				if len(handler.APIs) != 1 {
					t.Errorf("Expected 1 API, got %d", len(handler.APIs))
				}
				if config, ok := handler.APIs["test_api"]; !ok {
					t.Error("test_api not found")
				} else {
					if config.Title != "Test API" {
						t.Errorf("Expected title 'Test API', got '%s'", config.Title)
					}
					if config.Version != "1.0" {
						t.Errorf("Expected version '1.0', got '%s'", config.Version)
					}
					if config.Description != "Test description" {
						t.Errorf("Expected description 'Test description', got '%s'", config.Description)
					}
				}
			},
		},
		{
			name: "Multiple APIs",
			caddyfile: `
				caddy_api_registrar {
					path /api
					api1 {
						title "API 1"
					}
					api2 {
						version "2.0"
					}
				}
			`,
			expectError: false,
			checkFunc: func(t *testing.T, handler *ApiRegistrationHandler) {
				if len(handler.APIs) != 2 {
					t.Errorf("Expected 2 APIs, got %d", len(handler.APIs))
				}
			},
		},
		{
			name: "Missing path",
			caddyfile: `
				caddy_api_registrar {
					test_api {
						title "Test API"
					}
				}
			`,
			expectError: true,
		},
		{
			name: "With arguments (invalid)",
			caddyfile: `
				caddy_api_registrar arg1 {
					path /api
					test_api {}
				}
			`,
			expectError: true,
		},
		{
			name: "Empty block",
			caddyfile: `
				caddy_api_registrar {
				}
			`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispenser := caddyfile.NewTestDispenser(tt.caddyfile)
			helper := httpcaddyfile.Helper{
				Dispenser: dispenser,
			}

			handler, err := parseApiRegistration(helper)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if handler == nil {
					t.Fatal("Handler is nil")
				}
				if registrationHandler, ok := handler.(*ApiRegistrationHandler); ok {
					if tt.checkFunc != nil {
						tt.checkFunc(t, registrationHandler)
					}
				} else {
					t.Error("Handler is not *ApiRegistrationHandler")
				}
			}
		})
	}
}

func TestParseApiServing(t *testing.T) {
	tests := []struct {
		name        string
		caddyfile   string
		expectError bool
		checkFunc   func(t *testing.T, handler *ApiServingHandler)
	}{
		{
			name: "OpenAPI v3.0 format",
			caddyfile: `
				caddy_api_registrar_serve openapi-v3.0
			`,
			expectError: false,
			checkFunc: func(t *testing.T, handler *ApiServingHandler) {
				if handler.Format != "openapi-v3.0" {
					t.Errorf("Expected format 'openapi-v3.0', got '%s'", handler.Format)
				}
			},
		},
		{
			name: "Swagger UI with spec URL",
			caddyfile: `
				caddy_api_registrar_serve swagger-ui {
					spec_url /api/openapi.json
					server_url https://api.example.com
				}
			`,
			expectError: false,
			checkFunc: func(t *testing.T, handler *ApiServingHandler) {
				if handler.Format != "swagger-ui" {
					t.Errorf("Expected format 'swagger-ui', got '%s'", handler.Format)
				}
				if handler.SpecURL != "/api/openapi.json" {
					t.Errorf("Expected spec_url '/api/openapi.json', got '%s'", handler.SpecURL)
				}
				if handler.ServerURL != "https://api.example.com" {
					t.Errorf("Expected server_url 'https://api.example.com', got '%s'", handler.ServerURL)
				}
			},
		},
		{
			name: "Redoc format",
			caddyfile: `
				caddy_api_registrar_serve redoc
			`,
			expectError: false,
			checkFunc: func(t *testing.T, handler *ApiServingHandler) {
				if handler.Format != "redoc" {
					t.Errorf("Expected format 'redoc', got '%s'", handler.Format)
				}
			},
		},
		{
			name: "Missing format argument",
			caddyfile: `
				caddy_api_registrar_serve
			`,
			expectError: true,
		},
		{
			name: "Too many arguments",
			caddyfile: `
				caddy_api_registrar_serve openapi-v3.0 extra
			`,
			expectError: true,
		},
		{
			name: "Unknown subdirective",
			caddyfile: `
				caddy_api_registrar_serve openapi-v3.0 {
					unknown_option value
				}
			`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispenser := caddyfile.NewTestDispenser(tt.caddyfile)
			helper := httpcaddyfile.Helper{
				Dispenser: dispenser,
			}

			handler, err := parseApiServing(helper)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if handler == nil {
					t.Fatal("Handler is nil")
				}
				if servingHandler, ok := handler.(*ApiServingHandler); ok {
					if tt.checkFunc != nil {
						tt.checkFunc(t, servingHandler)
					}
				} else {
					t.Error("Handler is not *ApiServingHandler")
				}
			}
		})
	}
}
