# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Caddy server plugin that provides intelligent failover between multiple upstream servers with mixed HTTP/HTTPS support, health checking, and OpenAPI documentation capabilities. The plugin is written in Go and integrates with Caddy v2.

## Build and Development Commands

### Building
```bash
# Build the plugin locally (requires Go 1.22+)
go build -v ./...

# Build Caddy with this plugin using xcaddy
xcaddy build --with github.com/ejlevin1/caddy-failover=.

# Build Docker image
./scripts/docker-build-local.sh

# Build Docker image with specific tag
./scripts/build-docker-with-tag.sh "my-tag"
./scripts/build-docker-with-tag.sh "my-tag" loaded  # with additional plugins
```

### Testing
```bash
# Run all tests
./scripts/test.sh all

# Run unit tests only
./scripts/test.sh unit

# Run unit tests with coverage and race detection
./scripts/test.sh unit -c -r

# Run integration tests
./scripts/test.sh integration -v

# Run benchmarks
./scripts/test.sh benchmark

# Generate coverage report
./scripts/test.sh coverage

# Run specific test
go test -v -run TestFailoverProxy ./...

# Run tests with race detection
go test -race ./...
```

### Formatting and Linting
```bash
# Format code
go fmt ./...
gofmt -s -w .

# Run linter (requires golangci-lint)
golangci-lint run

# Tidy modules
go mod tidy
```

### Docker Operations
```bash
# Run integration tests with Docker
./test/test.sh

# Test failover status endpoint
./test/test-failover-status.sh

# Test OpenAPI endpoints
./test/test-openapi-docker.sh
```

## Architecture and Code Structure

### Module Registration
The plugin registers with Caddy through `module.go`:
- `FailoverProxy`: Main handler for failover functionality
- `FailoverStatusHandler`: Status endpoint for monitoring
- Directives: `failover_proxy` and `failover_status` for Caddyfile
- API Registrar: OpenAPI documentation generation

### Core Components

1. **Failover Proxy (`failover/handler.go`)**
   - Manages multiple upstream servers with automatic failover
   - Tracks failed upstreams and remembers failures for `fail_duration`
   - Supports per-upstream headers and configuration
   - Implements health checking per upstream

2. **Health Checking System**
   - Configurable health check intervals and timeouts
   - Per-upstream health check paths and expected status codes
   - Background goroutines manage health checks independently

3. **API Documentation (`api_registrar/`)**
   - Generates OpenAPI 3.0/3.1 specifications dynamically
   - Provides Swagger UI and Redoc interfaces
   - Registers multiple API specs (Caddy Admin API, Failover API)

4. **Status Monitoring**
   - Global `ProxyRegistry` tracks all failover proxy instances
   - Real-time status reporting through `/admin/failover/status`
   - Prevents duplicate proxy registrations for the same path

### Directive Ordering
Caddy requires explicit ordering for custom directives. The plugin uses:
```
order failover_proxy before reverse_proxy
order failover_status before respond
order caddy_api_registrar before respond
```

### Testing Approach
- Unit tests: Core functionality in `*_test.go` files
- Integration tests: Full Caddy server tests using `caddytest`
- Benchmark tests: Performance testing in `*_benchmark_test.go`
- Docker tests: End-to-end testing with containerized setup
- Use build tag `//go:build !short` for integration tests

### Key Patterns

1. **Parser Functions**: All Caddyfile parsing uses `httpcaddyfile.Helper`
2. **Error Handling**: Always return structured errors with context
3. **Logging**: Use `zap.Logger` from Caddy context
4. **Context Propagation**: Pass `context.Context` through request handling
5. **Mutex Protection**: Use `sync.RWMutex` for concurrent access to shared state

### GitHub Workflows
- **test.yml**: Runs on PRs and main branch, tests Go 1.22 and 1.23
- **release.yml**: Semantic versioning with automatic Docker image builds
- **commitlint.yml**: Enforces conventional commit messages

### Environment Variables
The plugin supports Caddy's `{env.VARIABLE}` syntax in Caddyfile for dynamic configuration.

### Release Process
Uses semantic-release for automated versioning:
- Commits follow conventional format (feat:, fix:, etc.)
- Releases trigger Docker image builds for multiple tags
- Images published to ghcr.io/ejlevin1/caddy-failover
