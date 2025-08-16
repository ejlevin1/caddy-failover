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

// TestStatusEndpointWithMultipleProxies tests the status endpoint returns correct JSON
func TestStatusEndpointWithMultipleProxies(t *testing.T) {
	// Create mock upstream servers with health endpoints
	authPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "auth primary response")
	}))
	defer authPrimary.Close()

	authSecondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "auth secondary response")
	}))
	defer authSecondary.Close()

	apiPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "api primary response")
	}))
	defer apiPrimary.Close()

	apiSecondary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate unhealthy secondary
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiSecondary.Close()

	// Start Caddy with multiple failover proxy configurations
	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2996
			http_port 9083
			https_port 9443
		}

		localhost:9083 {
			# Authentication endpoints
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

			# API endpoints
			handle /api/* {
				failover_proxy %s %s {
					fail_duration 3s
					status_path /api/*

					health_check %s {
						path /api/health
						interval 1s
						timeout 500ms
						expected_status 200
					}

					health_check %s {
						path /api/health
						interval 1s
						timeout 500ms
						expected_status 200
					}
				}
			}

			# Admin endpoints without health checks
			handle /admin/* {
				failover_proxy %s {
					fail_duration 10s
					status_path /admin/*
				}
			}

			# Status endpoint
			handle /failover/status {
				failover_status
			}

			handle {
				respond "Not Found" 404
			}
		}
	`, authPrimary.URL, authSecondary.URL, authPrimary.URL, authSecondary.URL,
		apiPrimary.URL, apiSecondary.URL, apiPrimary.URL, apiSecondary.URL,
		authPrimary.URL), "caddyfile")

	// Wait for health checks to run
	time.Sleep(2 * time.Second)

	// Test the status endpoint
	req, _ := http.NewRequest("GET", "http://localhost:9083/failover/status", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Parse JSON response
	var status []PathStatus
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, string(body))
	}

	// Pretty print the JSON response
	prettyJSON, err := json.MarshalIndent(status, "", "  ")
	if err == nil {
		t.Log("\n========================================")
		t.Log("STATUS ENDPOINT JSON RESPONSE:")
		t.Log("========================================")
		t.Log(string(prettyJSON))
		t.Log("========================================\n")
	}

	// Also print formatted summary
	t.Logf("Status endpoint returned %d paths:", len(status))
	for _, ps := range status {
		t.Logf("  Path: %s", ps.Path)
		for _, fp := range ps.FailoverProxies {
			healthCheckStr := "disabled"
			if fp.HealthCheck {
				healthCheckStr = "enabled"
			}
			t.Logf("    - %s: %s (health check: %s)", fp.Host, fp.Status, healthCheckStr)
		}
	}

	// Validate the response structure
	if len(status) != 3 {
		t.Errorf("Expected 3 paths in status, got %d", len(status))
	}

	// Create a map for easier validation
	pathMap := make(map[string]*PathStatus)
	for i := range status {
		pathMap[status[i].Path] = &status[i]
	}

	// Validate /auth/* path
	if authStatus, ok := pathMap["/auth/*"]; ok {
		if len(authStatus.FailoverProxies) != 2 {
			t.Errorf("Expected 2 upstreams for /auth/*, got %d", len(authStatus.FailoverProxies))
		}
		// Both auth upstreams should be healthy
		for _, fp := range authStatus.FailoverProxies {
			if fp.Status != "UP" {
				t.Errorf("Expected auth upstream %s to be UP, got %s", fp.Host, fp.Status)
			}
			if !fp.HealthCheck {
				t.Errorf("Expected health check to be enabled for auth upstream %s", fp.Host)
			}
		}
	} else {
		t.Error("Path /auth/* not found in status response")
	}

	// Validate /api/* path
	if apiStatus, ok := pathMap["/api/*"]; ok {
		if len(apiStatus.FailoverProxies) != 2 {
			t.Errorf("Expected 2 upstreams for /api/*, got %d", len(apiStatus.FailoverProxies))
		}
		// Primary should be UP, secondary should be DOWN
		primaryFound := false
		secondaryFound := false
		for _, fp := range apiStatus.FailoverProxies {
			if fp.Host == apiPrimary.URL {
				primaryFound = true
				if fp.Status != "UP" {
					t.Errorf("Expected API primary to be UP, got %s", fp.Status)
				}
			} else if fp.Host == apiSecondary.URL {
				secondaryFound = true
				// Secondary is unhealthy, should be DOWN
				if fp.Status != "DOWN" {
					t.Errorf("Expected API secondary to be DOWN, got %s", fp.Status)
				}
			}
		}
		if !primaryFound || !secondaryFound {
			t.Error("Not all API upstreams found in status")
		}
	} else {
		t.Error("Path /api/* not found in status response")
	}

	// Validate /admin/* path
	if adminStatus, ok := pathMap["/admin/*"]; ok {
		if len(adminStatus.FailoverProxies) != 1 {
			t.Errorf("Expected 1 upstream for /admin/*, got %d", len(adminStatus.FailoverProxies))
		}
		// Admin has no health checks
		if len(adminStatus.FailoverProxies) > 0 {
			fp := adminStatus.FailoverProxies[0]
			if fp.HealthCheck {
				t.Errorf("Expected health check to be disabled for admin upstream %s", fp.Host)
			}
			// Without health checks, should default to UP
			if fp.Status != "UP" {
				t.Errorf("Expected admin upstream to be UP (no health check), got %s", fp.Status)
			}
		}
	} else {
		t.Error("Path /admin/* not found in status response")
	}

	// Ensure no "auto:" prefix in any path
	for _, ps := range status {
		if len(ps.Path) > 5 && ps.Path[:5] == "auto:" {
			t.Errorf("Path should not have 'auto:' prefix, got: %s", ps.Path)
		}
	}
}

// TestStatusEndpointConcurrentAccess tests that the status endpoint handles concurrent requests
func TestStatusEndpointConcurrentAccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	tester := caddytest.NewTester(t)
	tester.InitServer(fmt.Sprintf(`
		{
			order failover_proxy before reverse_proxy
			order failover_status before respond
			admin localhost:2995
			http_port 9084
		}

		localhost:9084 {
			handle /api/* {
				failover_proxy %s {
					fail_duration 3s
				}
			}

			handle /status {
				failover_status
			}
		}
	`, upstream.URL), "caddyfile")

	// Make concurrent requests to the status endpoint
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req, _ := http.NewRequest("GET", "http://localhost:9084/status", nil)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Goroutine %d: Failed to get status: %v", id, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Goroutine %d: Expected status 200, got %d", id, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			var status []PathStatus
			if err := json.Unmarshal(body, &status); err != nil {
				t.Errorf("Goroutine %d: Failed to parse JSON: %v", id, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}
