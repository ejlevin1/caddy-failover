package failover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDynamicRoutesWithFailoverProxy tests that failover_proxy continues to work
// correctly when routes are dynamically added/removed via the Caddy Admin API.
// This reproduces the issue where dynamic route changes cause failover_proxy to malfunction.
func TestDynamicRoutesWithFailoverProxy(t *testing.T) {
	// Start Caddy with failover_proxy configuration
	caddyConfig := `
	{
		admin localhost:9999
		http_port 9080
		https_port 9443
		order failover_proxy before reverse_proxy
		order failover_status before respond
	}

	http://localhost:9080 {
		# Static failover_proxy route
		handle /api/* {
			failover_proxy http://localhost:9081 http://localhost:9082 {
				path /api/
				fail_duration 3s
				dial_timeout 2s
				response_timeout 5s
			}
		}

		# Another static failover_proxy route
		handle /service/* {
			failover_proxy http://localhost:9083 {
				path /service/
				fail_duration 3s
			}
		}

		# Failover status endpoint
		handle /status {
			failover_status
		}

		# Default response
		respond "OK" 200
	}
	`

	// Start test servers to act as upstreams
	upstream1 := startTestServer(t, 9081, "upstream1")
	defer upstream1.Close()
	upstream2 := startTestServer(t, 9082, "upstream2")
	defer upstream2.Close()
	upstream3 := startTestServer(t, 9083, "upstream3")
	defer upstream3.Close()

	// Start Caddy
	tester := caddytest.NewTester(t)
	tester.InitServer(caddyConfig, "caddyfile")

	// Helper function to check failover status
	checkFailoverStatus := func(testName string) bool {
		resp, err := http.Get("http://localhost:9080/status")
		if err != nil {
			t.Logf("❌ %s: Failed to get status: %v", testName, err)
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("❌ %s: Status endpoint returned %d", testName, resp.StatusCode)
			return false
		}

		var status []PathStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Logf("❌ %s: Failed to decode status: %v", testName, err)
			return false
		}

		// Check that we got a valid response (not null)
		if status == nil {
			t.Errorf("❌ %s: Status endpoint returned null!", testName)
			return false
		}

		t.Logf("✅ %s: Status endpoint returned valid list with %d entries", testName, len(status))

		// Verify expected paths are present
		expectedPaths := map[string]bool{
			"/api/":     false,
			"/service/": false,
		}

		for _, ps := range status {
			if _, ok := expectedPaths[ps.Path]; ok {
				expectedPaths[ps.Path] = true
			}
		}

		for path, found := range expectedPaths {
			if !found {
				t.Logf("⚠️  %s: Expected path %s not found in status", testName, path)
			}
		}

		return true
	}

	// Helper function to test failover proxy
	testFailoverProxy := func(path string, testName string) bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:9080%s", path))
		if err != nil {
			t.Logf("❌ %s: Failed to access %s: %v", testName, path, err)
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			t.Logf("❌ %s: Path %s returned error %d", testName, path, resp.StatusCode)
			return false
		}

		t.Logf("✅ %s: Path %s responded with %d", testName, path, resp.StatusCode)
		return true
	}

	// Test 1: Initial state - everything should work
	t.Run("InitialState", func(t *testing.T) {
		assert.True(t, checkFailoverStatus("Initial status check"))
		assert.True(t, testFailoverProxy("/api/test", "Initial failover proxy"))
		assert.True(t, testFailoverProxy("/service/health", "Initial service proxy"))
	})

	// Test 2: Add dynamic routes via Admin API
	t.Run("AfterAddingDynamicRoutes", func(t *testing.T) {
		// Add dynamic routes
		dynamicRoutes := []map[string]interface{}{
			{
				"@id": "dynamic-route-1",
				"match": []map[string]interface{}{
					{"path": []string{"/dynamic/route1/*"}},
				},
				"handle": []map[string]interface{}{
					{
						"handler": "static_response",
						"body":    "Dynamic Route 1",
					},
				},
			},
			{
				"@id": "dynamic-route-2",
				"match": []map[string]interface{}{
					{"path": []string{"/dynamic/route2/*"}},
				},
				"handle": []map[string]interface{}{
					{
						"handler": "static_response",
						"body":    "Dynamic Route 2",
					},
				},
			},
		}

		// Add routes via PATCH
		for _, route := range dynamicRoutes {
			routeJSON, err := json.Marshal(route)
			require.NoError(t, err)

			req, err := http.NewRequest("PATCH",
				"http://localhost:9999/config/apps/http/servers/srv0/routes",
				bytes.NewBuffer(routeJSON))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Logf("Warning: Failed to add route %s: status %d", route["@id"], resp.StatusCode)
			} else {
				t.Logf("Added dynamic route: %s", route["@id"])
			}
		}

		// Give Caddy time to reconfigure
		time.Sleep(500 * time.Millisecond)

		// Critical tests: failover_proxy should still work after dynamic routes
		assert.True(t, checkFailoverStatus("Status after dynamic routes"),
			"CRITICAL: Status endpoint should NOT return null after dynamic routes")
		assert.True(t, testFailoverProxy("/api/test", "Failover proxy after dynamic routes"),
			"CRITICAL: Failover proxy should still work after dynamic routes")
		assert.True(t, testFailoverProxy("/service/health", "Service proxy after dynamic routes"))

		// Test that dynamic routes work too
		resp, err := http.Get("http://localhost:9080/dynamic/route1/test")
		if err == nil {
			defer resp.Body.Close()
			t.Logf("Dynamic route responded with status %d", resp.StatusCode)
		}
	})

	// Test 3: Remove dynamic routes
	t.Run("AfterRemovingDynamicRoutes", func(t *testing.T) {
		// Get current routes to find indices to delete
		resp, err := http.Get("http://localhost:9999/config/apps/http/servers/srv0/routes")
		require.NoError(t, err)
		defer resp.Body.Close()

		var routes []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&routes)
		require.NoError(t, err)

		// Find and delete dynamic routes
		routesToDelete := []string{"dynamic-route-1", "dynamic-route-2"}
		for _, routeID := range routesToDelete {
			for i, route := range routes {
				if id, ok := route["@id"].(string); ok && id == routeID {
					// Delete the route
					req, err := http.NewRequest("DELETE",
						fmt.Sprintf("http://localhost:9999/config/apps/http/servers/srv0/routes/%d", i),
						nil)
					if err == nil {
						resp, err := http.DefaultClient.Do(req)
						if err == nil {
							resp.Body.Close()
							t.Logf("Removed dynamic route: %s", routeID)
						}
					}
					break
				}
			}
		}

		// Give Caddy time to reconfigure
		time.Sleep(500 * time.Millisecond)

		// Test that everything still works after removal
		assert.True(t, checkFailoverStatus("Status after route removal"))
		assert.True(t, testFailoverProxy("/api/test", "Failover proxy after route removal"))
		assert.True(t, testFailoverProxy("/service/health", "Service proxy after route removal"))
	})
}

// TestFailoverStatusNullSafety specifically tests that the status endpoint never returns null
func TestFailoverStatusNullSafety(t *testing.T) {
	// Test with no registered proxies
	handler := FailoverStatusHandler{}

	// Create a test request
	req, err := http.NewRequest("GET", "/status", nil)
	require.NoError(t, err)

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	err = handler.ServeHTTP(rr, req, nil)
	require.NoError(t, err)

	// Check that we got a valid JSON response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	// Decode the response
	var status []PathStatus
	err = json.NewDecoder(rr.Body).Decode(&status)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify it's not null (should be empty array)
	assert.NotNil(t, status, "Status should never be null")
	assert.IsType(t, []PathStatus{}, status, "Status should be a slice")

	t.Logf("✅ Status endpoint returned empty array (not null) when no proxies registered")
}

// TestProxyRegistryReplacement tests that the registry correctly handles proxy replacement
func TestProxyRegistryReplacement(t *testing.T) {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Create first proxy
	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:8001"},
		HandlePath: "/api/",
	}

	// Register it
	registry.Register("/api/", proxy1)

	// Verify it's registered
	status := registry.GetStatus()
	assert.Len(t, status, 1)
	assert.Equal(t, "/api/", status[0].Path)

	// Create second proxy with same path (simulating re-provisioning)
	proxy2 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:8002", "http://localhost:8003"},
		HandlePath: "/api/",
	}

	// Register it (should replace the first one)
	registry.Register("/api/", proxy2)

	// Verify the replacement worked
	status = registry.GetStatus()
	assert.Len(t, status, 1, "Should still have only one entry")
	assert.Equal(t, "/api/", status[0].Path)

	// Verify it's using the new proxy's upstreams
	entry := registry.proxies["/api/"]
	assert.Equal(t, proxy2, entry.Proxy, "Should have replaced with new proxy")
	assert.Len(t, entry.Upstreams, 2, "Should have new proxy's upstreams")

	t.Logf("✅ Registry correctly replaced proxy when re-registered with same path")
}

// TestProxyRegistryCleanup tests that stale entries are cleaned up
func TestProxyRegistryCleanup(t *testing.T) {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Add a valid entry
	proxy := &FailoverProxy{
		Upstreams:  []string{"http://localhost:8001"},
		HandlePath: "/valid/",
	}
	registry.Register("/valid/", proxy)

	// Manually add a stale entry (nil proxy)
	registry.proxies["/stale/"] = &ProxyEntry{
		Path:  "/stale/",
		Proxy: nil, // Simulating a stale entry
	}
	registry.order = append(registry.order, "/stale/")

	// Get status (which should trigger cleanup)
	status := registry.GetStatus()

	// Should only return the valid entry
	assert.Len(t, status, 1, "Should only return valid entries")
	assert.Equal(t, "/valid/", status[0].Path)

	// Verify stale entry was removed
	_, exists := registry.proxies["/stale/"]
	assert.False(t, exists, "Stale entry should be removed")

	t.Logf("✅ Registry correctly cleaned up stale entries")
}

// startTestServer starts a simple HTTP server for testing
func startTestServer(t *testing.T, port int, name string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server", name)
		fmt.Fprintf(w, "Response from %s", name)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Test server %s error: %v", name, err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return server
}
