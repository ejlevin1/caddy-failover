//go:build !short
// +build !short

package caddyfailover

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// getAvailablePort finds an available port on the local system
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// TestFailoverProxyIntegration tests the full integration of the failover proxy
func TestFailoverProxyIntegration(t *testing.T) {
	// Get an available port
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

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
		:%d {
			failover_proxy %s %s {
				fail_duration 1s
			}
		}
	`, port, primary.URL, secondary.URL), "caddyfile")

	// Test that the proxy works
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/test", port))
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
	// Get an available port
	port, err := getAvailablePort()
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}

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
		:%d {
			handle /status {
				failover_status
			}

			handle /api {
				failover_proxy %s %s {
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
	`, port, upstream1.URL, upstream2.URL, upstream1.URL, upstream2.URL), "caddyfile")

	// Wait for health checks to run
	time.Sleep(200 * time.Millisecond)

	// Test the status endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/status", port))
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
