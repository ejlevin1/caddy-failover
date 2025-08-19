# API Registrar Module Design Document

## Overview

This document describes the design and implementation plan for adding API documentation capabilities to the caddy-failover module. The API Registrar is designed to be format-agnostic (supporting OpenAPI, GraphQL schemas, etc.) and easily extractable into a separate Caddy module in the future.

## Goals

1. **Provide API documentation** for Caddy's Admin API and registered module APIs
2. **Enable dynamic registration** of API endpoints by Caddy modules
3. **Support multiple output formats** (OpenAPI 3.0, OpenAPI 3.1, potentially GraphQL, etc.)
4. **Support flexible path configuration** through Caddyfile global settings
5. **Maintain modularity** for future extraction into a separate repository

## Architecture

### Directory Structure

```
caddy-failover/
├── api_registrar/                  # API Registrar module (future-extractable)
│   ├── registration_handler.go     # API registration handler (caddy_api_registrar directive)
│   ├── serving_handler.go          # API serving handler (caddy_api_registrar_serve directive)
│   ├── registry.go                 # Global registry for API specs
│   ├── types.go                    # API definition types (format-agnostic)
│   ├── caddy_api.go                # Caddy Admin API definitions
│   ├── formatters/                 # Output format generators
│   │   ├── openapi_v3.go          # OpenAPI 3.0 formatter
│   │   ├── openapi_v3_1.go        # OpenAPI 3.1 formatter
│   │   └── formatter.go            # Formatter interface
│   └── generator.go                # Core spec generation logic
└── failover.go                     # Modified to register failover API
```

### Core Components

#### 1. API Registry (`api_registrar/registry.go`)

A global registry where Caddy modules can register their API specifications during initialization:

```go
package api_registrar

var registry = &ApiRegistry{
    specs: make(map[string]*CaddyModuleApiSpec),
    configs: make(map[string]*ApiConfig),
}

// Called by modules during init()
func RegisterApiSpec(id string, specFunc func() *CaddyModuleApiSpec) {
    registry.mu.Lock()
    defer registry.mu.Unlock()
    registry.specs[id] = specFunc()
}

// Called during Caddyfile parsing to configure API paths
func ConfigureApi(id string, config *ApiConfig) {
    registry.mu.Lock()
    defer registry.mu.Unlock()
    registry.configs[id] = config
}
```

#### 2. Type Definitions (`api_registrar/types.go`)

Format-agnostic structs representing API components and module API specifications:

```go
type CaddyModuleApiSpec struct {
    ID          string                    `json:"id"`          // e.g., "caddy_api", "failover_api"
    Title       string                    `json:"title"`
    Version     string                    `json:"version"`
    Endpoints   []CaddyModuleApiEndpoint  `json:"endpoints"`
}

type CaddyModuleApiEndpoint struct {
    Method      string                    `json:"method"`      // GET, POST, etc.
    Path        string                    `json:"path"`        // Relative path like "/status" or "/config/{path}"
    Summary     string                    `json:"summary"`
    Description string                    `json:"description"`
    Request     interface{}               `json:"request,omitempty"`  // Request struct with JSON tags
    Response    map[int]interface{}       `json:"responses"`          // Status code -> Response struct
    PathParams  []Parameter               `json:"path_params,omitempty"`
    QueryParams []Parameter               `json:"query_params,omitempty"`
}

type Parameter struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Required    bool   `json:"required"`
    Type        string `json:"type"` // string, integer, boolean, etc.
    Pattern     string `json:"pattern,omitempty"`
}

type ApiConfig struct {
    Path        string `json:"path"`        // Base path for this API
    Enabled     bool   `json:"enabled"`     // Whether to include in OpenAPI spec
}
```

#### 3. Caddy Admin API Registration (`api_registrar/caddy_api.go`)

Pre-registered specifications for Caddy's Admin API endpoints:

```go
func init() {
    RegisterApiSpec("caddy_api", getCaddyAdminApiSpec)
}

func getCaddyAdminApiSpec() *CaddyModuleApiSpec {
    return &CaddyModuleApiSpec{
        ID:      "caddy_api",
        Title:   "Caddy Admin API",
        Version: "2.0",
        Endpoints: []CaddyModuleApiEndpoint{
            {
                Method:      "POST",
                Path:        "/load",
                Summary:     "Sets or replaces the active configuration",
                Description: "Loads a new configuration, replacing the current one",
                Request:     &CaddyConfig{},
                Response: map[int]interface{}{
                    200: nil,
                    400: &ErrorResponse{},
                },
            },
            {
                Method:  "GET",
                Path:    "/config/{path}",
                Summary: "Exports the config at the named path",
                PathParams: []Parameter{
                    {Name: "path", Description: "Config path", Required: false, Type: "string"},
                },
                Response: map[int]interface{}{
                    200: &json.RawMessage{},
                },
            },
            // Additional endpoints:
            // POST /config/{path} - Sets or replaces object; appends to array
            // PUT /config/{path} - Creates new object; inserts into array
            // PATCH /config/{path} - Replaces an existing object or array element
            // DELETE /config/{path} - Deletes the value at the named path
            // POST /stop - Stops the active configuration and exits
            // POST /adapt - Adapts a configuration to JSON without running it
            // GET /pki/ca/{id} - Returns information about a PKI app CA
            // GET /pki/ca/{id}/certificates - Returns the certificate chain
            // GET /reverse_proxy/upstreams - Returns proxy upstream status
        },
    }
}
```

#### 4. API Registration Handler (`api_registrar/registration_handler.go`)

A pass-through handler that registers API specifications at specific paths:

```go
type ApiRegistrationHandler struct {
    Path string                           `json:"path,omitempty"`
    APIs map[string]*ApiRegistrationConfig `json:"apis,omitempty"`
}

func (h *ApiRegistrationHandler) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.caddy_api_registrar",
        New: func() caddy.Module { return new(ApiRegistrationHandler) },
    }
}

func (h *ApiRegistrationHandler) Provision(ctx caddy.Context) error {
    if h.Path == "" {
        return fmt.Errorf("path is required for API registration")
    }

    // Register each API with its path
    for apiID, config := range h.APIs {
        apiConfig := &ApiConfig{
            Path:    h.Path,
            Enabled: true,
            Title:   config.Title,
            Version: config.Version,
        }
        if err := RegisterApiPath(apiID, apiConfig); err != nil {
            return err
        }
    }
    return nil
}

// Pass-through handler - registers but doesn't serve
func (h *ApiRegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    return next.ServeHTTP(w, r)
}
```

#### 5. API Serving Handler (`api_registrar/serving_handler.go`)

The HTTP handler that serves the generated API specification in the requested format:

```go
type ApiServingHandler struct {
    Format    string `json:"format,omitempty"`     // "openapi-v3.0", "swagger-ui", etc.
    SpecURL   string `json:"spec_url,omitempty"`   // For UI formatters
    ServerURL string `json:"server_url,omitempty"` // Optional server URL override
}

func (h *ApiServingHandler) CaddyModule() caddy.ModuleInfo {
    return caddy.ModuleInfo{
        ID:  "http.handlers.caddy_api_registrar_serve",
        New: func() caddy.Module { return new(ApiServingHandler) },
    }
}

func (h *ApiServingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
    formatter := getFormatter(h.Format)
    if formatter == nil {
        http.Error(w, "Unsupported format", http.StatusBadRequest)
        return nil
    }

    // For UI formatters, set spec URL
    if uiFormatter, ok := formatter.(UIFormatter); ok {
        uiFormatter.SetSpecURL(h.SpecURL)
    }

    // Generate and serve the spec
    spec := h.generateApiSpec(formatter)
    w.Header().Set("Content-Type", formatter.ContentType())
    formatter.Write(w, spec)
    return nil
}
```

#### 6. Formatter Interface (`api_registrar/formatters/formatter.go`)

Interface for different output format generators:

```go
package formatters

type Formatter interface {
    // Format converts the API specs to the specific format
    Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) interface{}

    // ContentType returns the HTTP content type for this format
    ContentType() string

    // Write outputs the formatted spec to the writer
    Write(w io.Writer, spec interface{}) error
}
```

#### 7. OpenAPI Formatter (`api_registrar/formatters/openapi_v3.go`)

OpenAPI 3.0 specific formatter implementation:

```go
package formatters

type OpenAPIv3Formatter struct{}

func (f *OpenAPIv3Formatter) Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) interface{} {
    // Convert generic API specs to OpenAPI 3.0 format
    openapi := &OpenAPISpec{
        OpenAPI: "3.0.3",
        Info: Info{
            Title: "Caddy Server API",
            Version: "2.0.0",
        },
        Paths: make(map[string]*PathItem),
        Components: &Components{
            Schemas: make(map[string]*Schema),
        },
    }

    // Process each configured API
    for id, config := range configs {
        if !config.Enabled {
            continue
        }
        spec := specs[id]
        // Convert endpoints to OpenAPI paths with base path
        for _, endpoint := range spec.Endpoints {
            path := config.Path + endpoint.Path
            // Add to openapi.Paths
        }
    }

    return openapi
}

func (f *OpenAPIv3Formatter) ContentType() string {
    return "application/json"
}

func (f *OpenAPIv3Formatter) Write(w io.Writer, spec interface{}) error {
    return json.NewEncoder(w).Encode(spec)
}
```

## Integration Points

### 1. Module Registration

Modules register their API specifications during initialization:

```go
// In failover.go
func init() {
    api_registrar.RegisterApiSpec("failover_api", getFailoverApiSpec)
}

func getFailoverApiSpec() *api_registrar.CaddyModuleApiSpec {
    return &api_registrar.CaddyModuleApiSpec{
        ID:      "failover_api",
        Title:   "Failover Status API",
        Version: "1.0",
        Endpoints: []api_registrar.CaddyModuleApiEndpoint{
            {
                Method:      "GET",
                Path:        "/status",
                Summary:     "Get failover proxy status",
                Description: "Returns the current status of all registered failover proxies",
                Response: map[int]interface{}{
                    200: &[]PathStatus{},
                },
            },
        },
    }
}
```

### 2. Caddyfile Configuration

The new two-directive system separates registration from serving:

#### Registration (at the actual API path):

```caddyfile
handle /caddy/failover/status {
    # Register the API at this exact path
    caddy_api_registrar {
        path /caddy/failover/status
        failover_api {
            title "Failover Plugin API"
            version "1.0.0"
        }
    }
    failover_status  # The actual handler
}
```

This ensures the API is registered at the exact path where it's served, preventing path mismatches.

### 3. Caddyfile Serving Configuration

Users add documentation endpoints using the serving directive:

```caddyfile
# Serve OpenAPI 3.0 format
handle /api/docs/openapi.json {
    caddy_api_registrar_serve openapi-v3.0
}

# Serve Swagger UI
handle /api/docs* {
    caddy_api_registrar_serve swagger-ui {
        spec_url /api/docs/openapi.json
    }
}

# Serve Redoc UI
handle /api/docs/redoc* {
    caddy_api_registrar_serve redoc {
        spec_url /api/docs/openapi.json
    }
}

# Future: Could support other formats
handle /graphql/schema {
    caddy_api_registrar_serve graphql-sdl
}
```

## Generated Output Example (OpenAPI 3.0 Format)

When using the OpenAPI 3.0 formatter, the handler generates:

```json
{
  "openapi": "3.0.3",
  "info": {
    "title": "Caddy Server API",
    "version": "2.0.0",
    "description": "Caddy web server administration and module APIs"
  },
  "servers": [
    {
      "url": "http://localhost",
      "description": "Default server"
    }
  ],
  "paths": {
    "/caddy-admin/load": {
      "post": {
        "summary": "Sets or replaces the active configuration",
        "operationId": "loadConfig",
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/CaddyConfig"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Configuration loaded successfully"
          },
          "400": {
            "description": "Invalid configuration",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorResponse"
                }
              }
            }
          }
        }
      }
    },
    "/caddy-admin/config/{path}": {
      "get": {
        "summary": "Exports the config at the named path",
        "parameters": [
          {
            "name": "path",
            "in": "path",
            "required": false,
            "schema": {
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Configuration object",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object"
                }
              }
            }
          }
        }
      },
      "post": { /* ... */ },
      "put": { /* ... */ },
      "patch": { /* ... */ },
      "delete": { /* ... */ }
    },
    "/caddy/failover/status": {
      "get": {
        "summary": "Get failover proxy status",
        "operationId": "getFailoverStatus",
        "responses": {
          "200": {
            "description": "Current status of all failover proxies",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "$ref": "#/components/schemas/PathStatus"
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "CaddyConfig": {
        "type": "object",
        "properties": {
          "apps": {
            "type": "object"
          }
        }
      },
      "PathStatus": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string"
          },
          "active": {
            "type": "string"
          },
          "failover_proxies": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/UpstreamStatus"
            }
          }
        }
      },
      "UpstreamStatus": {
        "type": "object",
        "properties": {
          "host": {
            "type": "string"
          },
          "status": {
            "type": "string",
            "enum": ["UP", "DOWN", "UNHEALTHY"]
          },
          "health_check_enabled": {
            "type": "boolean"
          },
          "response_time_ms": {
            "type": "integer"
          }
        }
      }
    }
  }
}
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. Create `api_registrar/` directory structure
2. Implement format-agnostic type definitions (`types.go`)
3. Build registry system (`registry.go`)
4. Create formatter interface (`formatters/formatter.go`)
5. Create basic handler (`handler.go`)

### Phase 2: OpenAPI Formatter
1. Implement OpenAPI 3.0 formatter (`formatters/openapi_v3.go`)
2. Implement OpenAPI 3.1 formatter (`formatters/openapi_v3_1.go`)
3. Add OpenAPI-specific type conversions

### Phase 3: Caddy Admin API
1. Define Caddy Admin API specifications (`caddy_api.go`)
2. Add Caddyfile parsing for global configuration
3. Test Caddy API documentation generation

### Phase 4: Module Integration
1. Update `failover.go` to register its API
2. Test registration and path configuration
3. Verify merged API documentation

### Phase 5: Testing & Documentation
1. Add unit tests for formatters
2. Test with example Caddyfile configurations
3. Document usage in README

## Future Considerations

### Extraction to Separate Module

The `api_registrar/` directory is designed to be self-contained and can be extracted to a separate repository (e.g., `github.com/ejlevin1/caddy-api-registrar`) by:

1. Moving the `api_registrar/` directory to new repository
2. Updating import paths
3. Building with `xcaddy`:
   ```bash
   xcaddy build \
     --with github.com/ejlevin1/caddy-failover \
     --with github.com/ejlevin1/caddy-api-registrar
   ```

### Additional Features

Potential future enhancements:
- **Additional Formatters**:
  - GraphQL SDL formatter for GraphQL schemas
  - AsyncAPI formatter for event-driven APIs
  - JSON Schema formatter for validation
  - Protobuf/gRPC service definitions
- **Swagger UI integration** for interactive documentation
- **Schema validation** middleware using the registered specs
- **Example generation** from struct tags
- **Authentication documentation** (OAuth2, API keys, etc.)
- **Webhook documentation** support
- **API versioning** support with multiple versions per module

## Design Decisions

### Why "API Registrar" Instead of "OpenAPI"?

We chose a format-agnostic naming and architecture because:
1. **Flexibility**: Supports multiple documentation formats (OpenAPI, GraphQL, etc.)
2. **Future-proof**: Can add new formatters without changing the core architecture
3. **Separation of concerns**: Registration logic is separate from format-specific output
4. **Extensibility**: Easy to add new formats as plugins/formatters

### Why Not External Schema Files?

We chose not to support loading external schema files in the Caddyfile because:
1. External schemas won't have corresponding endpoints to service them
2. Modules should be responsible for documenting their own APIs
3. Keeps the configuration simpler and more maintainable
4. Ensures documentation stays in sync with implementation

### Why Two-Directive System?

The separation of `caddy_api_registrar` (registration) and `caddy_api_registrar_serve` (serving) provides:
1. **Path accuracy**: APIs are registered exactly where they're mounted, preventing documentation mismatches
2. **Clear separation**: Registration happens with the API handler, serving happens at documentation endpoints
3. **Flexibility**: Multiple documentation endpoints can serve the same registered APIs
4. **Pass-through behavior**: Registration doesn't interfere with the actual API handler
5. **Better maintainability**: Changes to API paths automatically update documentation

### Why Native Go Structs?

Using native Go structs with reflection instead of external schema libraries:
1. Better integration with existing Caddy patterns
2. Single source of truth (the actual request/response structs)
3. No additional dependencies for basic functionality
4. Type safety at compile time
5. Format-agnostic internal representation

### Why Formatter Pattern?

The formatter pattern provides:
1. **Pluggability**: New formats can be added without changing core code
2. **Testability**: Each formatter can be tested independently
3. **Maintainability**: Format-specific logic is isolated
4. **Performance**: Formatters can be optimized individually

## Conclusion

This design provides a clean, modular, and format-agnostic approach to adding API documentation to Caddy modules. The architecture supports multiple output formats, maintains separation of concerns, and is designed for easy extraction into a standalone module. The use of formatters ensures that new documentation standards can be adopted without restructuring the core registration system.
