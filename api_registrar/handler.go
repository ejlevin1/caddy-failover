package api_registrar

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/ejlevin1/caddy-failover/api_registrar/formatters"
)

func init() {
	caddy.RegisterModule(&ApiRegistrarHandler{})
	httpcaddyfile.RegisterHandlerDirective("caddy_api_registrar", parseApiRegistrar)
	httpcaddyfile.RegisterGlobalOption("caddy_api_registrar", parseGlobalApiRegistrar)
}

// ApiRegistrarHandler serves API documentation in various formats
type ApiRegistrarHandler struct {
	// Format specifies the output format (e.g., "openapi-v3.0", "openapi-v3.1", "swagger-ui", "redoc")
	Format string `json:"format,omitempty"`
	// SpecURL is the URL to the OpenAPI spec (for UI formatters, optional)
	SpecURL string `json:"spec_url,omitempty"`
}

// CaddyModule returns the Caddy module information
func (*ApiRegistrarHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.caddy_api_registrar",
		New: func() caddy.Module { return new(ApiRegistrarHandler) },
	}
}

// Provision sets up the handler
func (h *ApiRegistrarHandler) Provision(ctx caddy.Context) error {
	// Set default format if not specified
	if h.Format == "" {
		h.Format = "openapi-v3.0"
	}
	return nil
}

// ServeHTTP handles the HTTP request and serves the API documentation
func (h *ApiRegistrarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
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
	}

	if formatter == nil {
		http.Error(w, fmt.Sprintf("Unsupported format: %s", h.Format), http.StatusBadRequest)
		return nil
	}

	// Get specs and configs from registry
	specs := GetSpecs()
	configs := GetConfigs()

	// Generate the API documentation
	doc, err := formatter.Format(specs, configs)
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

// parseApiRegistrar parses the caddy_api_registrar directive in handle blocks
func parseApiRegistrar(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	handler := &ApiRegistrarHandler{}

	// Parse the directive
	for h.Next() {
		// Check for format argument (optional)
		if h.NextArg() {
			handler.Format = h.Val()
			if h.NextArg() {
				return nil, h.ArgErr()
			}
		}

		// Parse block for additional options
		for h.NextBlock(0) {
			switch h.Val() {
			case "format":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				handler.Format = h.Val()
				if h.NextArg() {
					return nil, h.ArgErr()
				}
			case "spec_url":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				handler.SpecURL = h.Val()
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

// parseGlobalApiRegistrar parses the global caddy_api_registrar configuration
func parseGlobalApiRegistrar(d *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	// This parses the global configuration block:
	// {
	//     caddy_api_registrar {
	//         caddy_api {
	//             path /caddy-admin
	//         }
	//         failover_api {
	//             path /caddy/failover
	//         }
	//     }
	// }

	for d.Next() {
		// Parse each API configuration in the block
		for d.NextBlock(0) {
			apiID := d.Val()
			config := &ApiConfig{
				Enabled: true,
			}

			// Parse API-specific configuration
			for d.NextBlock(1) {
				switch d.Val() {
				case "path":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}
					config.Path = d.Val()
					if d.NextArg() {
						return nil, d.ArgErr()
					}
				case "title":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}
					config.Title = d.Val()
					if d.NextArg() {
						return nil, d.ArgErr()
					}
				case "version":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}
					config.Version = d.Val()
					if d.NextArg() {
						return nil, d.ArgErr()
					}
				default:
					return nil, d.Errf("unknown API configuration: %s", d.Val())
				}
			}

			// Register the configuration
			ConfigureApi(apiID, config)
		}
	}

	// Return a non-nil value to indicate successful parsing
	return struct{}{}, nil
}

// Interface guards
var (
	_ caddy.Module                = (*ApiRegistrarHandler)(nil)
	_ caddy.Provisioner           = (*ApiRegistrarHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*ApiRegistrarHandler)(nil)
)
