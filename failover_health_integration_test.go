package caddyfailover

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

// TestHealthCheckFailoverIntegration tests the complete health check and failover behavior
func TestHealthCheckFailoverIntegration(t *testing.T) {
	// Create test servers with dynamic ports
	var primaryHealthy atomic.Bool
	var secondaryHealthy atomic.Bool
	var tertiaryHealthy atomic.Bool

	primaryHealthy.Store(false)  // Start unhealthy
	secondaryHealthy.Store(true) // Start healthy
	tertiaryHealthy.Store(true)  // Start healthy

	var primaryRequests atomic.Int32
	var secondaryRequests atomic.Int32
	var tertiaryRequests atomic.Int32

	// Primary server - starts unhealthy
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if primaryHealthy.Load() {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("primary healthy"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("primary unhealthy"))
			}
			return
		}
		primaryRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response from primary")
	}))
	defer primaryServer.Close()

	// Secondary server - starts healthy
	secondaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if secondaryHealthy.Load() {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("secondary healthy"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("secondary unhealthy"))
			}
			return
		}
		secondaryRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response from secondary")
	}))
	defer secondaryServer.Close()

	// Tertiary server - always healthy (fallback)
	tertiaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if tertiaryHealthy.Load() {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("tertiary healthy"))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("tertiary unhealthy"))
			}
			return
		}
		tertiaryRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response from tertiary")
	}))
	defer tertiaryServer.Close()

	// Create FailoverProxy with health checks
	fp := &FailoverProxy{
		Upstreams: []string{
			primaryServer.URL,
			secondaryServer.URL,
			tertiaryServer.URL,
		},
		HealthChecks: map[string]*HealthCheck{
			primaryServer.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			},
			secondaryServer.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			},
			tertiaryServer.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			},
		},
		FailDuration:    caddy.Duration(1 * time.Second),
		DialTimeout:     caddy.Duration(1 * time.Second),
		ResponseTimeout: caddy.Duration(2 * time.Second),
	}

	// Provision the proxy
	ctx := caddy.Context{}

	err := fp.Provision(ctx)
	if err != nil {
		t.Fatalf("Failed to provision proxy: %v", err)
	}
	defer fp.Cleanup()

	// Wait for initial health checks to complete
	time.Sleep(200 * time.Millisecond)

	// Test 1: Primary is unhealthy, should use secondary
	t.Run("primary_unhealthy_uses_secondary", func(t *testing.T) {
		// Reset request counters
		primaryRequests.Store(0)
		secondaryRequests.Store(0)
		tertiaryRequests.Store(0)

		// Make a request
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api/test", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		// Should have used secondary server
		if primaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to primary, got %d", primaryRequests.Load())
		}
		if secondaryRequests.Load() != 1 {
			t.Errorf("Expected 1 request to secondary, got %d", secondaryRequests.Load())
		}
		if tertiaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to tertiary, got %d", tertiaryRequests.Load())
		}

		body := recorder.Body.String()
		if body != "response from secondary" {
			t.Errorf("Expected response from secondary, got: %s", body)
		}

		// Verify health status
		if fp.isHealthy(primaryServer.URL) {
			t.Error("Primary should be unhealthy")
		}
		if !fp.isHealthy(secondaryServer.URL) {
			t.Error("Secondary should be healthy")
		}
	})

	// Test 2: Make primary healthy, secondary unhealthy
	t.Run("switch_health_status", func(t *testing.T) {
		// Change health status
		primaryHealthy.Store(true)
		secondaryHealthy.Store(false)

		// Wait for health checks to detect changes
		time.Sleep(150 * time.Millisecond)

		// Reset request counters
		primaryRequests.Store(0)
		secondaryRequests.Store(0)
		tertiaryRequests.Store(0)

		// Make a request
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api/test", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		// Should now use primary server
		if primaryRequests.Load() != 1 {
			t.Errorf("Expected 1 request to primary, got %d", primaryRequests.Load())
		}
		if secondaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to secondary, got %d", secondaryRequests.Load())
		}
		if tertiaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to tertiary, got %d", tertiaryRequests.Load())
		}

		body := recorder.Body.String()
		if body != "response from primary" {
			t.Errorf("Expected response from primary, got: %s", body)
		}

		// Verify health status
		if !fp.isHealthy(primaryServer.URL) {
			t.Error("Primary should be healthy")
		}
		if fp.isHealthy(secondaryServer.URL) {
			t.Error("Secondary should be unhealthy")
		}
	})

	// Test 3: All unhealthy except tertiary
	t.Run("all_unhealthy_except_tertiary", func(t *testing.T) {
		// Make primary and secondary unhealthy
		primaryHealthy.Store(false)
		secondaryHealthy.Store(false)
		tertiaryHealthy.Store(true)

		// Wait for health checks to detect changes
		time.Sleep(150 * time.Millisecond)

		// Reset request counters
		primaryRequests.Store(0)
		secondaryRequests.Store(0)
		tertiaryRequests.Store(0)

		// Make a request
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api/test", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		// Should use tertiary server
		if primaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to primary, got %d", primaryRequests.Load())
		}
		if secondaryRequests.Load() != 0 {
			t.Errorf("Expected 0 requests to secondary, got %d", secondaryRequests.Load())
		}
		if tertiaryRequests.Load() != 1 {
			t.Errorf("Expected 1 request to tertiary, got %d", tertiaryRequests.Load())
		}

		body := recorder.Body.String()
		if body != "response from tertiary" {
			t.Errorf("Expected response from tertiary, got: %s", body)
		}
	})

	// Test 4: Verify GetActiveUpstream
	t.Run("get_active_upstream", func(t *testing.T) {
		// Set health states
		primaryHealthy.Store(false)
		secondaryHealthy.Store(true)
		tertiaryHealthy.Store(true)

		// Wait for health checks
		time.Sleep(150 * time.Millisecond)

		active := fp.GetActiveUpstream()
		if active != secondaryServer.URL {
			t.Errorf("Expected active upstream to be secondary (%s), got %s", secondaryServer.URL, active)
		}

		// Make primary healthy
		primaryHealthy.Store(true)
		time.Sleep(150 * time.Millisecond)

		active = fp.GetActiveUpstream()
		if active != primaryServer.URL {
			t.Errorf("Expected active upstream to be primary (%s), got %s", primaryServer.URL, active)
		}
	})

	// Test 5: Verify GetUpstreamStatus
	t.Run("get_upstream_status", func(t *testing.T) {
		// Set specific health states
		primaryHealthy.Store(true)
		secondaryHealthy.Store(false)
		tertiaryHealthy.Store(true)

		// Wait for health checks
		time.Sleep(150 * time.Millisecond)

		statuses := fp.GetUpstreamStatus()
		if len(statuses) != 3 {
			t.Fatalf("Expected 3 upstream statuses, got %d", len(statuses))
		}

		// Check each upstream status
		for _, status := range statuses {
			switch status.Host {
			case primaryServer.URL:
				if status.Status != "UP" {
					t.Errorf("Primary should be UP, got %s", status.Status)
				}
				if !status.HealthCheck {
					t.Error("Primary should have health check enabled")
				}
			case secondaryServer.URL:
				if status.Status != "UNHEALTHY" {
					t.Errorf("Secondary should be UNHEALTHY, got %s", status.Status)
				}
				if !status.HealthCheck {
					t.Error("Secondary should have health check enabled")
				}
			case tertiaryServer.URL:
				if status.Status != "UP" {
					t.Errorf("Tertiary should be UP, got %s", status.Status)
				}
				if !status.HealthCheck {
					t.Error("Tertiary should have health check enabled")
				}
			default:
				t.Errorf("Unexpected upstream: %s", status.Host)
			}
		}
	})
}

// TestHealthCheckWithFailureCaching tests health checks with failure caching
func TestHealthCheckWithFailureCaching(t *testing.T) {
	var serverHealthy atomic.Bool
	serverHealthy.Store(true)

	var requestCount atomic.Int32

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if serverHealthy.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}

		requestCount.Add(1)
		// Return 500 for regular requests when unhealthy
		if !serverHealthy.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "server error")
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "success")
		}
	}))
	defer server.Close()

	// Create backup server that's always healthy
	backupServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "backup response")
	}))
	defer backupServer.Close()

	// Create FailoverProxy
	fp := &FailoverProxy{
		Upstreams: []string{
			server.URL,
			backupServer.URL,
		},
		HealthChecks: map[string]*HealthCheck{
			server.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(100 * time.Millisecond),
				Timeout:        caddy.Duration(50 * time.Millisecond),
				ExpectedStatus: 200,
			},
			backupServer.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(100 * time.Millisecond),
				Timeout:        caddy.Duration(50 * time.Millisecond),
				ExpectedStatus: 200,
			},
		},
		FailDuration:    caddy.Duration(500 * time.Millisecond),
		DialTimeout:     caddy.Duration(1 * time.Second),
		ResponseTimeout: caddy.Duration(2 * time.Second),
	}

	// Provision
	ctx := caddy.Context{}

	err := fp.Provision(ctx)
	if err != nil {
		t.Fatalf("Failed to provision proxy: %v", err)
	}
	defer fp.Cleanup()

	// Wait for initial health check
	time.Sleep(150 * time.Millisecond)

	// Test 1: Healthy server responds normally
	t.Run("healthy_server", func(t *testing.T) {
		requestCount.Store(0)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		if recorder.Body.String() != "success" {
			t.Errorf("Expected 'success', got: %s", recorder.Body.String())
		}

		if requestCount.Load() != 1 {
			t.Errorf("Expected 1 request, got %d", requestCount.Load())
		}
	})

	// Test 2: Server becomes unhealthy, failover happens
	t.Run("server_becomes_unhealthy", func(t *testing.T) {
		// Make server unhealthy
		serverHealthy.Store(false)

		// Wait for health check to detect
		time.Sleep(200 * time.Millisecond)

		requestCount.Store(0)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		// Should get backup response
		if recorder.Body.String() != "backup response" {
			t.Errorf("Expected 'backup response', got: %s", recorder.Body.String())
		}

		// Primary server should not have been tried
		if requestCount.Load() != 0 {
			t.Errorf("Expected 0 requests to primary, got %d", requestCount.Load())
		}

		// Verify health status
		if fp.isHealthy(server.URL) {
			t.Error("Primary server should be unhealthy")
		}
	})

	// Test 3: Server recovers
	t.Run("server_recovers", func(t *testing.T) {
		// Make server healthy again
		serverHealthy.Store(true)

		// Wait for health check to detect recovery
		time.Sleep(200 * time.Millisecond)

		// Clear any failure cache
		fp.mu.Lock()
		delete(fp.failureCache, server.URL)
		fp.mu.Unlock()

		requestCount.Store(0)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://test.local/api", nil)

		err := fp.ServeHTTP(recorder, req, nil)
		if err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		// Should get primary response again
		if recorder.Body.String() != "success" {
			t.Errorf("Expected 'success', got: %s", recorder.Body.String())
		}

		if requestCount.Load() != 1 {
			t.Errorf("Expected 1 request to primary, got %d", requestCount.Load())
		}

		// Verify health status
		if !fp.isHealthy(server.URL) {
			t.Error("Primary server should be healthy")
		}
	})
}

// TestConcurrentHealthChecks tests that health checks work correctly under concurrent load
func TestConcurrentHealthChecks(t *testing.T) {
	// Create servers with controllable health
	var server1Healthy atomic.Bool
	var server2Healthy atomic.Bool
	server1Healthy.Store(true)
	server2Healthy.Store(true)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if server1Healthy.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "server1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if server2Healthy.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "server2")
	}))
	defer server2.Close()

	// Create FailoverProxy
	fp := &FailoverProxy{
		Upstreams: []string{server1.URL, server2.URL},
		HealthChecks: map[string]*HealthCheck{
			server1.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			},
			server2.URL: {
				Path:           "/health",
				Interval:       caddy.Duration(50 * time.Millisecond),
				Timeout:        caddy.Duration(100 * time.Millisecond),
				ExpectedStatus: 200,
			},
		},
		FailDuration:    caddy.Duration(1 * time.Second),
		DialTimeout:     caddy.Duration(1 * time.Second),
		ResponseTimeout: caddy.Duration(2 * time.Second),
	}

	// Provision
	ctx := caddy.Context{}

	err := fp.Provision(ctx)
	if err != nil {
		t.Fatalf("Failed to provision proxy: %v", err)
	}
	defer fp.Cleanup()

	// Wait for initial health checks
	time.Sleep(100 * time.Millisecond)

	// Run concurrent requests while changing health status
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Goroutine to randomly change health status
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Randomly change health status
				if time.Now().UnixNano()%2 == 0 {
					server1Healthy.Store(!server1Healthy.Load())
				} else {
					server2Healthy.Store(!server2Healthy.Load())
				}
			case <-stopChan:
				return
			}
		}
	}()

	// Run concurrent requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 20; j++ {
				select {
				case <-stopChan:
					return
				default:
				}

				recorder := httptest.NewRecorder()
				req := httptest.NewRequest("GET", fmt.Sprintf("http://test.local/api/%d/%d", id, j), nil)

				err := fp.ServeHTTP(recorder, req, nil)
				if err != nil {
					// This is expected when all servers are unhealthy
					continue
				}

				// Should get response from one of the servers
				body := recorder.Body.String()
				if body != "server1" && body != "server2" && body != "All upstreams failed" {
					t.Errorf("Unexpected response: %s", body)
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Let it run for a bit
	time.Sleep(1 * time.Second)

	// Stop everything
	close(stopChan)
	wg.Wait()

	// Final health check should be consistent
	statuses := fp.GetUpstreamStatus()
	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	// Verify no data races and consistent state
	for _, status := range statuses {
		if status.Host != server1.URL && status.Host != server2.URL {
			t.Errorf("Unexpected host in status: %s", status.Host)
		}
		if !status.HealthCheck {
			t.Errorf("Health check should be enabled for %s", status.Host)
		}
		// Status should be either UP, DOWN, or UNHEALTHY
		if status.Status != "UP" && status.Status != "DOWN" && status.Status != "UNHEALTHY" {
			t.Errorf("Invalid status for %s: %s", status.Host, status.Status)
		}
	}
}
