package api_registrar

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegisterApiSpec(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Test registering a spec
	spec := &CaddyModuleApiSpec{
		ID:      "test_api",
		Title:   "Test API",
		Version: "1.0",
	}

	RegisterApiSpec("test_api", func() *CaddyModuleApiSpec {
		return spec
	})

	// Verify registration
	registeredSpec := GetSpec("test_api")
	if registeredSpec == nil {
		t.Fatal("Expected spec to be registered")
	}
	if registeredSpec.ID != "test_api" {
		t.Errorf("Expected ID 'test_api', got '%s'", registeredSpec.ID)
	}
	if registeredSpec.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", registeredSpec.Title)
	}
}

func TestConfigureApi(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Test configuring an API
	config := &ApiConfig{
		Path:    "/api/v1",
		Enabled: true,
		Title:   "Custom Title",
	}

	ConfigureApi("test_api", config)

	// Verify configuration
	registeredConfig := GetConfig("test_api")
	if registeredConfig == nil {
		t.Fatal("Expected config to be registered")
	}
	if registeredConfig.Path != "/api/v1" {
		t.Errorf("Expected path '/api/v1', got '%s'", registeredConfig.Path)
	}
	if !registeredConfig.Enabled {
		t.Error("Expected config to be enabled")
	}
}

func TestIsConfigured(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Test with unconfigured API
	if IsConfigured("unknown_api") {
		t.Error("Expected unknown API to not be configured")
	}

	// Configure an API
	ConfigureApi("test_api", &ApiConfig{
		Path:    "/test",
		Enabled: true,
	})

	// Test with configured API
	if !IsConfigured("test_api") {
		t.Error("Expected test_api to be configured")
	}

	// Test with disabled API
	ConfigureApi("disabled_api", &ApiConfig{
		Path:    "/disabled",
		Enabled: false,
	})

	if IsConfigured("disabled_api") {
		t.Error("Expected disabled_api to not be configured (disabled)")
	}
}

func TestGetSpecs(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Register multiple specs
	RegisterApiSpec("api1", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{ID: "api1", Title: "API 1"}
	})
	RegisterApiSpec("api2", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{ID: "api2", Title: "API 2"}
	})

	// Get all specs
	specs := GetSpecs()
	if len(specs) != 2 {
		t.Errorf("Expected 2 specs, got %d", len(specs))
	}

	// Verify both specs are present
	if specs["api1"] == nil || specs["api1"].Title != "API 1" {
		t.Error("api1 not found or incorrect")
	}
	if specs["api2"] == nil || specs["api2"].Title != "API 2" {
		t.Error("api2 not found or incorrect")
	}
}

func TestGetConfigs(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Configure multiple APIs
	ConfigureApi("api1", &ApiConfig{Path: "/path1", Enabled: true})
	ConfigureApi("api2", &ApiConfig{Path: "/path2", Enabled: true})

	// Get all configs
	configs := GetConfigs()
	if len(configs) != 2 {
		t.Errorf("Expected 2 configs, got %d", len(configs))
	}

	// Verify both configs are present
	if configs["api1"] == nil || configs["api1"].Path != "/path1" {
		t.Error("api1 config not found or incorrect")
	}
	if configs["api2"] == nil || configs["api2"].Path != "/path2" {
		t.Error("api2 config not found or incorrect")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Reset registry before test
	Reset()
	defer Reset()

	// Test concurrent registration and configuration
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent spec registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			apiID := fmt.Sprintf("api_%d", id)
			RegisterApiSpec(apiID, func() *CaddyModuleApiSpec {
				return &CaddyModuleApiSpec{
					ID:    apiID,
					Title: fmt.Sprintf("API %d", id),
				}
			})
		}(i)
	}

	// Concurrent configuration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			apiID := fmt.Sprintf("api_%d", id)
			ConfigureApi(apiID, &ApiConfig{
				Path:    fmt.Sprintf("/api/%d", id),
				Enabled: true,
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = GetSpecs()
			_ = GetConfigs()
			_ = GetSpec(fmt.Sprintf("api_%d", id))
			_ = GetConfig(fmt.Sprintf("api_%d", id))
			_ = IsConfigured(fmt.Sprintf("api_%d", id))
		}(i)
	}

	wg.Wait()

	// Verify all registrations succeeded
	specs := GetSpecs()
	if len(specs) != numGoroutines {
		t.Errorf("Expected %d specs after concurrent registration, got %d", numGoroutines, len(specs))
	}

	configs := GetConfigs()
	if len(configs) != numGoroutines {
		t.Errorf("Expected %d configs after concurrent configuration, got %d", numGoroutines, len(configs))
	}
}

func TestReset(t *testing.T) {
	// Register and configure some APIs
	RegisterApiSpec("test1", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{ID: "test1"}
	})
	RegisterApiSpec("test2", func() *CaddyModuleApiSpec {
		return &CaddyModuleApiSpec{ID: "test2"}
	})
	ConfigureApi("test1", &ApiConfig{Path: "/test1"})
	ConfigureApi("test2", &ApiConfig{Path: "/test2"})

	// Verify they exist
	if len(GetSpecs()) != 2 {
		t.Error("Expected 2 specs before reset")
	}
	if len(GetConfigs()) != 2 {
		t.Error("Expected 2 configs before reset")
	}

	// Reset
	Reset()

	// Verify registry is empty
	if len(GetSpecs()) != 0 {
		t.Error("Expected 0 specs after reset")
	}
	if len(GetConfigs()) != 0 {
		t.Error("Expected 0 configs after reset")
	}
}
