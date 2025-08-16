package caddyfailover

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// BenchmarkFailoverProxy benchmarks the failover proxy performance
func BenchmarkFailoverProxy(b *testing.B) {
	// Create a test server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response")
	}))
	defer upstream.Close()

	// Create failover proxy
	fp := &FailoverProxy{
		Upstreams:       []string{upstream.URL},
		FailDuration:    caddy.Duration(30 * time.Second),
		DialTimeout:     caddy.Duration(2 * time.Second),
		ResponseTimeout: caddy.Duration(5 * time.Second),
	}

	// Provision the proxy
	ctx := caddy.Context{}
	if err := fp.Provision(ctx); err != nil {
		b.Fatalf("Failed to provision: %v", err)
	}
	defer fp.Cleanup()

	// Create test request and response recorder
	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			err := fp.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				return nil
			}))
			if err != nil {
				b.Errorf("ServeHTTP error: %v", err)
			}
		}
	})
}

// BenchmarkProxyRegistryRegister benchmarks the registry registration
func BenchmarkProxyRegistryRegister(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			registry := &ProxyRegistry{
				proxies: make(map[string]*ProxyEntry),
				order:   make([]string, 0),
			}

			// Register many proxies
			for i := 0; i < 100; i++ {
				path := fmt.Sprintf("/path%d/*", i)
				proxy := &FailoverProxy{
					Upstreams:  []string{fmt.Sprintf("http://localhost:%d", 5000+i)},
					HandlePath: path,
					Path:       path,
				}
				registry.Register(path, proxy)
			}
		}
	})
}

// BenchmarkProxyRegistryGetStatus benchmarks status retrieval
func BenchmarkProxyRegistryGetStatus(b *testing.B) {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Pre-populate registry
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/path%d/*", i)
		proxy := &FailoverProxy{
			Upstreams:  []string{fmt.Sprintf("http://localhost:%d", 5000+i)},
			HandlePath: path,
			Path:       path,
		}
		registry.Register(path, proxy)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			status := registry.GetStatus()
			if len(status) != 100 {
				b.Errorf("Expected 100 status entries, got %d", len(status))
			}
		}
	})
}

// BenchmarkFailoverWithMultipleUpstreams benchmarks failover with multiple upstreams
func BenchmarkFailoverWithMultipleUpstreams(b *testing.B) {
	// Create multiple test servers
	servers := make([]*httptest.Server, 5)
	upstreams := make([]string, 5)

	for i := 0; i < 5; i++ {
		idx := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if idx == 0 {
				// First server always fails to test failover
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "server %d", idx)
			}
		}))
		defer servers[i].Close()
		upstreams[i] = servers[i].URL
	}

	// Create failover proxy with multiple upstreams
	fp := &FailoverProxy{
		Upstreams:       upstreams,
		FailDuration:    caddy.Duration(1 * time.Second),
		DialTimeout:     caddy.Duration(100 * time.Millisecond),
		ResponseTimeout: caddy.Duration(200 * time.Millisecond),
	}

	// Provision the proxy
	ctx := caddy.Context{}
	if err := fp.Provision(ctx); err != nil {
		b.Fatalf("Failed to provision: %v", err)
	}
	defer fp.Cleanup()

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		err := fp.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			return nil
		}))
		if err != nil {
			b.Errorf("ServeHTTP error: %v", err)
		}
		if w.Code != http.StatusOK {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkHealthCheck benchmarks health check performance
func BenchmarkHealthCheck(b *testing.B) {
	// Create a test server for health checks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fp := &FailoverProxy{
		Upstreams:    []string{server.URL},
		FailDuration: caddy.Duration(30 * time.Second),
		HealthChecks: map[string]*HealthCheck{
			server.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(1 * time.Second),
				Timeout:        caddy.Duration(500 * time.Millisecond),
				ExpectedStatus: 200,
			},
		},
	}

	// Provision the proxy
	ctx := caddy.Context{}
	if err := fp.Provision(ctx); err != nil {
		b.Fatalf("Failed to provision: %v", err)
	}
	defer fp.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fp.performHealthCheck(server.URL+"/health", server.URL, fp.HealthChecks[server.URL])
	}
}
