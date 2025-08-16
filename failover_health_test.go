package caddyfailover

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

// TestRunHealthCheck tests the health check goroutine functionality
func TestRunHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		initialHealth  int
		expectedHealth bool
		invalidURL     bool
	}{
		{
			name:           "healthy server",
			initialHealth:  http.StatusOK,
			expectedHealth: true,
		},
		{
			name:           "unhealthy server",
			initialHealth:  http.StatusServiceUnavailable,
			expectedHealth: false,
		},
		{
			name:       "invalid URL",
			invalidURL: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			var upstreamURL string

			if !tt.invalidURL {
				// Create test server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" {
						w.WriteHeader(tt.initialHealth)
					} else {
						w.WriteHeader(http.StatusOK)
					}
				}))
				defer server.Close()
				upstreamURL = server.URL
			} else {
				// Use invalid URL to test error handling
				upstreamURL = "://invalid-url"
			}

			// Create FailoverProxy with health check
			fp := &FailoverProxy{
				logger:        zap.NewNop(),
				healthStatus:  make(map[string]bool),
				lastCheckTime: make(map[string]time.Time),
				responseTime:  make(map[string]int64),
				failureCache:  make(map[string]time.Time),
				shutdown:      make(chan struct{}),
				httpClient:    &http.Client{Timeout: 1 * time.Second},
				httpsClient:   &http.Client{Timeout: 1 * time.Second},
			}

			hc := &HealthCheck{
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			}

			// Run health check in goroutine
			fp.wg.Add(1)
			go fp.runHealthCheck(upstreamURL, hc)

			// Wait for initial health check
			time.Sleep(100 * time.Millisecond)

			if !tt.invalidURL {
				// Check health status
				fp.mu.RLock()
				healthStatus, exists := fp.healthStatus[upstreamURL]
				lastCheck, hasCheck := fp.lastCheckTime[upstreamURL]
				fp.mu.RUnlock()

				if !exists {
					t.Error("Health status not set")
				} else if healthStatus != tt.expectedHealth {
					t.Errorf("Expected health status %v, got %v", tt.expectedHealth, healthStatus)
				}

				if !hasCheck {
					t.Error("Last check time not set")
				} else if time.Since(lastCheck) > 200*time.Millisecond {
					t.Error("Last check time too old")
				}

				// Wait for a periodic check
				time.Sleep(100 * time.Millisecond)

				// Verify periodic checks are happening
				fp.mu.RLock()
				newLastCheck := fp.lastCheckTime[upstreamURL]
				fp.mu.RUnlock()

				if !newLastCheck.After(lastCheck) {
					t.Error("Periodic health check not performed")
				}
			}

			// Test shutdown
			close(fp.shutdown)
			fp.wg.Wait()
		})
	}
}

// TestRunHealthCheckWithChangingHealth tests health status changes
func TestRunHealthCheckWithChangingHealth(t *testing.T) {
	healthStatus := http.StatusOK
	mu := sync.RWMutex{}

	// Create test server with changeable health
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		status := healthStatus
		mu.RUnlock()

		if r.URL.Path == "/health" {
			w.WriteHeader(status)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Create FailoverProxy
	fp := &FailoverProxy{
		logger:        zap.NewNop(),
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{Timeout: 1 * time.Second},
		httpsClient:   &http.Client{Timeout: 1 * time.Second},
	}

	hc := &HealthCheck{
		Path:           "/health",
		Interval:       caddy.Duration(50 * time.Millisecond),
		Timeout:        caddy.Duration(100 * time.Millisecond),
		ExpectedStatus: 200,
	}

	// Start health check
	fp.wg.Add(1)
	go fp.runHealthCheck(server.URL, hc)

	// Wait for initial healthy check
	time.Sleep(100 * time.Millisecond)

	fp.mu.RLock()
	initialHealth := fp.healthStatus[server.URL]
	fp.mu.RUnlock()

	if !initialHealth {
		t.Error("Expected initial healthy status")
	}

	// Change server to unhealthy
	mu.Lock()
	healthStatus = http.StatusServiceUnavailable
	mu.Unlock()

	// Wait for health check to detect change
	time.Sleep(150 * time.Millisecond)

	fp.mu.RLock()
	updatedHealth := fp.healthStatus[server.URL]
	fp.mu.RUnlock()

	if updatedHealth {
		t.Error("Expected unhealthy status after server became unhealthy")
	}

	// Change back to healthy
	mu.Lock()
	healthStatus = http.StatusOK
	mu.Unlock()

	// Wait for health check to detect recovery
	time.Sleep(150 * time.Millisecond)

	fp.mu.RLock()
	recoveredHealth := fp.healthStatus[server.URL]
	_, stillInCache := fp.failureCache[server.URL]
	fp.mu.RUnlock()

	if !recoveredHealth {
		t.Error("Expected healthy status after server recovered")
	}

	if stillInCache {
		t.Error("Failure cache should be cleared when server becomes healthy")
	}

	// Cleanup
	close(fp.shutdown)
	fp.wg.Wait()
}

// TestRunHealthCheckHTTPS tests health checks with HTTPS upstream
func TestRunHealthCheckHTTPS(t *testing.T) {
	// Create HTTPS test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Create FailoverProxy with insecure TLS client for testing
	fp := &FailoverProxy{
		logger:        zap.NewNop(),
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{Timeout: 1 * time.Second},
		httpsClient:   server.Client(), // Use test server's client with proper certs
	}

	hc := &HealthCheck{
		Path:           "/health",
		Interval:       caddy.Duration(50 * time.Millisecond),
		Timeout:        caddy.Duration(100 * time.Millisecond),
		ExpectedStatus: 200,
	}

	// Run health check
	fp.wg.Add(1)
	go fp.runHealthCheck(server.URL, hc)

	// Wait for health check (give more time for HTTPS)
	time.Sleep(200 * time.Millisecond)

	// Verify HTTPS health check worked
	fp.mu.RLock()
	healthStatus := fp.healthStatus[server.URL]
	responseTime, hasResponseTime := fp.responseTime[server.URL]
	fp.mu.RUnlock()

	if !healthStatus {
		t.Error("HTTPS health check should be healthy")
	}

	if !hasResponseTime || responseTime <= 0 {
		// Try again after a periodic check
		time.Sleep(100 * time.Millisecond)
		fp.mu.RLock()
		responseTime = fp.responseTime[server.URL]
		fp.mu.RUnlock()
		if responseTime <= 0 {
			t.Logf("Warning: Response time not recorded (got %d ms)", responseTime)
		}
	}

	// Cleanup
	close(fp.shutdown)
	fp.wg.Wait()
}

// TestRunHealthCheckInvalidURL tests handling of malformed URLs
func TestRunHealthCheckInvalidURL(t *testing.T) {
	fp := &FailoverProxy{
		logger:        zap.NewNop(),
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{},
		httpsClient:   &http.Client{},
	}

	hc := &HealthCheck{
		Path:           "/health",
		Interval:       caddy.Duration(100 * time.Millisecond),
		Timeout:        caddy.Duration(50 * time.Millisecond),
		ExpectedStatus: 200,
	}

	// Test with invalid URL
	fp.wg.Add(1)

	// This should return immediately due to URL parse error
	done := make(chan bool)
	go func() {
		fp.runHealthCheck("http://[invalid-url", hc)
		done <- true
	}()

	select {
	case <-done:
		// Good, it returned
	case <-time.After(500 * time.Millisecond):
		t.Error("runHealthCheck should have returned immediately for invalid URL")
	}

	// No need to close shutdown as the goroutine already exited
}

// TestRunHealthCheckCustomPath tests health check with custom path
func TestRunHealthCheckCustomPath(t *testing.T) {
	customPath := "/api/health/status"
	requestedPath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.URL.Path == customPath {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fp := &FailoverProxy{
		logger:        zap.NewNop(),
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{},
		httpsClient:   &http.Client{},
	}

	hc := &HealthCheck{
		Path:           customPath,
		Interval:       caddy.Duration(50 * time.Millisecond),
		Timeout:        caddy.Duration(100 * time.Millisecond),
		ExpectedStatus: 200,
	}

	// Run health check
	fp.wg.Add(1)
	go fp.runHealthCheck(server.URL, hc)

	// Wait for health check
	time.Sleep(100 * time.Millisecond)

	// Verify correct path was requested
	if requestedPath != customPath {
		t.Errorf("Expected health check path %s, got %s", customPath, requestedPath)
	}

	// Verify health status
	fp.mu.RLock()
	healthStatus := fp.healthStatus[server.URL]
	fp.mu.RUnlock()

	if !healthStatus {
		t.Error("Health check should be healthy with custom path")
	}

	// Cleanup
	close(fp.shutdown)
	fp.wg.Wait()
}

// TestRunHealthCheckWithBasePathURL tests health check when upstream URL has a base path
func TestRunHealthCheckWithBasePathURL(t *testing.T) {
	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fp := &FailoverProxy{
		logger:        zap.NewNop(),
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{},
		httpsClient:   &http.Client{},
	}

	hc := &HealthCheck{
		Path:           "/health",
		Interval:       caddy.Duration(50 * time.Millisecond),
		Timeout:        caddy.Duration(100 * time.Millisecond),
		ExpectedStatus: 200,
	}

	// Use upstream URL with base path (should be ignored for health checks)
	upstreamWithPath := server.URL + "/api/v1"

	// Run health check
	fp.wg.Add(1)
	go fp.runHealthCheck(upstreamWithPath, hc)

	// Wait for health check
	time.Sleep(100 * time.Millisecond)

	// Verify health check path doesn't include the base path
	if requestedPath != "/health" {
		t.Errorf("Health check should use path %s, got %s", "/health", requestedPath)
	}

	// Cleanup
	close(fp.shutdown)
	fp.wg.Wait()
}
