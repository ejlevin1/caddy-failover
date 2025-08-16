package caddyfailover

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func TestProxyRegistry(t *testing.T) {
	// Create a new registry
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Create test proxies
	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/auth/*",
		Path:       "/auth/*",
	}

	proxy2 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5041", "https://example.com"},
		HandlePath: "/admin/*",
		Path:       "/admin/*",
	}

	// Test duplicate registration with same path (should not duplicate)
	proxy3 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/auth/*",
		Path:       "/auth/*",
	}

	// Register proxies
	registry.Register("/auth/*", proxy1)
	registry.Register("/admin/*", proxy2)
	registry.Register("/auth/*", proxy3) // This should not create a duplicate

	// Test that we have exactly 2 entries (no duplicates)
	if len(registry.proxies) != 2 {
		t.Errorf("Expected 2 entries in registry, got %d", len(registry.proxies))
	}

	// Test order preservation
	if len(registry.order) != 2 {
		t.Errorf("Expected 2 entries in order, got %d", len(registry.order))
	}

	if registry.order[0] != "/auth/*" || registry.order[1] != "/admin/*" {
		t.Errorf("Order not preserved: %v", registry.order)
	}

	// Test GetStatus
	status := registry.GetStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}

	// Verify paths don't have "auto:" prefix
	for _, s := range status {
		if len(s.Path) > 5 && s.Path[:5] == "auto:" {
			t.Errorf("Path should not have 'auto:' prefix, got: %s", s.Path)
		}
	}

	// Test unregistration
	registry.Unregister("/auth/*", proxy1)
	if len(registry.proxies) != 1 {
		t.Errorf("Expected 1 entry after unregister, got %d", len(registry.proxies))
	}
}

func TestProxyRegistryNoDuplicateUpstreams(t *testing.T) {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Create proxies with overlapping upstreams
	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/api/*",
		Path:       "/api/*",
	}

	proxy2 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://backup.com"},
		HandlePath: "/api/*",
		Path:       "/api/*",
	}

	// Register both proxies with same path
	registry.Register("/api/*", proxy1)
	registry.Register("/api/*", proxy2)

	// Should have only one entry
	if len(registry.proxies) != 1 {
		t.Errorf("Expected 1 entry in registry, got %d", len(registry.proxies))
	}

	// Check that upstreams are tracked properly
	entry := registry.proxies["/api/*"]
	if entry == nil {
		t.Fatal("Entry not found for /api/*")
	}

	// Should have 3 unique upstreams total
	expectedUpstreams := map[string]bool{
		"http://localhost:5051": true,
		"https://example.com":   true,
		"https://backup.com":    true,
	}

	if len(entry.Upstreams) != len(expectedUpstreams) {
		t.Errorf("Expected %d unique upstreams, got %d", len(expectedUpstreams), len(entry.Upstreams))
	}

	for upstream := range expectedUpstreams {
		if !entry.Upstreams[upstream] {
			t.Errorf("Expected upstream %s to be tracked", upstream)
		}
	}
}

func TestProxyPathHandling(t *testing.T) {
	tests := []struct {
		name         string
		proxy        *FailoverProxy
		expectedPath string
	}{
		{
			name: "Path and HandlePath both set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "/auth/*",
				Path:       "/auth/*",
			},
			expectedPath: "/auth/*",
		},
		{
			name: "Only HandlePath set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "/api/*",
				Path:       "",
			},
			expectedPath: "/api/*",
		},
		{
			name: "Only Path set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "",
				Path:       "/admin/*",
			},
			expectedPath: "/admin/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &ProxyRegistry{
				proxies: make(map[string]*ProxyEntry),
				order:   make([]string, 0),
			}

			// Determine registration path (mimics Provision logic)
			registrationPath := tt.proxy.Path
			if registrationPath == "" && tt.proxy.HandlePath != "" {
				registrationPath = tt.proxy.HandlePath
			}

			if registrationPath != "" {
				registry.Register(registrationPath, tt.proxy)
			}

			status := registry.GetStatus()
			if len(status) != 1 {
				t.Fatalf("Expected 1 status entry, got %d", len(status))
			}

			if status[0].Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, status[0].Path)
			}

			// Ensure no "auto:" prefix
			if len(status[0].Path) > 5 && status[0].Path[:5] == "auto:" {
				t.Errorf("Path should not have 'auto:' prefix, got: %s", status[0].Path)
			}
		})
	}
}

// TestFailoverProxyServeHTTP tests the HTTP serving functionality
func TestFailoverProxyServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		setupServers   func() ([]*httptest.Server, []string)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful primary server",
			setupServers: func() ([]*httptest.Server, []string) {
				primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("primary response"))
				}))
				return []*httptest.Server{primary}, []string{primary.URL}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "primary response",
		},
		{
			name: "failover to secondary on primary failure",
			setupServers: func() ([]*httptest.Server, []string) {
				primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("secondary response"))
				}))
				return []*httptest.Server{primary, secondary}, []string{primary.URL, secondary.URL}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "secondary response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			servers, urls := tt.setupServers()
			for _, server := range servers {
				defer server.Close()
			}

			fp := &FailoverProxy{
				Upstreams:       urls,
				FailDuration:    caddy.Duration(1 * time.Second),
				DialTimeout:     caddy.Duration(1 * time.Second),
				ResponseTimeout: caddy.Duration(2 * time.Second),
			}

			ctx := caddy.Context{}
			if err := fp.Provision(ctx); err != nil {
				t.Fatalf("Failed to provision: %v", err)
			}
			defer fp.Cleanup()

			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			w := httptest.NewRecorder()

			err := fp.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				return nil
			}))

			if err != nil {
				t.Errorf("ServeHTTP returned error: %v", err)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestHealthCheckFunctionality tests health check behavior
func TestHealthCheckFunctionality(t *testing.T) {
	healthStatus := http.StatusOK
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(healthStatus)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer healthServer.Close()

	fp := &FailoverProxy{
		Upstreams:    []string{healthServer.URL},
		FailDuration: caddy.Duration(30 * time.Second),
		HealthChecks: map[string]*HealthCheck{
			healthServer.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(100 * time.Millisecond),
				Timeout:        caddy.Duration(50 * time.Millisecond),
				ExpectedStatus: 200,
			},
		},
		healthStatus:  make(map[string]bool),
		lastCheckTime: make(map[string]time.Time),
		responseTime:  make(map[string]int64),
		failureCache:  make(map[string]time.Time),
		shutdown:      make(chan struct{}),
		httpClient:    &http.Client{},
		httpsClient:   &http.Client{},
		logger:        zap.NewNop(),
	}

	// Test health check passes
	fp.performHealthCheck(healthServer.URL+"/health", healthServer.URL, fp.HealthChecks[healthServer.URL])
	if !fp.isHealthy(healthServer.URL) {
		t.Error("Expected server to be healthy")
	}

	// Test health check fails
	healthStatus = http.StatusServiceUnavailable
	fp.performHealthCheck(healthServer.URL+"/health", healthServer.URL, fp.HealthChecks[healthServer.URL])
	if fp.isHealthy(healthServer.URL) {
		t.Error("Expected server to be unhealthy")
	}
}

// TestUnmarshalCaddyfile tests Caddyfile parsing
func TestUnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		name         string
		caddyfile    string
		expectError  bool
		validateFunc func(*testing.T, *FailoverProxy)
	}{
		{
			name: "basic configuration",
			caddyfile: `failover_proxy http://localhost:5051 https://backup.com {
				fail_duration 5s
				insecure_skip_verify
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, fp *FailoverProxy) {
				if len(fp.Upstreams) != 2 {
					t.Errorf("Expected 2 upstreams, got %d", len(fp.Upstreams))
				}
				if fp.FailDuration != caddy.Duration(5*time.Second) {
					t.Errorf("Expected fail_duration of 5s, got %v", fp.FailDuration)
				}
				if !fp.InsecureSkipVerify {
					t.Error("Expected insecure_skip_verify to be true")
				}
			},
		},
		{
			name:        "missing upstreams",
			caddyfile:   `failover_proxy`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := caddyfile.NewTestDispenser(tt.caddyfile)
			fp := &FailoverProxy{}
			err := fp.UnmarshalCaddyfile(d)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && err == nil {
				tt.validateFunc(t, fp)
			}
		})
	}
}

// TestFailoverStatusHandler tests the status endpoint handler
func TestFailoverStatusHandler(t *testing.T) {
	// Setup registry with test data
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051"},
		HandlePath: "/api/*",
		Path:       "/api/*",
	}
	registry.Register("/api/*", proxy1)

	// Replace global registry temporarily
	oldRegistry := proxyRegistry
	proxyRegistry = registry
	defer func() { proxyRegistry = oldRegistry }()

	handler := FailoverStatusHandler{}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request succeeds",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request fails",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/admin/failover/status", nil)
			w := httptest.NewRecorder()

			err := handler.ServeHTTP(w, req, nil)
			if err != nil {
				t.Errorf("ServeHTTP returned error: %v", err)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == http.MethodGet && w.Header().Get("Content-Type") != "application/json" {
				t.Error("Expected Content-Type to be application/json")
			}
		})
	}
}
