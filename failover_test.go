package caddyfailover

import (
	"testing"
)

func TestProxyRegistry(t *testing.T) {
	// Create a new registry
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Create test proxies
	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/auth/*",
		Path:       "/auth/*",
	}

	proxy2 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5041", "https://example.com"},
		HandlePath: "/admin/*",
		Path:       "/admin/*",
	}

	// Test duplicate registration with same path (should not duplicate)
	proxy3 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/auth/*",
		Path:       "/auth/*",
	}

	// Register proxies
	registry.Register("/auth/*", proxy1)
	registry.Register("/admin/*", proxy2)
	registry.Register("/auth/*", proxy3) // This should not create a duplicate

	// Test that we have exactly 2 entries (no duplicates)
	if len(registry.proxies) != 2 {
		t.Errorf("Expected 2 entries in registry, got %d", len(registry.proxies))
	}

	// Test order preservation
	if len(registry.order) != 2 {
		t.Errorf("Expected 2 entries in order, got %d", len(registry.order))
	}

	if registry.order[0] != "/auth/*" || registry.order[1] != "/admin/*" {
		t.Errorf("Order not preserved: %v", registry.order)
	}

	// Test GetStatus
	status := registry.GetStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}

	// Verify paths don't have "auto:" prefix
	for _, s := range status {
		if len(s.Path) > 5 && s.Path[:5] == "auto:" {
			t.Errorf("Path should not have 'auto:' prefix, got: %s", s.Path)
		}
	}

	// Test unregistration
	registry.Unregister("/auth/*", proxy1)
	if len(registry.proxies) != 1 {
		t.Errorf("Expected 1 entry after unregister, got %d", len(registry.proxies))
	}
}

func TestProxyRegistryNoDuplicateUpstreams(t *testing.T) {
	registry := &ProxyRegistry{
		proxies: make(map[string]*ProxyEntry),
		order:   make([]string, 0),
	}

	// Create proxies with overlapping upstreams
	proxy1 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://example.com"},
		HandlePath: "/api/*",
		Path:       "/api/*",
	}

	proxy2 := &FailoverProxy{
		Upstreams:  []string{"http://localhost:5051", "https://backup.com"},
		HandlePath: "/api/*",
		Path:       "/api/*",
	}

	// Register both proxies with same path
	registry.Register("/api/*", proxy1)
	registry.Register("/api/*", proxy2)

	// Should have only one entry
	if len(registry.proxies) != 1 {
		t.Errorf("Expected 1 entry in registry, got %d", len(registry.proxies))
	}

	// Check that upstreams are tracked properly
	entry := registry.proxies["/api/*"]
	if entry == nil {
		t.Fatal("Entry not found for /api/*")
	}

	// Should have 3 unique upstreams total
	expectedUpstreams := map[string]bool{
		"http://localhost:5051": true,
		"https://example.com":   true,
		"https://backup.com":    true,
	}

	if len(entry.Upstreams) != len(expectedUpstreams) {
		t.Errorf("Expected %d unique upstreams, got %d", len(expectedUpstreams), len(entry.Upstreams))
	}

	for upstream := range expectedUpstreams {
		if !entry.Upstreams[upstream] {
			t.Errorf("Expected upstream %s to be tracked", upstream)
		}
	}
}

func TestProxyPathHandling(t *testing.T) {
	tests := []struct {
		name         string
		proxy        *FailoverProxy
		expectedPath string
	}{
		{
			name: "Path and HandlePath both set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "/auth/*",
				Path:       "/auth/*",
			},
			expectedPath: "/auth/*",
		},
		{
			name: "Only HandlePath set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "/api/*",
				Path:       "",
			},
			expectedPath: "/api/*",
		},
		{
			name: "Only Path set",
			proxy: &FailoverProxy{
				Upstreams:  []string{"http://localhost:5051"},
				HandlePath: "",
				Path:       "/admin/*",
			},
			expectedPath: "/admin/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &ProxyRegistry{
				proxies: make(map[string]*ProxyEntry),
				order:   make([]string, 0),
			}

			// Determine registration path (mimics Provision logic)
			registrationPath := tt.proxy.Path
			if registrationPath == "" && tt.proxy.HandlePath != "" {
				registrationPath = tt.proxy.HandlePath
			}

			if registrationPath != "" {
				registry.Register(registrationPath, tt.proxy)
			}

			status := registry.GetStatus()
			if len(status) != 1 {
				t.Fatalf("Expected 1 status entry, got %d", len(status))
			}

			if status[0].Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, status[0].Path)
			}

			// Ensure no "auto:" prefix
			if len(status[0].Path) > 5 && status[0].Path[:5] == "auto:" {
				t.Errorf("Path should not have 'auto:' prefix, got: %s", status[0].Path)
			}
		})
	}
}
