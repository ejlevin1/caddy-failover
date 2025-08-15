package caddyfailover

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(&FailoverProxy{})
	httpcaddyfile.RegisterHandlerDirective("failover_proxy", parseFailoverProxy)
}

// FailoverProxy is a Caddy HTTP handler that tries multiple upstream servers
// in sequence until one succeeds, supporting mixed HTTP/HTTPS schemes
type FailoverProxy struct {
	// Upstreams is the list of upstream URLs to try in order
	Upstreams []string `json:"upstreams,omitempty"`

	// UpstreamHeaders is a map of upstream URL to headers
	UpstreamHeaders map[string]map[string]string `json:"upstream_headers,omitempty"`

	// InsecureSkipVerify allows skipping TLS verification for HTTPS upstreams
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// FailDuration is how long to remember a failed upstream (default 30s)
	FailDuration caddy.Duration `json:"fail_duration,omitempty"`

	// DialTimeout is the timeout for establishing connection (default 2s)
	DialTimeout caddy.Duration `json:"dial_timeout,omitempty"`

	// ResponseTimeout is the timeout for receiving response (default 5s)
	ResponseTimeout caddy.Duration `json:"response_timeout,omitempty"`

	logger       *zap.Logger
	httpClient   *http.Client
	httpsClient  *http.Client
	failureCache map[string]time.Time
	mu           sync.RWMutex
}

// CaddyModule returns the Caddy module information
func (*FailoverProxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.failover_proxy",
		New: func() caddy.Module { return new(FailoverProxy) },
	}
}

// Provision sets up the handler
func (f *FailoverProxy) Provision(ctx caddy.Context) error {
	f.logger = ctx.Logger(f)
	f.failureCache = make(map[string]time.Time)

	// Set defaults
	if f.FailDuration == 0 {
		f.FailDuration = caddy.Duration(30 * time.Second)
	}
	if f.DialTimeout == 0 {
		f.DialTimeout = caddy.Duration(2 * time.Second)
	}
	if f.ResponseTimeout == 0 {
		f.ResponseTimeout = caddy.Duration(5 * time.Second)
	}

	// Create HTTP transport
	httpTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Duration(f.DialTimeout),
		}).DialContext,
		ResponseHeaderTimeout: time.Duration(f.ResponseTimeout),
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
	}

	// Create HTTPS transport
	httpsTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Duration(f.DialTimeout),
		}).DialContext,
		ResponseHeaderTimeout: time.Duration(f.ResponseTimeout),
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: f.InsecureSkipVerify,
		},
	}

	// Create clients
	f.httpClient = &http.Client{
		Transport: httpTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	f.httpsClient = &http.Client{
		Transport: httpsTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return nil
}

// ServeHTTP handles the HTTP request
func (f *FailoverProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Try each upstream in order
	for _, upstreamURL := range f.Upstreams {
		// Check if upstream is in failure state
		f.mu.RLock()
		lastFail, failed := f.failureCache[upstreamURL]
		f.mu.RUnlock()

		if failed && time.Since(lastFail) < time.Duration(f.FailDuration) {
			f.logger.Debug("skipping failed upstream",
				zap.String("url", upstreamURL),
				zap.Duration("remaining", time.Duration(f.FailDuration)-time.Since(lastFail)))
			continue
		}

		// Log which upstream we're trying
		f.logger.Debug("attempting upstream",
			zap.String("url", upstreamURL),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path))

		// Try this upstream
		err := f.tryUpstream(w, r, upstreamURL)
		if err == nil {
			// Success! Clear failure cache for this upstream
			f.mu.Lock()
			delete(f.failureCache, upstreamURL)
			f.mu.Unlock()

			f.logger.Info("successfully proxied request",
				zap.String("upstream", upstreamURL),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path))
			return nil
		}

		// Mark failure
		f.mu.Lock()
		f.failureCache[upstreamURL] = time.Now()
		f.mu.Unlock()

		f.logger.Warn("upstream failed, trying next",
			zap.String("url", upstreamURL),
			zap.Error(err))
	}

	// All upstreams failed
	f.logger.Error("all upstreams failed",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Int("upstream_count", len(f.Upstreams)))
	http.Error(w, "All upstreams failed", http.StatusBadGateway)
	return nil
}

// tryUpstream attempts to proxy the request to a single upstream
func (f *FailoverProxy) tryUpstream(w http.ResponseWriter, r *http.Request, upstreamURL string) error {
	// Parse upstream URL
	u, err := url.Parse(upstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}

	// Build target URL preserving upstream base path
	targetURL := *u
	// Join the upstream base path with the request path
	if u.Path != "" && u.Path != "/" {
		// Remove trailing slash from base path to avoid double slashes
		basePath := strings.TrimSuffix(u.Path, "/")
		targetURL.Path = basePath + r.URL.Path
	} else {
		targetURL.Path = r.URL.Path
	}
	targetURL.RawQuery = r.URL.RawQuery

	f.logger.Debug("proxying request",
		zap.String("target_url", targetURL.String()),
		zap.String("method", r.Method))

	// Create new request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers from original request
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Add upstream-specific headers
	if headers, ok := f.UpstreamHeaders[upstreamURL]; ok {
		for name, value := range headers {
			proxyReq.Header.Set(name, value)
		}
	}

	// Set X-Forwarded headers
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}
	// Determine the original protocol (inbound request protocol)
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	// Also check if there's already an X-Forwarded-Proto header from a previous proxy
	if existingProto := r.Header.Get("X-Forwarded-Proto"); existingProto != "" {
		proto = existingProto
	}
	proxyReq.Header.Set("X-Forwarded-Proto", proto)
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	// Choose client based on scheme
	client := f.httpClient
	if u.Scheme == "https" {
		client = f.httpsClient
	}

	// Send request
	resp, err := client.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if response indicates failure (5xx errors)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	return err
}

// parseFailoverProxy parses the Caddyfile configuration
func parseFailoverProxy(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	f := &FailoverProxy{
		UpstreamHeaders: make(map[string]map[string]string),
	}

	// Parse directive arguments (upstream URLs)
	for h.Next() {
		f.Upstreams = h.RemainingArgs()
		if len(f.Upstreams) == 0 {
			return nil, h.Err("at least one upstream URL is required")
		}

		// Parse block for additional options
		for h.NextBlock(0) {
			switch h.Val() {
			case "fail_duration":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				dur, err := caddy.ParseDuration(h.Val())
				if err != nil {
					return nil, h.Errf("invalid fail_duration: %v", err)
				}
				f.FailDuration = caddy.Duration(dur)

			case "dial_timeout":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				dur, err := caddy.ParseDuration(h.Val())
				if err != nil {
					return nil, h.Errf("invalid dial_timeout: %v", err)
				}
				f.DialTimeout = caddy.Duration(dur)

			case "response_timeout":
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				dur, err := caddy.ParseDuration(h.Val())
				if err != nil {
					return nil, h.Errf("invalid response_timeout: %v", err)
				}
				f.ResponseTimeout = caddy.Duration(dur)

			case "insecure_skip_verify":
				f.InsecureSkipVerify = true

			case "header_up":
				// Format: header_up <upstream_url> <header_name> <header_value>
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				upstreamURL := h.Val()

				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				headerName := h.Val()

				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				headerValue := h.Val()

				// Initialize map if needed
				if f.UpstreamHeaders[upstreamURL] == nil {
					f.UpstreamHeaders[upstreamURL] = make(map[string]string)
				}
				f.UpstreamHeaders[upstreamURL][headerName] = headerValue

			default:
				return nil, h.Errf("unknown subdirective: %s", h.Val())
			}
		}
	}

	return f, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler
func (f *FailoverProxy) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		f.Upstreams = d.RemainingArgs()
		if len(f.Upstreams) == 0 {
			return d.Err("at least one upstream URL is required")
		}

		for d.NextBlock(0) {
			switch d.Val() {
			case "fail_duration":
				if !d.NextArg() {
					return d.ArgErr()
				}
				dur, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return d.Errf("invalid fail_duration: %v", err)
				}
				f.FailDuration = caddy.Duration(dur)

			case "insecure_skip_verify":
				f.InsecureSkipVerify = true

			default:
				return d.Errf("unknown subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*FailoverProxy)(nil)
	_ caddyhttp.MiddlewareHandler = (*FailoverProxy)(nil)
	_ caddyfile.Unmarshaler       = (*FailoverProxy)(nil)
)
