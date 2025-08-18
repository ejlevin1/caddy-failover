package failover

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

// TestServer represents a test HTTP server with controllable behavior
type TestServer struct {
	*httptest.Server
	Healthy      bool
	ResponseCode int
	ResponseBody string
	Latency      time.Duration
	RequestCount int
}

// NewTestServer creates a new test server with customizable behavior
func NewTestServer(healthy bool, responseCode int, responseBody string) *TestServer {
	ts := &TestServer{
		Healthy:      healthy,
		ResponseCode: responseCode,
		ResponseBody: responseBody,
	}

	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.RequestCount++

		if ts.Latency > 0 {
			time.Sleep(ts.Latency)
		}

		if r.URL.Path == "/health" {
			if ts.Healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}

		w.WriteHeader(ts.ResponseCode)
		if ts.ResponseBody != "" {
			fmt.Fprint(w, ts.ResponseBody)
		}
	}))

	return ts
}

// SetHealthy updates the health status of the test server
func (ts *TestServer) SetHealthy(healthy bool) {
	ts.Healthy = healthy
}

// SetResponse updates the response configuration
func (ts *TestServer) SetResponse(code int, body string) {
	ts.ResponseCode = code
	ts.ResponseBody = body
}

// ResetRequestCount resets the request counter
func (ts *TestServer) ResetRequestCount() {
	ts.RequestCount = 0
}

// CreateTestProxy creates a properly configured FailoverProxy for testing
func CreateTestProxy(t *testing.T, upstreams []string, opts ...ProxyOption) *FailoverProxy {
	fp := &FailoverProxy{
		Upstreams:       upstreams,
		FailDuration:    caddy.Duration(30 * time.Second),
		DialTimeout:     caddy.Duration(2 * time.Second),
		ResponseTimeout: caddy.Duration(5 * time.Second),
		UpstreamHeaders: make(map[string]map[string]string),
		HealthChecks:    make(map[string]*HealthCheck),
	}

	// Apply options
	for _, opt := range opts {
		opt(fp)
	}

	// Provision the proxy
	ctx := caddy.Context{}

	if err := fp.Provision(ctx); err != nil {
		t.Fatalf("Failed to provision proxy: %v", err)
	}

	t.Cleanup(func() {
		if err := fp.Cleanup(); err != nil {
			t.Errorf("Failed to cleanup proxy: %v", err)
		}
	})

	return fp
}

// ProxyOption is a functional option for configuring test proxies
type ProxyOption func(*FailoverProxy)

// WithFailDuration sets the fail duration
func WithFailDuration(d time.Duration) ProxyOption {
	return func(fp *FailoverProxy) {
		fp.FailDuration = caddy.Duration(d)
	}
}

// WithDialTimeout sets the dial timeout
func WithDialTimeout(d time.Duration) ProxyOption {
	return func(fp *FailoverProxy) {
		fp.DialTimeout = caddy.Duration(d)
	}
}

// WithResponseTimeout sets the response timeout
func WithResponseTimeout(d time.Duration) ProxyOption {
	return func(fp *FailoverProxy) {
		fp.ResponseTimeout = caddy.Duration(d)
	}
}

// WithHealthCheck adds a health check for an upstream
func WithHealthCheck(upstream string, hc *HealthCheck) ProxyOption {
	return func(fp *FailoverProxy) {
		if fp.HealthChecks == nil {
			fp.HealthChecks = make(map[string]*HealthCheck)
		}
		fp.HealthChecks[upstream] = hc
	}
}

// WithPath sets the path for the proxy
func WithPath(path string) ProxyOption {
	return func(fp *FailoverProxy) {
		fp.Path = path
		fp.HandlePath = path
	}
}

// AssertJSONContains checks if a JSON response contains expected fields
func AssertJSONContains(t *testing.T, jsonStr string, expectedFields map[string]interface{}) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	for key, expectedValue := range expectedFields {
		actualValue, exists := data[key]
		if !exists {
			t.Errorf("Expected field %q not found in JSON", key)
			continue
		}

		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			t.Errorf("Field %q: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

// AssertStatusResponse validates a status endpoint response
func AssertStatusResponse(t *testing.T, response string) []PathStatus {
	var status []PathStatus
	if err := json.Unmarshal([]byte(response), &status); err != nil {
		t.Fatalf("Failed to parse status response: %v", err)
	}

	// Check for auto: prefix
	for _, s := range status {
		if len(s.Path) > 5 && s.Path[:5] == "auto:" {
			t.Errorf("Path should not have 'auto:' prefix: %s", s.Path)
		}
	}

	return status
}

// CreateTestRegistry creates a test registry with sample data
func CreateTestRegistry(paths ...string) *ProxyRegistry {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	for i, path := range paths {
		proxy := &FailoverProxy{
			Upstreams:  []string{fmt.Sprintf("http://localhost:%d", 5000+i)},
			HandlePath: path,
			Path:       path,
		}
		registry.Register(path, proxy)
	}

	return registry
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, timeout time.Duration, interval time.Duration, condition func() bool, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// MockHealthCheck creates a mock health check configuration
func MockHealthCheck(path string, interval, timeout time.Duration, expectedStatus int) *HealthCheck {
	return &HealthCheck{
		Path:           path,
		Interval:       caddy.Duration(interval),
		Timeout:        caddy.Duration(timeout),
		ExpectedStatus: expectedStatus,
	}
}
