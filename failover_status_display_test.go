package caddyfailover

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

// TestDisplayStatusEndpoint is a simple test that shows the status endpoint output
func TestDisplayStatusEndpoint(t *testing.T) {
	// Create mock upstream servers
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream1 response")
	}))
	defer upstream1.Close()

	upstream2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			// Simulate unhealthy upstream
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream2 response")
	}))
	defer upstream2.Close()

	upstream3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream3 response")
	}))
	defer upstream3.Close()

	// Start Caddy with failover configuration
	tester := caddytest.NewTester(t)
	caddyConfig := fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2993
			http_port 9086
		}

		localhost:9086 {
			# Authentication endpoints with health checks
			handle /auth/* {
				failover_proxy %s %s {
					fail_duration 5s
					status_path /auth/*

					health_check %s {
						path /health
						interval 1s
						timeout 500ms
						expected_status 200
					}

					health_check %s {
						path /health
						interval 1s
						timeout 500ms
						expected_status 200
					}
				}
			}

			# API endpoints without health checks
			handle /api/* {
				failover_proxy %s %s {
					fail_duration 3s
					status_path /api/*
				}
			}

			# Admin endpoint with single upstream
			handle /admin/* {
				failover_proxy %s {
					fail_duration 10s
					status_path /admin/*
				}
			}

			# Status endpoint
			handle /status {
				failover_status
			}

			# Default handler
			handle {
				respond "Not Found" 404
			}
		}
	`, upstream1.URL, upstream2.URL, upstream1.URL, upstream2.URL,
		upstream1.URL, upstream3.URL,
		upstream3.URL)

	tester.InitServer(caddyConfig, "caddyfile")

	// Wait for health checks to run
	t.Log("Waiting for health checks to initialize...")
	time.Sleep(2 * time.Second)

	// Make request to status endpoint
	req, _ := http.NewRequest("GET", "http://localhost:9086/status", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Parse JSON for pretty printing
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to format JSON: %v", err)
	}

	// Display the output
	fmt.Println("\n========================================")
	fmt.Println("FAILOVER STATUS ENDPOINT OUTPUT:")
	fmt.Println("========================================")
	fmt.Println(string(prettyJSON))
	fmt.Println("========================================")
	fmt.Println("\nEXPLANATION:")
	fmt.Println("- Each path shows the configured route pattern")
	fmt.Println("- 'active' shows the currently selected upstream")
	fmt.Println("- 'failover_proxies' lists all upstreams with:")
	fmt.Println("  - host: The upstream URL")
	fmt.Println("  - status: UP, DOWN, or UNHEALTHY")
	fmt.Println("  - health_check_enabled: Whether health checks are configured")
	fmt.Println("  - last_check: When the last health check occurred")
	fmt.Println("========================================")

	// Basic validation
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
