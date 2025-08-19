package api_registrar

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(&ApiRegistrationHandler{})
	httpcaddyfile.RegisterHandlerDirective("caddy_api_registrar", parseApiRegistration)
	// Note: Removed global option since we're using per-handle registration
}

// ApiRegistrationHandler registers API metadata for documentation generation
// This is a pass-through handler that doesn't serve content
type ApiRegistrationHandler struct {
	// APIs is a map of API ID to configuration
	APIs map[string]*ApiRegistrationConfig `json:"apis,omitempty"`
	// Path is the path where the APIs are registered (required)
	Path string `json:"path,omitempty"`
}

// ApiRegistrationConfig contains configuration for a registered API
type ApiRegistrationConfig struct {
	// Title overrides the default title
	Title string `json:"title,omitempty"`
	// Version overrides the default version
	Version string `json:"version,omitempty"`
	// Description overrides the default description
	Description string `json:"description,omitempty"`
}

// CaddyModule returns the Caddy module information
func (*ApiRegistrationHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.caddy_api_registrar",
		New: func() caddy.Module { return new(ApiRegistrationHandler) },
	}
}

// Provision sets up the handler and registers the APIs
func (h *ApiRegistrationHandler) Provision(ctx caddy.Context) error {
	if h.Path == "" {
		return fmt.Errorf("path is required for API registration")
	}

	// Register each API with its path
	for apiID, config := range h.APIs {
		// Check if this API spec is registered
		if !IsApiSpecRegistered(apiID) {
			return fmt.Errorf("unknown API '%s' - it must be registered during module initialization", apiID)
		}

		// Create an ApiConfig with the path
		apiConfig := &ApiConfig{
			Path:    h.Path,
			Enabled: true,
		}

		// Apply overrides from registration config
		if config.Title != "" {
			apiConfig.Title = config.Title
		}
		if config.Version != "" {
			apiConfig.Version = config.Version
		}

		// Register the API configuration
		if err := RegisterApiPath(apiID, apiConfig); err != nil {
			return err
		}
	}

	return nil
}

// ServeHTTP is a pass-through handler - registration doesn't serve content
func (h *ApiRegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Pass through to the next handler (the actual API handler)
	return next.ServeHTTP(w, r)
}

// parseApiRegistration parses the caddy_api_registrar directive for registration
func parseApiRegistration(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	handler := &ApiRegistrationHandler{
		APIs: make(map[string]*ApiRegistrationConfig),
	}

	// Parse the directive
	for h.Next() {
		// Should not have arguments in registration mode
		if h.NextArg() {
			return nil, h.Err("caddy_api_registrar in registration mode should not have arguments")
		}

		// Parse the block with API registrations
		for h.NextBlock(0) {
			switch h.Val() {
			case "path":
				// Required path specification
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				handler.Path = h.Val()
				if h.NextArg() {
					return nil, h.ArgErr()
				}
			default:
				// This is an API ID
				apiID := h.Val()
				config := &ApiRegistrationConfig{}

				// Parse API-specific configuration
				for h.NextBlock(1) {
					switch h.Val() {
					case "title":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						config.Title = h.Val()
						if h.NextArg() {
							return nil, h.ArgErr()
						}
					case "version":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						config.Version = h.Val()
						if h.NextArg() {
							return nil, h.ArgErr()
						}
					case "description":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						config.Description = h.Val()
						if h.NextArg() {
							return nil, h.ArgErr()
						}
					default:
						return nil, h.Errf("unknown subdirective: %s", h.Val())
					}
				}

				handler.APIs[apiID] = config
			}
		}
	}

	if len(handler.APIs) == 0 {
		return nil, h.Err("caddy_api_registrar must have at least one API registered")
	}

	// Path is required for API registration
	if handler.Path == "" {
		return nil, h.Err("caddy_api_registrar requires a 'path' directive to specify where the API is mounted")
	}

	return handler, nil
}

// Interface guards
var (
	_ caddy.Module                = (*ApiRegistrationHandler)(nil)
	_ caddy.Provisioner           = (*ApiRegistrationHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*ApiRegistrationHandler)(nil)
)
