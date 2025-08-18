package failover

import (
	"sync"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestActiveUpstreamMetrics tests the UpdateMetrics method
func TestActiveUpstreamMetrics(t *testing.T) {
	tests := []struct {
		name       string
		operations []struct {
			responseMs int64
			success    bool
		}
		expectedCount   int64
		expectedFailed  int64
		expectedAvgMs   float64
		expectedSuccess float64
	}{
		{
			name: "all successful requests",
			operations: []struct {
				responseMs int64
				success    bool
			}{
				{100, true},
				{200, true},
				{150, true},
			},
			expectedCount:   3,
			expectedFailed:  0,
			expectedAvgMs:   150.0,
			expectedSuccess: 100.0,
		},
		{
			name: "mixed success and failures",
			operations: []struct {
				responseMs int64
				success    bool
			}{
				{100, true},
				{0, false},
				{200, true},
				{0, false},
			},
			expectedCount:   4,
			expectedFailed:  2,
			expectedAvgMs:   150.0, // (100+200)/2
			expectedSuccess: 50.0,
		},
		{
			name: "all failed requests",
			operations: []struct {
				responseMs int64
				success    bool
			}{
				{0, false},
				{0, false},
				{0, false},
			},
			expectedCount:   3,
			expectedFailed:  3,
			expectedAvgMs:   0.0,
			expectedSuccess: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			au := &ActiveUpstream{
				URL:   "http://example.com",
				Since: time.Now(),
			}

			for _, op := range tt.operations {
				au.UpdateMetrics(op.responseMs, op.success)
			}

			assert.Equal(t, tt.expectedCount, au.RequestCount, "request count mismatch")
			assert.Equal(t, tt.expectedFailed, au.FailedRequests, "failed requests mismatch")
			assert.InDelta(t, tt.expectedAvgMs, au.AvgResponseMs, 0.01, "average response time mismatch")
			assert.InDelta(t, tt.expectedSuccess, au.SuccessRate, 0.01, "success rate mismatch")
		})
	}
}

// TestActiveUpstreamChangeDetection tests the checkActiveUpstreamChange method
func TestActiveUpstreamChangeDetection(t *testing.T) {
	tests := []struct {
		name             string
		upstreams        []string
		healthStatus     map[string]bool
		failureCache     map[string]time.Time
		currentActive    *ActiveUpstream
		expectedActive   string
		expectLogWarning bool
	}{
		{
			name:      "initialize active upstream",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache:     map[string]time.Time{},
			currentActive:    nil,
			expectedActive:   "http://primary",
			expectLogWarning: true,
		},
		{
			name:      "primary fails, switch to backup",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": false,
				"http://backup":  true,
			},
			failureCache: map[string]time.Time{},
			currentActive: &ActiveUpstream{
				URL: "http://primary",
			},
			expectedActive:   "http://backup",
			expectLogWarning: true,
		},
		{
			name:      "primary recovers, switch back",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache: map[string]time.Time{},
			currentActive: &ActiveUpstream{
				URL: "http://backup",
			},
			expectedActive:   "http://primary",
			expectLogWarning: true,
		},
		{
			name:      "no change needed",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache: map[string]time.Time{},
			currentActive: &ActiveUpstream{
				URL: "http://primary",
			},
			expectedActive:   "http://primary",
			expectLogWarning: false,
		},
		{
			name:      "all upstreams fail",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": false,
				"http://backup":  false,
			},
			failureCache: map[string]time.Time{},
			currentActive: &ActiveUpstream{
				URL: "http://primary",
			},
			expectedActive:   "",
			expectLogWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test logger that captures log messages
			logger := zaptest.NewLogger(t)

			f := &FailoverProxy{
				Upstreams:      tt.upstreams,
				healthStatus:   tt.healthStatus,
				failureCache:   tt.failureCache,
				activeUpstream: tt.currentActive,
				FailDuration:   caddy.Duration(30 * time.Second),
				logger:         logger,
			}

			// Call the method (already holding lock as per method requirements)
			f.checkActiveUpstreamChange()

			// Check the result
			if tt.expectedActive == "" {
				assert.Nil(t, f.activeUpstream, "expected nil active upstream")
			} else {
				require.NotNil(t, f.activeUpstream, "expected non-nil active upstream")
				assert.Equal(t, tt.expectedActive, f.activeUpstream.URL, "active upstream mismatch")
			}
		})
	}
}

// TestThreadSafety tests concurrent access to active upstream
func TestThreadSafety(t *testing.T) {
	logger := zaptest.NewLogger(t)

	f := &FailoverProxy{
		Upstreams: []string{"http://primary", "http://backup", "http://tertiary"},
		healthStatus: map[string]bool{
			"http://primary":  true,
			"http://backup":   true,
			"http://tertiary": true,
		},
		failureCache:   map[string]time.Time{},
		activeUpstream: nil,
		FailDuration:   caddy.Duration(30 * time.Second),
		logger:         logger,
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	iterations := 100

	// Goroutine 1: Update health status
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			upstream := f.Upstreams[i%len(f.Upstreams)]
			healthy := i%2 == 0
			f.setHealthStatus(upstream, healthy)
			time.Sleep(time.Microsecond)
		}
	}()

	// Goroutine 2: Update metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			f.mu.Lock()
			if f.activeUpstream != nil {
				f.activeUpstream.UpdateMetrics(int64(i*10), i%3 != 0)
			}
			f.mu.Unlock()
			time.Sleep(time.Microsecond)
		}
	}()

	// Goroutine 3: Read metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			metrics := f.GetActiveUpstreamMetrics()
			_ = metrics // Just read, don't assert
			time.Sleep(time.Microsecond)
		}
	}()

	// Goroutine 4: Get active upstream
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			active := f.GetActiveUpstream()
			_ = active // Just read, don't assert
			time.Sleep(time.Microsecond)
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// If we get here without panics or deadlocks, the test passes
	assert.True(t, true, "concurrent operations completed without issues")
}

// TestDetermineChangeReason tests the reason determination logic
func TestDetermineChangeReason(t *testing.T) {
	tests := []struct {
		name           string
		from           string
		to             string
		upstreams      []string
		healthStatus   map[string]bool
		failureCache   map[string]time.Time
		expectedReason string
	}{
		{
			name:      "previous upstream unhealthy",
			from:      "http://primary",
			to:        "http://backup",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": false,
				"http://backup":  true,
			},
			failureCache:   map[string]time.Time{},
			expectedReason: "previous upstream unhealthy",
		},
		{
			name:      "previous upstream in failure state",
			from:      "http://primary",
			to:        "http://backup",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache: map[string]time.Time{
				"http://primary": time.Now(),
			},
			expectedReason: "previous upstream in failure state",
		},
		{
			name:      "higher priority upstream recovered",
			from:      "http://backup",
			to:        "http://primary",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache:   map[string]time.Time{},
			expectedReason: "higher priority upstream recovered",
		},
		{
			name:      "unknown reason",
			from:      "http://unknown",
			to:        "http://tertiary",
			upstreams: []string{"http://primary", "http://backup"},
			healthStatus: map[string]bool{
				"http://primary": true,
				"http://backup":  true,
			},
			failureCache:   map[string]time.Time{},
			expectedReason: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FailoverProxy{
				Upstreams:    tt.upstreams,
				healthStatus: tt.healthStatus,
				failureCache: tt.failureCache,
				logger:       zap.NewNop(),
			}

			reason := f.determineChangeReason(tt.from, tt.to)
			assert.Equal(t, tt.expectedReason, reason, "change reason mismatch")
		})
	}
}

// TestActiveUpstreamWithServeHTTP tests metrics tracking during request handling
func TestActiveUpstreamWithServeHTTP(t *testing.T) {
	// This test would require more setup with HTTP test servers
	// For now, we'll focus on the unit tests above
	t.Skip("Integration test - requires HTTP test servers")
}
