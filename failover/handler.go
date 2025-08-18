// Package caddyfailover provides automatic failover capabilities for Caddy server
package failover

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
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
	"github.com/ejlevin1/caddy-failover/api_registrar"
	"go.uber.org/zap"
)

// ParseFailoverProxy parses the failover_proxy directive
func ParseFailoverProxy(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return parseFailoverProxy(h)
}

// ParseFailoverStatus parses the failover_status directive
func ParseFailoverStatus(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return parseFailoverStatus(h)
}

var (
	// Global registry to track all failover proxy instances
	proxyRegistry = &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}
)

// ProxyEntry represents a unique proxy configuration for a path
type ProxyEntry struct {
	Path      string
	Proxy     *FailoverProxy
	Upstreams map[string]bool // Track unique upstreams to prevent duplicates
}

// ProxyRegistry tracks all failover proxy instances for status reporting
type ProxyRegistry struct {
	mu      sync.RWMutex
	proxies map[string]*ProxyEntry // path -> proxy entry
	order   []string               // maintains registration order
}

// Register adds a proxy to the registry
func (r *ProxyRegistry) Register(path string, proxy *FailoverProxy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if this path already exists
	if entry, exists := r.proxies[path]; exists {
		// Merge upstreams if not already present
		for _, upstream := range proxy.Upstreams {
			if !entry.Upstreams[upstream] {
				// This is a new upstream for this path, but we'll use the first proxy's configuration
				// to avoid duplicates. This ensures we don't duplicate the same upstream
				entry.Upstreams[upstream] = true
			}
		}
	} else {
		// New path, create entry
		entry := &ProxyEntry{
			Path:      path,
			Proxy:     proxy,
			Upstreams: make(map[string]bool),
		}
		for _, upstream := range proxy.Upstreams {
			entry.Upstreams[upstream] = true
		}
		r.proxies[path] = entry
		r.order = append(r.order, path)
	}
}

// Unregister removes a proxy from the registry
func (r *ProxyRegistry) Unregister(path string, proxy *FailoverProxy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.proxies[path]; exists {
		// Only remove if this is the same proxy instance
		if entry.Proxy == proxy {
			delete(r.proxies, path)
			// Remove from order slice
			for i, p := range r.order {
				if p == path {
					r.order = append(r.order[:i], r.order[i+1:]...)
					break
				}
			}
		}
	}
}

// GetStatus returns the status of all registered proxies
func (r *ProxyRegistry) GetStatus() []PathStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var status []PathStatus
	// Use order to maintain consistent ordering
	for _, path := range r.order {
		entry, exists := r.proxies[path]
		if !exists {
			continue
		}

		// Use the clean path for display
		displayPath := path
		if entry.Proxy.HandlePath != "" {
			displayPath = entry.Proxy.HandlePath
		}

		ps := PathStatus{
			Path:            displayPath,
			FailoverProxies: entry.Proxy.GetUpstreamStatus(),
		}

		// Get the active upstream
		if active := entry.Proxy.GetActiveUpstream(); active != "" {
			ps.Active = active
		}

		status = append(status, ps)
	}
	return status
}

// PathStatus represents the status of failover proxies for a path
type PathStatus struct {
	Path            string           `json:"path"`
	Active          string           `json:"active,omitempty"`
	FailoverProxies []UpstreamStatus `json:"failover_proxies"`
}

// UpstreamStatus represents the status of a single upstream
type UpstreamStatus struct {
	Host         string    `json:"host"`
	Status       string    `json:"status"` // UP, DOWN, UNHEALTHY
	LastCheck    time.Time `json:"last_check,omitempty"`
	LastFailure  time.Time `json:"last_failure,omitempty"`
	HealthCheck  bool      `json:"health_check_enabled"`
	ResponseTime int64     `json:"response_time_ms,omitempty"`
}

// HealthCheck defines health check configuration for an upstream
type HealthCheck struct {
	// Path is the health check endpoint path
	Path string `json:"path,omitempty"`

	// Interval is how often to perform health checks (default 30s)
	Interval caddy.Duration `json:"interval,omitempty"`

	// Timeout is the timeout for health check requests (default 5s)
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// ExpectedStatus is the expected HTTP status code (default 200)
	ExpectedStatus int `json:"expected_status,omitempty"`
}

// FailoverProxy is a Caddy HTTP handler that tries multiple upstream servers
// in sequence until one succeeds, supporting mixed HTTP/HTTPS schemes
type FailoverProxy struct {
	// Upstreams is the list of upstream URLs to try in order
	Upstreams []string `json:"upstreams,omitempty"`

	// UpstreamHeaders is a map of upstream URL to headers
	UpstreamHeaders map[string]map[string]string `json:"upstream_headers,omitempty"`

	// HealthChecks is a map of upstream URL to health check configuration
	HealthChecks map[string]*HealthCheck `json:"health_checks,omitempty"`

	// InsecureSkipVerify allows skipping TLS verification for HTTPS upstreams
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// FailDuration is how long to remember a failed upstream (default 30s)
	FailDuration caddy.Duration `json:"fail_duration,omitempty"`

	// DialTimeout is the timeout for establishing connection (default 2s)
	DialTimeout caddy.Duration `json:"dial_timeout,omitempty"`

	// ResponseTimeout is the timeout for receiving response (default 5s)
	ResponseTimeout caddy.Duration `json:"response_timeout,omitempty"`

	// Path is the route path this proxy handles (for status reporting)
	Path string `json:"path,omitempty"`

	// HandlePath is the actual handle block path (e.g., /auth/*)
	HandlePath string `json:"handle_path,omitempty"`

	logger        *zap.Logger
	replacer      *caddy.Replacer
	httpClient    *http.Client
	httpsClient   *http.Client
	failureCache  map[string]time.Time
	healthStatus  map[string]bool // true = healthy, false = unhealthy
	lastCheckTime map[string]time.Time
	responseTime  map[string]int64 // response time in milliseconds
	mu            sync.RWMutex
	shutdown      chan struct{}
	wg            sync.WaitGroup
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
	f.replacer = caddy.NewReplacer()
	f.failureCache = make(map[string]time.Time)
	f.healthStatus = make(map[string]bool)
	f.lastCheckTime = make(map[string]time.Time)
	f.responseTime = make(map[string]int64)
	f.shutdown = make(chan struct{})

	// Register with global registry
	// Use Path if explicitly set, otherwise use HandlePath
	registrationPath := f.Path
	if registrationPath == "" && f.HandlePath != "" {
		registrationPath = f.HandlePath
	}

	// If still no path, generate one based on the upstreams for status tracking
	// This ensures the status endpoint always shows something
	if registrationPath == "" && len(f.Upstreams) > 0 {
		// Generate a unique identifier based on the first upstream
		// This is a fallback to ensure status tracking works even without explicit path
		registrationPath = fmt.Sprintf("auto-%x", hashString(f.Upstreams[0]))
		f.logger.Debug("No path found for failover proxy, using auto-generated path",
			zap.String("auto_path", registrationPath),
			zap.Strings("upstreams", f.Upstreams))
	}

	// Register if we have a valid path (explicit or auto-generated)
	if registrationPath != "" {
		proxyRegistry.Register(registrationPath, f)
	}

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

	// Expand environment variables in upstream URLs
	for i, upstream := range f.Upstreams {
		expanded := f.replacer.ReplaceAll(upstream, "")
		if expanded != upstream {
			f.logger.Debug("expanded upstream URL",
				zap.String("original", upstream),
				zap.String("expanded", expanded))
		}
		f.Upstreams[i] = expanded
	}

	// Expand environment variables in upstream headers
	expandedHeaders := make(map[string]map[string]string)
	for upstream, headers := range f.UpstreamHeaders {
		expandedUpstream := f.replacer.ReplaceAll(upstream, "")
		if expandedUpstream != upstream {
			f.logger.Debug("expanded upstream in header_up",
				zap.String("original", upstream),
				zap.String("expanded", expandedUpstream))
		}
		expandedHeaders[expandedUpstream] = make(map[string]string)
		for name, value := range headers {
			expandedValue := f.replacer.ReplaceAll(value, "")
			if expandedValue != value {
				f.logger.Debug("expanded header value",
					zap.String("upstream", expandedUpstream),
					zap.String("header", name),
					zap.String("original", value),
					zap.String("expanded", expandedValue))
			}
			expandedHeaders[expandedUpstream][name] = expandedValue
		}
	}
	f.UpstreamHeaders = expandedHeaders

	// Expand environment variables in health check URLs
	expandedHealthChecks := make(map[string]*HealthCheck)
	for upstream, hc := range f.HealthChecks {
		expandedUpstream := f.replacer.ReplaceAll(upstream, "")
		if expandedUpstream != upstream {
			f.logger.Debug("expanded upstream in health_check",
				zap.String("original", upstream),
				zap.String("expanded", expandedUpstream))
		}
		expandedHealthChecks[expandedUpstream] = hc
	}
	f.HealthChecks = expandedHealthChecks

	// Set health check defaults and start health checkers
	// Initialize health check defaults (but don't start goroutines yet)
	for _, hc := range f.HealthChecks {
		if hc.Interval == 0 {
			hc.Interval = caddy.Duration(30 * time.Second)
		}
		if hc.Timeout == 0 {
			hc.Timeout = caddy.Duration(5 * time.Second)
		}
		if hc.ExpectedStatus == 0 {
			hc.ExpectedStatus = 200
		}
		if hc.Path == "" {
			hc.Path = "/health"
		}
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

	// Now start health check goroutines after clients are initialized
	for upstream, hc := range f.HealthChecks {
		f.wg.Add(1)
		go f.runHealthCheck(upstream, hc)
	}

	return nil
}

// Cleanup stops health check goroutines
func (f *FailoverProxy) Cleanup() error {
	close(f.shutdown)
	f.wg.Wait()

	// Unregister from global registry
	registrationPath := f.Path
	if registrationPath == "" && f.HandlePath != "" {
		registrationPath = f.HandlePath
	}
	if registrationPath != "" {
		proxyRegistry.Unregister(registrationPath, f)
	}
	return nil
}

// GetActiveUpstream returns the currently active (healthy and not failed) upstream
func (f *FailoverProxy) GetActiveUpstream() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Find the first healthy upstream that isn't in failure state
	for _, upstream := range f.Upstreams {
		// Check if upstream is healthy
		if hc := f.HealthChecks[upstream]; hc != nil {
			if healthy, exists := f.healthStatus[upstream]; exists && !healthy {
				continue // Skip unhealthy upstreams
			}
		}

		// Check if upstream is in failure state
		if lastFail, failed := f.failureCache[upstream]; failed {
			if time.Since(lastFail) < time.Duration(f.FailDuration) {
				continue // Skip failed upstreams
			}
		}

		// This upstream is active
		return upstream
	}

	// If no healthy upstreams, return the first one as fallback
	if len(f.Upstreams) > 0 {
		return f.Upstreams[0]
	}
	return ""
}

// GetUpstreamStatus returns the current status of all upstreams
func (f *FailoverProxy) GetUpstreamStatus() []UpstreamStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var statuses []UpstreamStatus
	for _, upstream := range f.Upstreams {
		status := UpstreamStatus{
			Host:        upstream,
			HealthCheck: f.HealthChecks[upstream] != nil,
		}

		// Determine status
		if healthy, exists := f.healthStatus[upstream]; exists {
			if healthy {
				status.Status = "UP"
			} else {
				status.Status = "UNHEALTHY"
			}
		} else if lastFail, failed := f.failureCache[upstream]; failed {
			if time.Since(lastFail) < time.Duration(f.FailDuration) {
				status.Status = "DOWN"
				status.LastFailure = lastFail
			} else {
				status.Status = "UP"
			}
		} else {
			status.Status = "UP"
		}

		// Add last check time if available
		if checkTime, exists := f.lastCheckTime[upstream]; exists {
			status.LastCheck = checkTime
		}

		// Add response time if available
		if respTime, exists := f.responseTime[upstream]; exists {
			status.ResponseTime = respTime
		}

		statuses = append(statuses, status)
	}
	return statuses
}

// runHealthCheck runs periodic health checks for an upstream
func (f *FailoverProxy) runHealthCheck(upstreamURL string, hc *HealthCheck) {
	defer f.wg.Done()

	u, err := url.Parse(upstreamURL)
	if err != nil {
		f.logger.Error("invalid upstream URL for health check",
			zap.String("upstream", upstreamURL),
			zap.Error(err))
		return
	}

	// Build health check URL
	healthURL := *u
	healthURL.Path = hc.Path
	healthURL.RawQuery = ""

	ticker := time.NewTicker(time.Duration(hc.Interval))
	defer ticker.Stop()

	// Perform initial health check
	f.performHealthCheck(healthURL.String(), upstreamURL, hc)

	for {
		select {
		case <-ticker.C:
			f.performHealthCheck(healthURL.String(), upstreamURL, hc)
		case <-f.shutdown:
			return
		}
	}
}

// performHealthCheck performs a single health check
func (f *FailoverProxy) performHealthCheck(healthURL, upstreamURL string, hc *HealthCheck) {
	u, _ := url.Parse(healthURL)
	client := f.httpClient
	if u.Scheme == "https" {
		client = f.httpsClient
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(hc.Timeout))
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		f.setHealthStatus(upstreamURL, false)
		f.logger.Debug("health check failed to create request",
			zap.String("upstream", upstreamURL),
			zap.Error(err))
		return
	}

	// Set custom user agent for health checks
	req.Header.Set("User-Agent", "Caddy-failover-health-check/1.0")

	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	// Update check time and response time
	f.mu.Lock()
	f.lastCheckTime[upstreamURL] = time.Now()
	if err == nil {
		f.responseTime[upstreamURL] = elapsed
	}
	f.mu.Unlock()

	if err != nil {
		f.setHealthStatus(upstreamURL, false)
		f.logger.Debug("health check failed",
			zap.String("upstream", upstreamURL),
			zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// Drain the body to allow connection reuse
	io.Copy(io.Discard, resp.Body)

	healthy := resp.StatusCode == hc.ExpectedStatus
	f.setHealthStatus(upstreamURL, healthy)

	if healthy {
		f.logger.Debug("health check passed",
			zap.String("upstream", upstreamURL),
			zap.Int("status", resp.StatusCode))
	} else {
		f.logger.Warn("health check failed",
			zap.String("upstream", upstreamURL),
			zap.Int("status", resp.StatusCode),
			zap.Int("expected", hc.ExpectedStatus))
	}
}

// setHealthStatus updates the health status of an upstream
func (f *FailoverProxy) setHealthStatus(upstreamURL string, healthy bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	prevStatus, exists := f.healthStatus[upstreamURL]
	f.healthStatus[upstreamURL] = healthy

	// Log status changes
	if !exists || prevStatus != healthy {
		if healthy {
			// Clear failure cache when upstream becomes healthy
			delete(f.failureCache, upstreamURL)
			f.logger.Info("upstream became healthy",
				zap.String("upstream", upstreamURL))
		} else {
			f.logger.Warn("upstream became unhealthy",
				zap.String("upstream", upstreamURL))
		}
	}
}

// isHealthy checks if an upstream is healthy
func (f *FailoverProxy) isHealthy(upstreamURL string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// If no health check is configured, consider it healthy
	if _, hasHealthCheck := f.HealthChecks[upstreamURL]; !hasHealthCheck {
		return true
	}

	// Return health status (default to unhealthy if not yet checked)
	healthy, exists := f.healthStatus[upstreamURL]
	return exists && healthy
}

// ServeHTTP handles the HTTP request
func (f *FailoverProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Track the index of the upstream we're trying
	attemptedUpstreams := 0

	// Try each upstream in order
	for i, upstreamURL := range f.Upstreams {
		// Check if upstream is healthy
		if !f.isHealthy(upstreamURL) {
			f.logger.Debug("skipping unhealthy upstream",
				zap.String("url", upstreamURL))
			attemptedUpstreams++
			continue
		}

		// Check if upstream is in failure state
		f.mu.RLock()
		lastFail, failed := f.failureCache[upstreamURL]
		f.mu.RUnlock()

		if failed && time.Since(lastFail) < time.Duration(f.FailDuration) {
			f.logger.Debug("skipping failed upstream",
				zap.String("url", upstreamURL),
				zap.Duration("remaining", time.Duration(f.FailDuration)-time.Since(lastFail)))
			attemptedUpstreams++
			continue
		}

		// Log failover warning if we're not using the primary upstream
		if attemptedUpstreams > 0 {
			f.logger.Warn("failing over to alternate upstream",
				zap.String("primary", f.Upstreams[0]),
				zap.String("failover_to", upstreamURL),
				zap.Int("upstream_index", i),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path))
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

		f.logger.Debug("upstream failed, trying next",
			zap.String("url", upstreamURL),
			zap.Error(err))
		attemptedUpstreams++
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
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Try to extract the path from the current context
	// This is important for status tracking - without a path, the proxy won't be registered
	if h.State != nil {
		if segments := h.State["matcher_segments"]; segments != nil {
			if segs, ok := segments.([]caddyhttp.MatcherSet); ok && len(segs) > 0 {
				for _, matcherSet := range segs {
					for _, matcher := range matcherSet {
						if pathMatcher, ok := matcher.(caddyhttp.MatchPath); ok && len(pathMatcher) > 0 {
							f.HandlePath = string(pathMatcher[0])
							// Also set Path as default if not explicitly overridden later
							f.Path = f.HandlePath
							break
						}
					}
				}
			}
		}
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

			case "status_path":
				// Allow manual configuration of the path for status reporting
				// This overrides the registration key but preserves HandlePath for display
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				f.Path = h.Val()

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

			case "health_check":
				// Format: health_check <upstream_url> { ... }
				if !h.NextArg() {
					return nil, h.ArgErr()
				}
				upstreamURL := h.Val()

				hc := &HealthCheck{}

				// Parse nested block for health check options
				for h.NextBlock(1) {
					switch h.Val() {
					case "path":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						hc.Path = h.Val()

					case "interval":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						dur, err := caddy.ParseDuration(h.Val())
						if err != nil {
							return nil, h.Errf("invalid health check interval: %v", err)
						}
						hc.Interval = caddy.Duration(dur)

					case "timeout":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						dur, err := caddy.ParseDuration(h.Val())
						if err != nil {
							return nil, h.Errf("invalid health check timeout: %v", err)
						}
						hc.Timeout = caddy.Duration(dur)

					case "expected_status":
						if !h.NextArg() {
							return nil, h.ArgErr()
						}
						var status int
						_, err := fmt.Sscanf(h.Val(), "%d", &status)
						if err != nil {
							return nil, h.Errf("invalid expected_status: %v", err)
						}
						hc.ExpectedStatus = status

					default:
						return nil, h.Errf("unknown health_check subdirective: %s", h.Val())
					}
				}

				f.HealthChecks[upstreamURL] = hc

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

// FailoverStatusHandler provides an HTTP endpoint for status information
type FailoverStatusHandler struct{}

// CaddyModule returns the Caddy module information
func (FailoverStatusHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.failover_status",
		New: func() caddy.Module { return new(FailoverStatusHandler) },
	}
}

// ServeHTTP handles the status request
func (h FailoverStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return nil
	}

	status := proxyRegistry.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
	return nil
}

// parseFailoverStatus parses the failover_status directive
func parseFailoverStatus(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	for h.Next() {
		if h.NextArg() {
			return nil, h.ArgErr()
		}
	}
	return FailoverStatusHandler{}, nil
}

// hashString creates a short hash of a string for use as an identifier
func hashString(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h[:4]) // Use first 4 bytes for a shorter hash
}

// getFailoverApiSpec returns the failover API specification
func getFailoverApiSpec() *api_registrar.CaddyModuleApiSpec {
	return &api_registrar.CaddyModuleApiSpec{
		ID:          "failover_api",
		Title:       "Failover Status API",
		Version:     "1.0",
		Description: "API for monitoring and managing failover proxy status",
		Endpoints: []api_registrar.CaddyModuleApiEndpoint{
			{
				Method:      "GET",
				Path:        "/status",
				Summary:     "Get failover proxy status",
				Description: "Returns the current status of all registered failover proxies including their upstreams, health checks, and active states",
				Responses: map[int]api_registrar.ResponseDef{
					200: {
						Description: "List of failover proxy statuses",
						Body:        []PathStatus{},
					},
				},
			},
		},
	}
}

// GetFailoverApiSpec returns the failover API specification
func GetFailoverApiSpec() *api_registrar.CaddyModuleApiSpec {
	return &api_registrar.CaddyModuleApiSpec{
		ID:          "failover_api",
		Title:       "Failover Status API",
		Version:     "1.0",
		Description: "API for monitoring and managing failover proxy status",
		Endpoints: []api_registrar.CaddyModuleApiEndpoint{
			{
				Method:      "GET",
				Path:        "/status",
				Summary:     "Get failover proxy status",
				Description: "Returns the current status of all registered failover proxies including their upstreams, health checks, and active states",
				Responses: map[int]api_registrar.ResponseDef{
					200: {
						Description: "List of failover proxy statuses",
						Body:        []PathStatus{},
					},
				},
			},
		},
	}
}

// Interface guards
var (
	_ caddy.Provisioner           = (*FailoverProxy)(nil)
	_ caddy.CleanerUpper          = (*FailoverProxy)(nil)
	_ caddyhttp.MiddlewareHandler = (*FailoverProxy)(nil)
	_ caddyfile.Unmarshaler       = (*FailoverProxy)(nil)
	_ caddy.Module                = (*FailoverStatusHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*FailoverStatusHandler)(nil)
)
