//go:build !short
// +build !short

package caddyfailover

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// TestFailoverProxyIntegration tests the full integration of the failover proxy
func TestFailoverProxyIntegration(t *testing.T) {
	// Create mock upstream servers
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "primary server response")
	}))
	defer primary.Close()

	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "secondary server response")
	}))
	defer secondary.Close()

	// Start Caddy with our plugin configuration
	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2999
		}
		:9080 {
			failover_proxy %s %s {
				fail_duration 1s
			}
		}
	`, primary.URL, secondary.URL), "caddyfile")

	// Test that the proxy works
	resp, err := http.Get("http://localhost:9080/api/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestFailoverStatusEndpointIntegration tests the failover status endpoint
func TestFailoverStatusEndpointIntegration(t *testing.T) {
	// Create test servers
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream1")
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream2")
	}))
	defer upstream2.Close()

	// Start Caddy with our plugin configuration
	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2999
		}
		:9080 {
			handle /status {
				failover_status
			}

			handle {
				failover_proxy %s %s {
					status_path /api
					fail_duration 1s
					health_check %s {
						path /health
						interval 100ms
						timeout 50ms
					}
					health_check %s {
						path /health
						interval 100ms
						timeout 50ms
					}
				}
			}
		}
	`, upstream1.URL, upstream2.URL, upstream1.URL, upstream2.URL), "caddyfile")

	// Wait for health checks to run
	time.Sleep(200 * time.Millisecond)

	// Test the status endpoint
	resp, err := http.Get("http://localhost:9080/status")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}
