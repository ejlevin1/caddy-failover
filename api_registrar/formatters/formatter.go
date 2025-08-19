package formatters

import (
	"io"
)

// Formatter is the interface for API documentation format generators
type Formatter interface {
	// Format converts the API specs to the specific format
	Format(specs map[string]*CaddyModuleApiSpec, configs map[string]*ApiConfig) (interface{}, error)

	// ContentType returns the HTTP content type for this format
	ContentType() string

	// Write outputs the formatted spec to the writer
	Write(w io.Writer, spec interface{}) error
}

// CaddyModuleApiSpec represents a module's API specification in a format-agnostic way
type CaddyModuleApiSpec struct {
	ID          string                   `json:"id"`          // e.g., "caddy_api", "failover_api"
	Title       string                   `json:"title"`       // Human-readable title
	Version     string                   `json:"version"`     // API version
	Description string                   `json:"description"` // Optional description
	Endpoints   []CaddyModuleApiEndpoint `json:"endpoints"`   // List of endpoints
}

// CaddyModuleApiEndpoint represents a single API endpoint
type CaddyModuleApiEndpoint struct {
	Method      string              `json:"method"`                 // GET, POST, PUT, PATCH, DELETE
	Path        string              `json:"path"`                   // Relative path like "/status" or "/config/{path}"
	Summary     string              `json:"summary"`                // Short summary
	Description string              `json:"description"`            // Detailed description
	Request     interface{}         `json:"request,omitempty"`      // Request body structure
	Responses   map[int]ResponseDef `json:"responses"`              // Status code -> Response definition
	PathParams  []Parameter         `json:"path_params,omitempty"`  // Path parameters
	QueryParams []Parameter         `json:"query_params,omitempty"` // Query parameters
	Headers     []Parameter         `json:"headers,omitempty"`      // Header parameters
}

// ResponseDef defines a response for a specific status code
type ResponseDef struct {
	Description string      `json:"description"`       // Response description
	Body        interface{} `json:"body,omitempty"`    // Response body structure
	Headers     []Parameter `json:"headers,omitempty"` // Response headers
}

// Parameter represents a parameter (path, query, or header)
type Parameter struct {
	Name        string      `json:"name"`              // Parameter name
	Description string      `json:"description"`       // Parameter description
	Required    bool        `json:"required"`          // Is parameter required?
	Type        string      `json:"type"`              // string, integer, boolean, array, object
	Format      string      `json:"format,omitempty"`  // Format hint (e.g., "date-time", "email")
	Pattern     string      `json:"pattern,omitempty"` // Regex pattern for validation
	Enum        []string    `json:"enum,omitempty"`    // Allowed values
	Default     interface{} `json:"default,omitempty"` // Default value
	Example     interface{} `json:"example,omitempty"` // Example value
}

// ApiConfig represents the configuration for a registered API
type ApiConfig struct {
	Path    string            `json:"path"`              // Base path for this API
	Enabled bool              `json:"enabled"`           // Whether to include in documentation
	Title   string            `json:"title,omitempty"`   // Override title
	Version string            `json:"version,omitempty"` // Override version
	Headers map[string]string `json:"headers,omitempty"` // Global headers for this API
}

// ApiSpecFunc is a function that returns an API specification
type ApiSpecFunc func() *CaddyModuleApiSpec

// GetFormatter returns a formatter for the specified format
func GetFormatter(format string) Formatter {
	switch format {
	case "openapi-v3.0", "openapi-3.0", "openapi":
		return &OpenAPIv3Formatter{}
	case "openapi-v3.1", "openapi-3.1":
		return &OpenAPIv31Formatter{}
	case "swagger-ui", "swaggerui":
		// For UI formatters, the spec URL needs to be set dynamically
		// This should be handled by the handler
		return &SwaggerUIFormatter{}
	case "redoc", "redoc-ui":
		return &RedocUIFormatter{}
	default:
		// Return nil for unknown formats
		return nil
	}
}

// GetFormatterWithContext returns a formatter with context-aware configuration
func GetFormatterWithContext(format string, specPath string) Formatter {
	switch format {
	case "swagger-ui", "swaggerui":
		return &SwaggerUIFormatter{SpecURL: specPath}
	case "redoc", "redoc-ui":
		return &RedocUIFormatter{SpecURL: specPath}
	default:
		return GetFormatter(format)
	}
}

// GetAvailableFormats returns a list of available format names
func GetAvailableFormats() []string {
	return []string{
		"openapi-v3.0",
		"openapi-v3.1",
		"swagger-ui",
		"redoc",
	}
}
