package api_registrar

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/ejlevin1/caddy-failover/api_registrar/formatters"
)

func init() {
	caddy.RegisterModule(&ApiServingHandler{})
	httpcaddyfile.RegisterHandlerDirective("caddy_api_registrar_serve", parseApiServing)
}

// ApiServingHandler serves API documentation in various formats
type ApiServingHandler struct {
	// Format specifies the output format (e.g., "openapi-v3.0", "openapi-v3.1", "swagger-ui", "redoc")
	Format string `json:"format,omitempty"`
	// SpecURL is the URL to the OpenAPI spec (for UI formatters, optional)
	SpecURL string `json:"spec_url,omitempty"`
	// ServerURL is the base URL for the API server (optional, defaults to dynamic detection)
	ServerURL string `json:"server_url,omitempty"`
}

// CaddyModule returns the Caddy module information
func (*ApiServingHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.caddy_api_registrar_serve",
		New: func() caddy.Module { return new(ApiServingHandler) },
	}
}

// Provision sets up the handler
func (h *ApiServingHandler) Provision(ctx caddy.Context) error {
	// Set default format if not specified
	if h.Format == "" {
		h.Format = "openapi-v3.0"
	}
	return nil
}

// ServeHTTP handles the HTTP request and serves the API documentation
func (h *ApiServingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Only serve on GET requests
	if r.Method != http.MethodGet {
		return next.ServeHTTP(w, r)
	}

	// Get the appropriate formatter with context for UI formatters
	var formatter formatters.Formatter

	// Check if this is a UI format that needs spec URL context
	switch h.Format {
	case "swagger-ui", "swaggerui", "redoc", "redoc-ui":
		// For UI formatters, determine the spec URL
		specURL := h.SpecURL
		if specURL == "" {
			// Build absolute path based on the current request path
			// Remove any trailing parts to get the base path
			basePath := r.URL.Path
			// Remove trailing slash if present
			if len(basePath) > 1 && basePath[len(basePath)-1] == '/' {
				basePath = basePath[:len(basePath)-1]
			}
			// For paths like /caddy/openapi/ -> /caddy/openapi/openapi.json
			// For paths like /api/docs -> /api/docs/openapi.json
			specURL = basePath + "/openapi.json"
		} else if specURL[0] == '.' {
			// Convert relative path to absolute based on current request path
			// For /api/docs with ./openapi.json -> /api/docs/openapi.json
			// For /api/docs/redoc with ../openapi.json -> /api/docs/openapi.json
			basePath := r.URL.Path
			if specURL == "./openapi.json" {
				specURL = basePath + "/openapi.json"
			} else if specURL == "../openapi.json" {
				// Go up one directory
				lastSlash := len(basePath) - 1
				for lastSlash > 0 && basePath[lastSlash] != '/' {
					lastSlash--
				}
				if lastSlash > 0 {
					specURL = basePath[:lastSlash] + "/openapi.json"
				} else {
					specURL = "/openapi.json"
				}
			}
		}
		formatter = formatters.GetFormatterWithContext(h.Format, specURL)
	default:
		formatter = formatters.GetFormatter(h.Format)

		// Determine server URL - use configured value or detect dynamically
		serverURL := h.ServerURL
		if serverURL == "" {
			// Build the server URL from the request
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			// Check X-Forwarded-Proto header (for proxied requests)
			if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			}
			serverURL = fmt.Sprintf("%s://%s", scheme, r.Host)
		}

		// Set server URL for OpenAPI formatters
		switch h.Format {
		case "openapi-v3.0", "openapi-3.0", "openapi":
			if openapiFormatter, ok := formatter.(*formatters.OpenAPIv3Formatter); ok {
				openapiFormatter.ServerURL = serverURL
			}
		case "openapi-v3.1", "openapi-3.1":
			if openapiFormatter, ok := formatter.(*formatters.OpenAPIv31Formatter); ok {
				openapiFormatter.ServerURL = serverURL
			}
		}
	}

	if formatter == nil {
		http.Error(w, fmt.Sprintf("Unsupported format: %s", h.Format), http.StatusBadRequest)
		return nil
	}

	// Get specs and configs from registry
	specs := GetSpecs()
	configs := GetRegisteredApiPaths()

	// Convert to formatters.ApiConfig map format
	formatterConfigs := make(map[string]*formatters.ApiConfig)
	for k, v := range configs {
		formatterConfigs[k] = v
	}

	// Generate the API documentation
	doc, err := formatter.Format(specs, formatterConfigs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating documentation: %v", err), http.StatusInternalServerError)
		return nil
	}

	// Set content type and write response
	w.Header().Set("Content-Type", formatter.ContentType())
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if err := formatter.Write(w, doc); err != nil {
		// Response already started, log error
		return fmt.Errorf("error writing API documentation: %v", err)
	}

	return nil
}

// parseApiServing parses the caddy_api_registrar_serve directive
func parseApiServing(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	handler := &ApiServingHandler{}

	// Parse the directive
	for h.Next() {
		// Check for format argument (required)
		if !h.NextArg() {
			return nil, h.Err("caddy_api_registrar_serve requires a format argument")
		}
		handler.Format = h.Val()

		if h.NextArg() {
			return nil, h.ArgErr()
		}

		// Parse block for additional options
		for h.NextBlock(0) {
			switch h.Val() {
			case "spec_url":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				handler.SpecURL = h.Val()
				if h.NextArg() {
					return nil, h.ArgErr()
				}
			case "server_url":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				handler.ServerURL = h.Val()
				if h.NextArg() {
					return nil, h.ArgErr()
				}
			default:
				return nil, h.Errf("unknown subdirective: %s", h.Val())
			}
		}
	}

	return handler, nil
}

// Interface guards
var (
	_ caddy.Module                = (*ApiServingHandler)(nil)
	_ caddy.Provisioner           = (*ApiServingHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*ApiServingHandler)(nil)
)
