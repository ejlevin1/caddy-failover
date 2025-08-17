package caddyfailover

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
)

// TestFailoverWithHeaders tests that custom headers are properly forwarded
func TestFailoverWithHeaders(t *testing.T) {
	receivedHeaders := make(map[string]string)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture headers
		for key := range r.Header {
			receivedHeaders[key] = r.Header.Get(key)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "success")
	}))
	defer server.Close()

	fp := &FailoverProxy{
		Upstreams: []string{server.URL},
		UpstreamHeaders: map[string]map[string]string{
			server.URL: {
				"X-Custom-Header": "custom-value",
				"X-Service-Name":  "failover-test",
			},
		},
		FailDuration:    caddy.Duration(30 * time.Second),
		DialTimeout:     caddy.Duration(2 * time.Second),
		ResponseTimeout: caddy.Duration(5 * time.Second),
	}

	ctx := caddy.Context{}
	if err := fp.Provision(ctx); err != nil {
		t.Fatalf("Failed to provision: %v", err)
	}
	defer fp.Cleanup()

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("X-Original-Header", "original-value")
	w := httptest.NewRecorder()

	err := fp.ServeHTTP(w, req, nil)
	if err != nil {
		t.Errorf("ServeHTTP error: %v", err)
	}

	// Check custom headers were added
	if receivedHeaders["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected X-Custom-Header to be 'custom-value', got %q", receivedHeaders["X-Custom-Header"])
	}
	if receivedHeaders["X-Service-Name"] != "failover-test" {
		t.Errorf("Expected X-Service-Name to be 'failover-test', got %q", receivedHeaders["X-Service-Name"])
	}
	// Check original headers were preserved
	if receivedHeaders["X-Original-Header"] != "original-value" {
		t.Errorf("Expected X-Original-Header to be preserved, got %q", receivedHeaders["X-Original-Header"])
	}
}

// TestFailoverWithRequestBody tests that request bodies are properly forwarded
func TestFailoverWithRequestBody(t *testing.T) {
	expectedBody := `{"test": "data", "value": 123}`
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "received")
	}))
	defer server.Close()

	fp := CreateTestProxy(t, []string{server.URL})

	req := httptest.NewRequest("POST", "http://example.com/api", bytes.NewBufferString(expectedBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	err := fp.ServeHTTP(w, req, nil)
	if err != nil {
		t.Errorf("ServeHTTP error: %v", err)
	}

	if receivedBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, receivedBody)
	}
}

// TestFailoverPreservesPath tests that URL paths are correctly preserved
func TestFailoverPreservesPath(t *testing.T) {
	tests := []struct {
		name         string
		upstreamPath string
		requestPath  string
		expectedPath string
	}{
		{
			name:         "upstream with no path",
			upstreamPath: "",
			requestPath:  "/api/users",
			expectedPath: "/api/users",
		},
		{
			name:         "upstream with base path",
			upstreamPath: "/v1",
			requestPath:  "/api/users",
			expectedPath: "/v1/api/users",
		},
		{
			name:         "upstream with trailing slash",
			upstreamPath: "/v1/",
			requestPath:  "/api/users",
			expectedPath: "/v1/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			upstreamURL := server.URL + tt.upstreamPath
			fp := CreateTestProxy(t, []string{upstreamURL})

			req := httptest.NewRequest("GET", "http://example.com"+tt.requestPath, nil)
			w := httptest.NewRecorder()

			err := fp.ServeHTTP(w, req, nil)
			if err != nil {
				t.Errorf("ServeHTTP error: %v", err)
			}

			if capturedPath != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, capturedPath)
			}
		})
	}
}

// TestFailoverRetryLogic tests the retry behavior on failures
func TestFailoverRetryLogic(t *testing.T) {
	attemptCounts := make([]int, 3)

	servers := make([]*httptest.Server, 3)
	urls := make([]string, 3)

	for i := 0; i < 3; i++ {
		idx := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCounts[idx]++
			if idx < 2 {
				// First two servers fail
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				// Third server succeeds
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "server-%d", idx)
			}
		}))
		defer servers[i].Close()
		urls[i] = servers[i].URL
	}

	fp := CreateTestProxy(t, urls, WithFailDuration(100*time.Millisecond))

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	err := fp.ServeHTTP(w, req, nil)
	if err != nil {
		t.Errorf("ServeHTTP error: %v", err)
	}

	// Check that it tried servers in order
	if attemptCounts[0] != 1 {
		t.Errorf("Expected 1 attempt to server 0, got %d", attemptCounts[0])
	}
	if attemptCounts[1] != 1 {
		t.Errorf("Expected 1 attempt to server 1, got %d", attemptCounts[1])
	}
	if attemptCounts[2] != 1 {
		t.Errorf("Expected 1 attempt to server 2, got %d", attemptCounts[2])
	}

	// Check response is from third server
	if w.Body.String() != "server-2" {
		t.Errorf("Expected response from server-2, got %q", w.Body.String())
	}
}

// TestFailoverCacheExpiration tests that failed upstreams are retried after cache expiration
func TestFailoverCacheExpiration(t *testing.T) {
	requestCount := 0
	shouldFail := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "success")
		}
	}))
	defer server.Close()

	fp := CreateTestProxy(t, []string{server.URL}, WithFailDuration(200*time.Millisecond))

	// First request should fail
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()
	_ = fp.ServeHTTP(w, req, nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}

	// Server is now healthy but still in failure cache
	shouldFail = false
	requestCount = 0

	// Immediate retry should not attempt (still in cache)
	w = httptest.NewRecorder()
	_ = fp.ServeHTTP(w, req, nil)

	if requestCount != 0 {
		t.Error("Should not retry immediately while in failure cache")
	}

	// Wait for cache to expire
	time.Sleep(250 * time.Millisecond)

	// Now it should retry and succeed
	w = httptest.NewRecorder()
	err := fp.ServeHTTP(w, req, nil)
	if err != nil {
		t.Errorf("ServeHTTP error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after cache expiration, got %d", w.Code)
	}
	if requestCount != 1 {
		t.Errorf("Expected 1 request after cache expiration, got %d", requestCount)
	}
}

// TestXForwardedHeaders tests that X-Forwarded headers are properly set
func TestXForwardedHeaders(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fp := CreateTestProxy(t, []string{server.URL})

	req := httptest.NewRequest("GET", "https://example.com/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Host = "example.com"
	req.TLS = &tls.ConnectionState{} // Simulate HTTPS

	w := httptest.NewRecorder()
	err := fp.ServeHTTP(w, req, nil)
	if err != nil {
		t.Errorf("ServeHTTP error: %v", err)
	}

	// Check X-Forwarded headers
	if capturedHeaders.Get("X-Forwarded-For") != "192.168.1.100" {
		t.Errorf("Expected X-Forwarded-For to be '192.168.1.100', got %q", capturedHeaders.Get("X-Forwarded-For"))
	}
	if capturedHeaders.Get("X-Forwarded-Proto") != "https" {
		t.Errorf("Expected X-Forwarded-Proto to be 'https', got %q", capturedHeaders.Get("X-Forwarded-Proto"))
	}
	if capturedHeaders.Get("X-Forwarded-Host") != "example.com" {
		t.Errorf("Expected X-Forwarded-Host to be 'example.com', got %q", capturedHeaders.Get("X-Forwarded-Host"))
	}
}

// TestConcurrentRequests tests that the proxy handles concurrent requests correctly
func TestConcurrentRequests(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		time.Sleep(10 * time.Millisecond) // Simulate some processing
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "response-%d", count)
	}))
	defer server.Close()

	fp := CreateTestProxy(t, []string{server.URL})

	// Send multiple concurrent requests
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/test%d", id), nil)
			w := httptest.NewRecorder()

			err := fp.ServeHTTP(w, req, nil)
			if err != nil {
				t.Errorf("Request %d: ServeHTTP error: %v", id, err)
			}
			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", id, w.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	if requestCount.Load() != int32(concurrency) {
		t.Errorf("Expected %d requests, got %d", concurrency, requestCount.Load())
	}
}

// TestStatusEndpointConcurrency tests concurrent access to status endpoint
func TestStatusEndpointConcurrency(t *testing.T) {
	// Setup registry with multiple paths
	registry := CreateTestRegistry("/api/*", "/auth/*", "/admin/*", "/gateway/*")

	// Replace global registry temporarily
	oldRegistry := proxyRegistry
	proxyRegistry = registry
	defer func() { proxyRegistry = oldRegistry }()

	handler := FailoverStatusHandler{}

	// Send concurrent requests to status endpoint
	concurrency := 20
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/admin/failover/status", nil)
			w := httptest.NewRecorder()

			err := handler.ServeHTTP(w, req, nil)
			if err != nil {
				errors <- fmt.Errorf("ServeHTTP error: %v", err)
				done <- false
				return
			}

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("Expected status 200, got %d", w.Code)
				done <- false
				return
			}

			var status []PathStatus
			if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
				errors <- fmt.Errorf("Failed to parse JSON: %v", err)
				done <- false
				return
			}

			if len(status) != 4 {
				errors <- fmt.Errorf("Expected 4 paths, got %d", len(status))
				done <- false
				return
			}

			done <- true
		}()
	}

	// Wait for all requests and check for errors
	successCount := 0
	for i := 0; i < concurrency; i++ {
		if <-done {
			successCount++
		}
	}

	close(errors)
	for err := range errors {
		t.Error(err)
	}

	if successCount != concurrency {
		t.Errorf("Expected %d successful requests, got %d", concurrency, successCount)
	}
}
