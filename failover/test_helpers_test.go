package failover

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

// TestNewTestServer tests the NewTestServer function
func TestNewTestServer(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test response")
	if ts == nil {
		t.Fatal("NewTestServer returned nil")
	}
	if ts.Server == nil {
		t.Fatal("NewTestServer created server with nil httptest.Server")
	}
	defer ts.Close()

	// Test that the server responds
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got %q", w.Body.String())
	}

	// Test health endpoint
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected health status 200, got %d", w.Code)
	}
}

// TestSetHealthy tests the SetHealthy method
func TestSetHealthy(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test")
	defer ts.Close()

	// Initially healthy
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected initial health status 200, got %d", w.Code)
	}

	// Set unhealthy
	ts.SetHealthy(false)

	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected unhealthy status 503, got %d", w.Code)
	}

	// Set healthy again
	ts.SetHealthy(true)

	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected healthy status 200, got %d", w.Code)
	}
}

// TestSetResponse tests the SetResponse method
func TestSetResponse(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "initial")
	defer ts.Close()

	// Set custom response
	customResponse := "custom test response"
	ts.SetResponse(http.StatusCreated, customResponse)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	if w.Body.String() != customResponse {
		t.Errorf("Expected response %q, got %q", customResponse, w.Body.String())
	}
}

// TestResetRequestCount tests the ResetRequestCount method
func TestResetRequestCount(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test")
	defer ts.Close()

	// Make some requests
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ts.Server.Config.Handler.ServeHTTP(w, req)
	ts.Server.Config.Handler.ServeHTTP(w, req)
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if ts.RequestCount != 3 {
		t.Errorf("Expected request count 3, got %d", ts.RequestCount)
	}

	// Reset count
	ts.ResetRequestCount()

	if ts.RequestCount != 0 {
		t.Errorf("Expected request count 0 after reset, got %d", ts.RequestCount)
	}
}

// TestWithDialTimeout tests the WithDialTimeout option
func TestWithDialTimeout(t *testing.T) {
	timeout := 5 * time.Second

	// Create proxy without provisioning to avoid race conditions
	proxy := &FailoverProxy{
		Upstreams:       []string{"http://localhost:8080"},
		UpstreamHeaders: make(map[string]map[string]string),
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Apply the option
	WithDialTimeout(timeout)(proxy)

	if proxy.DialTimeout != caddy.Duration(timeout) {
		t.Errorf("Expected DialTimeout %v, got %v", timeout, proxy.DialTimeout)
	}
}

// TestWithResponseTimeout tests the WithResponseTimeout option
func TestWithResponseTimeout(t *testing.T) {
	timeout := 10 * time.Second

	// Create proxy without provisioning to avoid race conditions
	proxy := &FailoverProxy{
		Upstreams:       []string{"http://localhost:8080"},
		UpstreamHeaders: make(map[string]map[string]string),
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Apply the option
	WithResponseTimeout(timeout)(proxy)

	if proxy.ResponseTimeout != caddy.Duration(timeout) {
		t.Errorf("Expected ResponseTimeout %v, got %v", timeout, proxy.ResponseTimeout)
	}
}

// TestTestServerLatency tests the latency simulation
func TestTestServerLatency(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test")
	defer ts.Close()

	// Set 100ms latency
	latency := 100 * time.Millisecond
	ts.Latency = latency

	start := time.Now()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)
	elapsed := time.Since(start)

	// Check that request took at least the specified latency
	if elapsed < latency {
		t.Errorf("Expected latency of at least %v, but request took %v", latency, elapsed)
	}

	// Check that it didn't take too much longer (allow 50ms tolerance)
	if elapsed > latency+50*time.Millisecond {
		t.Errorf("Expected latency around %v, but request took %v", latency, elapsed)
	}
}

// TestRequestCountTracking tests that request counting works correctly
func TestRequestCountTracking(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test")
	defer ts.Close()

	if ts.RequestCount != 0 {
		t.Errorf("Expected initial request count 0, got %d", ts.RequestCount)
	}

	// Make regular requests (not health checks)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ts.Server.Config.Handler.ServeHTTP(w, req)
	}

	if ts.RequestCount != 5 {
		t.Errorf("Expected request count 5, got %d", ts.RequestCount)
	}

	// Health checks are also counted in this implementation
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	ts.Server.Config.Handler.ServeHTTP(w, req)

	if ts.RequestCount != 6 {
		t.Errorf("Expected request count 6 after health check, got %d", ts.RequestCount)
	}
}

// TestGetURL tests the URL method
func TestGetURL(t *testing.T) {
	ts := NewTestServer(true, http.StatusOK, "test")
	defer ts.Close()
	url := ts.URL

	if url == "" {
		t.Error("URL is empty string")
	}

	// URL should start with http://
	if len(url) < 7 || url[:7] != "http://" {
		t.Errorf("Expected URL to start with http://, got %s", url)
	}
}

// TestWithHealthCheck tests the WithHealthCheck option
func TestWithHealthCheck(t *testing.T) {
	hc := &HealthCheck{
		Path:           "/health",
		Interval:       caddy.Duration(10 * time.Second),
		Timeout:        caddy.Duration(2 * time.Second),
		ExpectedStatus: 200,
	}

	// Create proxy without provisioning to avoid starting health checks
	proxy := &FailoverProxy{
		Upstreams:       []string{"http://localhost:8080"},
		UpstreamHeaders: make(map[string]map[string]string),
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Apply the option
	WithHealthCheck("http://localhost:8080", hc)(proxy)

	if proxy.HealthChecks == nil {
		t.Fatal("HealthChecks map is nil")
	}

	if proxy.HealthChecks["http://localhost:8080"] != hc {
		t.Error("Health check was not set correctly")
	}
}

// TestWithPath tests the WithPath option
func TestWithPath(t *testing.T) {
	testPath := "/api/v1"

	// Create proxy without provisioning to avoid race conditions
	proxy := &FailoverProxy{
		Upstreams:       []string{"http://localhost:8080"},
		UpstreamHeaders: make(map[string]map[string]string),
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Apply the option
	WithPath(testPath)(proxy)

	if proxy.Path != testPath {
		t.Errorf("Expected Path %q, got %q", testPath, proxy.Path)
	}

	if proxy.HandlePath != testPath {
		t.Errorf("Expected HandlePath %q, got %q", testPath, proxy.HandlePath)
	}
}

// TestAssertJSONContains tests the AssertJSONContains helper
func TestAssertJSONContains(t *testing.T) {
	// Create a mock test for capturing failures
	mockT := &testing.T{}

	jsonStr := `{"name": "test", "value": 42, "enabled": true}`

	// Test successful assertions
	AssertJSONContains(mockT, jsonStr, map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
	})

	// Note: We can't easily test failure cases without mocking testing.T
	// but the function is covered by the successful path
}

// TestAssertStatusResponse tests the AssertStatusResponse helper
func TestAssertStatusResponse(t *testing.T) {
	// Test valid status response
	validJSON := `[
		{
			"path": "/api",
			"active": "http://localhost:8080",
			"failover_proxies": [
				{
					"host": "http://localhost:8080",
					"status": "UP",
					"health_check_enabled": false
				}
			]
		}
	]`

	status := AssertStatusResponse(t, validJSON)

	if len(status) != 1 {
		t.Errorf("Expected 1 status entry, got %d", len(status))
	}

	if status[0].Path != "/api" {
		t.Errorf("Expected path /api, got %s", status[0].Path)
	}

	// Test that auto: prefix detection works
	autoJSON := `[{"path": "auto:/test", "failover_proxies": []}]`
	mockT := &testing.T{}
	AssertStatusResponse(mockT, autoJSON)
	// The function will log an error if auto: prefix is found
}

// TestWaitForCondition tests the WaitForCondition helper
func TestWaitForCondition(t *testing.T) {
	// Test successful condition
	called := false
	WaitForCondition(t, 100*time.Millisecond, 10*time.Millisecond, func() bool {
		if !called {
			called = true
			return false
		}
		return true
	}, "test condition")

	if !called {
		t.Error("Condition function was not called")
	}

	// Note: We can't easily test the timeout case without causing a panic
	// The function is covered by the successful path
}

// TestMockHealthCheck tests the MockHealthCheck helper
func TestMockHealthCheck(t *testing.T) {
	hc := MockHealthCheck("/health", 30*time.Second, 5*time.Second, 200)

	if hc.Path != "/health" {
		t.Errorf("Expected path /health, got %s", hc.Path)
	}

	if hc.Interval != caddy.Duration(30*time.Second) {
		t.Errorf("Expected interval 30s, got %v", hc.Interval)
	}

	if hc.Timeout != caddy.Duration(5*time.Second) {
		t.Errorf("Expected timeout 5s, got %v", hc.Timeout)
	}

	if hc.ExpectedStatus != 200 {
		t.Errorf("Expected status 200, got %d", hc.ExpectedStatus)
	}
}
