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
			http_port 9080
			https_port 9443
		}

		localhost:9080 {
			handle /api/* {
				failover_proxy %s %s {
					fail_duration 2s
					dial_timeout 1s
					response_timeout 2s
				}
			}

			handle /admin/failover/status {
				failover_status
			}

			handle {
				respond "OK" 200
			}
		}
	`, primary.URL, secondary.URL), "caddyfile")
	// Test 1: Normal request should hit primary
	tester.AssertGetResponse("http://localhost:9080/api/test", 200, "primary server response")

	// Test 2: Status endpoint should return JSON
	resp, body := tester.AssertGetResponse("http://localhost:9080/admin/failover/status", 200, "")
	if resp.StatusCode != 200 {
		t.Error("Expected status 200 from status endpoint")
	}
	if body == "" {
		t.Error("Expected non-empty response from status endpoint")
	}
}

// TestFailoverWithUnhealthyPrimary tests failover when primary is unhealthy
func TestFailoverWithUnhealthyPrimary(t *testing.T) {
	var primaryHealthy = true

	// Create mock upstream servers with controllable health
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if primaryHealthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		if primaryHealthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "primary server")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer primary.Close()

	secondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "secondary server")
	}))
	defer secondary.Close()

	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			admin localhost:2998
			http_port 9081
		}

		localhost:9081 {
			handle /api/* {
				failover_proxy %s %s {
					fail_duration 1s

					health_check %s {
						path /health
						interval 500ms
						timeout 200ms
						expected_status 200
					}

					health_check %s {
						path /health
						interval 500ms
						timeout 200ms
						expected_status 200
					}
				}
			}
		}
	`, primary.URL, secondary.URL, primary.URL, secondary.URL), "caddyfile")

	// Initially should hit primary
	tester.AssertGetResponse("http://localhost:9081/api/test", 200, "primary server")

	// Make primary unhealthy
	primaryHealthy = false

	// Wait for health check to detect the failure
	time.Sleep(1 * time.Second)

	// Now should failover to secondary
	tester.AssertGetResponse("http://localhost:9081/api/test", 200, "secondary server")
}

// TestMultiplePathsWithStatus tests multiple paths registered with status endpoint
func TestMultiplePathsWithStatus(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "response for %s", r.URL.Path)
	}))
	defer upstream.Close()

	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2997
			http_port 9082
		}

		localhost:9082 {
			handle /auth/* {
				failover_proxy %s {
					fail_duration 3s
					status_path /auth/*
				}
			}

			handle /admin/* {
				failover_proxy %s {
					fail_duration 3s
					status_path /admin/*
				}
			}

			handle /api/* {
				failover_proxy %s {
					fail_duration 3s
				}
			}

			handle /status {
				failover_status
			}
		}
	`, upstream.URL, upstream.URL, upstream.URL), "caddyfile")

	// Test each path works
	tester.AssertGetResponse("http://localhost:9082/auth/login", 200, "response for /auth/login")
	tester.AssertGetResponse("http://localhost:9082/admin/users", 200, "response for /admin/users")
	tester.AssertGetResponse("http://localhost:9082/api/data", 200, "response for /api/data")

	// Test status endpoint returns all paths
	_, statusBody := tester.AssertGetResponse("http://localhost:9082/status", 200, "")

	// Verify paths are present and correctly formatted
	expectedPaths := []string{`"/auth/*"`, `"/admin/*"`, `"/api/*"`}
	for _, path := range expectedPaths {
		if !contains(statusBody, path) {
			t.Errorf("Expected path %s not found in status response", path)
		}
	}

	// Verify no auto: prefix
	if contains(statusBody, "auto:") {
		t.Error("Found 'auto:' prefix in status response, this should not happen")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
