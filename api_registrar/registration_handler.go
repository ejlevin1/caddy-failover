package api_registrar

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
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
	// Path is the path where the APIs are registered (auto-detected or explicit)
	Path string `json:"path,omitempty"`
	// autoDetectedPath stores the auto-detected path for warning purposes (not serialized)
	autoDetectedPath string
	// logger for runtime warnings
	logger *zap.Logger
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
	h.logger = ctx.Logger(h)

	// Log warning if path was explicitly set when auto-detection was available
	if h.autoDetectedPath != "" && h.Path != h.autoDetectedPath {
		h.logger.Warn("path explicitly set, overriding auto-detected handle block path",
			zap.String("explicit_path", h.Path),
			zap.String("auto_detected_path", h.autoDetectedPath))
	}

	if h.Path == "" {
		// This shouldn't happen if parseApiRegistration works correctly, but be defensive
		return fmt.Errorf("path could not be determined for API registration")
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

	// Try to extract the path from the current context
	// This is important for API registration - the path will be auto-detected from the handle block
	autoDetectedPath := ""
	if h.State != nil {
		// Debug: log what keys are available in State
		// fmt.Printf("DEBUG: State keys: %v\n", h.State)

		if segments := h.State["matcher_segments"]; segments != nil {
			if segs, ok := segments.([]caddyhttp.MatcherSet); ok && len(segs) > 0 {
				for _, matcherSet := range segs {
					for _, matcher := range matcherSet {
						if pathMatcher, ok := matcher.(caddyhttp.MatchPath); ok && len(pathMatcher) > 0 {
							autoDetectedPath = string(pathMatcher[0])
							break
						}
					}
				}
			}
		}
	}

	// Check if we're in a snippet
	snippetName := ""
	if h.State != nil {
		if name, ok := h.State["snippet_name"]; ok {
			if nameStr, ok := name.(string); ok {
				snippetName = nameStr
			}
		}
	}

	// Parse the directive
	explicitPath := ""
	for h.Next() {
		// Should not have arguments in registration mode
		if h.NextArg() {
			return nil, h.Err("caddy_api_registrar in registration mode should not have arguments")
		}

		// Parse the block with API registrations
		for h.NextBlock(0) {
			switch h.Val() {
			case "path":
				// Optional path specification (overrides auto-detection)
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				explicitPath = h.Val()
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

	// Determine which path to use based on the fallback logic
	// Get file and line information for better error reporting
	file := h.File()
	line := h.Line()

	// Provide API-specific hints
	hint := ""
	for apiID := range handler.APIs {
		switch apiID {
		case "failover_api":
			hint = " For failover_api, typically use 'path /caddy/failover/status'."
		case "caddy_api":
			hint = " For caddy_api, typically use 'path /caddy'."
		}
		break // Just use the first API's hint
	}

	// Apply fallback logic for path determination
	if explicitPath != "" && autoDetectedPath != "" {
		// Both explicit and auto-detected paths available - use explicit but warn
		handler.Path = explicitPath
		// Log warning will be done during Provision when we have access to logger
		handler.autoDetectedPath = autoDetectedPath // Store for warning during provision
	} else if explicitPath != "" {
		// Only explicit path available - use it
		handler.Path = explicitPath
	} else if autoDetectedPath != "" {
		// Only auto-detected path available - use it
		handler.Path = autoDetectedPath
	} else {
		// No path available - error with helpful context
		location := fmt.Sprintf("%s:%d", file, line)
		if snippetName != "" {
			location = fmt.Sprintf("%s:%d (in snippet '%s')", file, line, snippetName)
			return nil, h.Errf("caddy_api_registrar at %s requires a 'path' directive. "+
				"When used in snippets, path cannot be auto-detected from handle blocks. "+
				"Add 'path /your/api/path' to fix this.%s", location, hint)
		}
		return nil, h.Errf("caddy_api_registrar at %s requires a 'path' directive. "+
			"Either use it inside a 'handle /path/*' block or add 'path /your/api/path' explicitly.%s",
			location, hint)
	}

	return handler, nil
}

// Interface guards
var (
	_ caddy.Module                = (*ApiRegistrationHandler)(nil)
	_ caddy.Provisioner           = (*ApiRegistrationHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*ApiRegistrationHandler)(nil)
)
