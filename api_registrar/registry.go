package api_registrar

import (
	"sync"

	"github.com/ejlevin1/caddy-failover/api_registrar/formatters"
)

// ApiRegistry is the global registry for API specifications
type ApiRegistry struct {
	mu      sync.RWMutex
	specs   map[string]*formatters.CaddyModuleApiSpec // ID -> API specification
	configs map[string]*formatters.ApiConfig          // ID -> API configuration
}

// Global registry instance
var registry = &ApiRegistry{
	specs:   make(map[string]*formatters.CaddyModuleApiSpec),
	configs: make(map[string]*formatters.ApiConfig),
}

// RegisterApiSpec registers an API specification
// This is called by modules during init()
func RegisterApiSpec(id string, specFunc formatters.ApiSpecFunc) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if specFunc != nil {
		registry.specs[id] = specFunc()
	}
}

// ConfigureApi configures an API with path and settings
// This is called during Caddyfile parsing
func ConfigureApi(id string, config *formatters.ApiConfig) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if config != nil {
		// Store the configuration as-is
		// The Enabled flag should be set by the caller
		registry.configs[id] = config
	}
}

// GetSpecs returns all registered API specifications
func GetSpecs() map[string]*formatters.CaddyModuleApiSpec {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	specs := make(map[string]*formatters.CaddyModuleApiSpec)
	for k, v := range registry.specs {
		specs[k] = v
	}
	return specs
}

// GetConfigs returns all API configurations
func GetConfigs() map[string]*formatters.ApiConfig {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	configs := make(map[string]*formatters.ApiConfig)
	for k, v := range registry.configs {
		configs[k] = v
	}
	return configs
}

// GetSpec returns a specific API specification by ID
func GetSpec(id string) *formatters.CaddyModuleApiSpec {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.specs[id]
}

// GetConfig returns a specific API configuration by ID
func GetConfig(id string) *formatters.ApiConfig {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return registry.configs[id]
}

// Reset clears the registry (useful for testing)
func Reset() {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.specs = make(map[string]*formatters.CaddyModuleApiSpec)
	registry.configs = make(map[string]*formatters.ApiConfig)
}

// IsConfigured checks if an API is configured and enabled
func IsConfigured(id string) bool {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	config, exists := registry.configs[id]
	return exists && config != nil && config.Enabled
}
