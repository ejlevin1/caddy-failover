# Caddy Failover Plugin

[![Test Plugin](https://github.com/ejlevin1/caddy-failover/actions/workflows/test.yml/badge.svg)](https://github.com/ejlevin1/caddy-failover/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/ejlevin1/caddy-failover/branch/main/graph/badge.svg)](https://codecov.io/gh/ejlevin1/caddy-failover)
[![Go Report Card](https://goreportcard.com/badge/github.com/ejlevin1/caddy-failover)](https://goreportcard.com/report/github.com/ejlevin1/caddy-failover)
[![Go Reference](https://pkg.go.dev/badge/github.com/ejlevin1/caddy-failover.svg)](https://pkg.go.dev/github.com/ejlevin1/caddy-failover)

A comprehensive Caddy plugin that provides intelligent failover between multiple upstream servers with mixed HTTP/HTTPS support, health checking, and OpenAPI documentation capabilities.

## Features

- **Intelligent Failover**: Automatic failover between multiple upstreams with health checking
- **Mixed-scheme Support**: HTTP and HTTPS upstreams in the same directive
- **API Documentation**: Built-in Swagger UI, Redoc, and OpenAPI spec generation
- **Health Monitoring**: Per-upstream health checks with configurable intervals
- **Status Dashboard**: Real-time status endpoint for all failover proxies
- **Per-upstream Configuration**: Different headers and settings for each upstream
- **Environment Variables**: Full support for environment variable expansion
- **Path Preservation**: Maintains upstream base paths in routing

## Quick Start

### Using Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/ejlevin1/caddy-failover:latest

# Run with your Caddyfile
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -p 2019:2019 \
  -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
  ghcr.io/ejlevin1/caddy-failover:latest
```

### Using xcaddy

```bash
# Build Caddy with the plugin
xcaddy build --with github.com/ejlevin1/caddy-failover

# Run with your Caddyfile
./caddy run --config Caddyfile
```

## Configuration Guide

### Global Configuration

The plugin requires global configuration to set directive ordering and API registrar settings:

```caddyfile
{
    # Required: Set directive ordering
    order failover_proxy before reverse_proxy
    order failover_status before respond
    order caddy_api_registrar before respond

    # Optional: Admin API endpoint
    admin :2019

    # Optional: API Registrar configuration for OpenAPI documentation
    caddy_api_registrar {
        # Register Caddy's admin API documentation
        caddy_api {
            path /caddy-admin
            title "Caddy Admin API"
            version "2.0"
        }

        # Register failover plugin API documentation
        failover_api {
            path /caddy/failover
            title "Failover Plugin API"
            version "1.0"
        }
    }

    # Optional: Enable debug logging
    # debug
}
```

### Basic Failover Configuration

```caddyfile
:443 {
    # Simple failover between two upstreams
    handle /api/* {
        failover_proxy http://primary.local:3000 http://backup.local:3000 {
            fail_duration 5s
            dial_timeout 2s
            response_timeout 10s
        }
    }
}
```

### Advanced Failover with Health Checks

```caddyfile
:443 {
    handle /api/* {
        failover_proxy http://primary.local:3000 https://backup.cloud {
            # Basic settings
            fail_duration 10s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify  # For self-signed certs in dev

            # Status tracking (required for status endpoint)
            status_path /api/*

            # Health check for primary upstream
            health_check http://primary.local:3000 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }

            # Health check for backup upstream
            health_check https://backup.cloud {
                path /status
                interval 60s
                timeout 10s
                expected_status 204
            }

            # Per-upstream headers
            header_up http://primary.local:3000 X-Environment development
            header_up http://primary.local:3000 X-Source local
            header_up https://backup.cloud X-Environment production
            header_up https://backup.cloud Host api.backup.cloud
        }
    }
}
```

## API Documentation Endpoints

The plugin includes built-in API documentation generation with Swagger UI and Redoc support.

### Setting Up API Documentation

```caddyfile
{
    # Global configuration (required)
    order failover_proxy before reverse_proxy
    order failover_status before respond
    order caddy_api_registrar before respond

    # Configure which APIs to document
    caddy_api_registrar {
        failover_api {
            path /caddy/failover
            title "Failover Plugin API"
            version "1.0.0"
        }
        caddy_api {
            path /caddy-admin
            title "Caddy Admin API"
            version "2.0"
        }
    }
}

:443 {
    # Swagger UI endpoint
    handle /api/docs {
        caddy_api_registrar swagger-ui
    }

    # Alternative with trailing slash support
    handle /api/docs* {
        caddy_api_registrar swagger-ui
    }

    # Redoc UI endpoint
    handle /api/docs/redoc* {
        caddy_api_registrar redoc
    }

    # Raw OpenAPI JSON endpoints
    handle /api/docs/openapi.json {
        caddy_api_registrar openapi-v3.0
    }

    handle /api/docs/openapi-3.1.json {
        caddy_api_registrar openapi-v3.1
    }

    # Your failover configurations...
    handle /api/* {
        failover_proxy http://localhost:3000 http://backup:3000 {
            status_path /api/*
        }
    }
}
```

### Accessing API Documentation

Once configured, you can access:
- **Swagger UI**: `https://your-domain/api/docs/`
- **Redoc UI**: `https://your-domain/api/docs/redoc/`
- **OpenAPI 3.0 JSON**: `https://your-domain/api/docs/openapi.json`
- **OpenAPI 3.1 JSON**: `https://your-domain/api/docs/openapi-3.1.json`

## Status Monitoring

The plugin provides a real-time status endpoint to monitor all configured failover proxies.

### Configuring Status Endpoint

```caddyfile
{
    # Required global ordering
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:443 {
    # Status endpoint
    handle /admin/failover/status {
        failover_status
    }

    # Failover proxies with status tracking
    handle /api/* {
        failover_proxy http://api1.local http://api2.local {
            # IMPORTANT: status_path enables tracking for this proxy
            status_path /api/*
            fail_duration 5s

            health_check http://api1.local {
                path /health
                interval 30s
            }
        }
    }

    handle /auth/* {
        failover_proxy http://auth1.local http://auth2.local {
            status_path /auth/*
            fail_duration 5s
        }
    }
}
```

### Status Response Format

```json
[
    {
        "path": "/api/*",
        "failover_proxies": [
            {
                "host": "http://api1.local",
                "status": "UP",
                "health_check_enabled": true,
                "last_check": "2024-01-15T10:30:45Z",
                "response_time_ms": 125
            },
            {
                "host": "http://api2.local",
                "status": "DOWN",
                "health_check_enabled": false,
                "last_failure": "2024-01-15T10:29:30Z"
            }
        ]
    }
]
```

## Handle vs Route Directives

Caddy offers two ways to configure request handling: `handle` and `route`. Understanding the difference is crucial for proper failover configuration.

### Using `handle` (Requires Global Ordering)

```caddyfile
{
    # Global ordering required when using handle
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:443 {
    # handle blocks are mutually exclusive
    # Only ONE handle block matches per request
    handle /api/* {
        failover_proxy http://api1.local http://api2.local
    }

    handle /auth/* {
        failover_proxy http://auth1.local http://auth2.local
    }

    handle {
        # Default handler for everything else
        respond "Not found" 404
    }
}
```

### Using `route` (No Global Ordering Needed)

```caddyfile
# No global ordering needed with route

:443 {
    # route blocks preserve directive order explicitly
    route /api/* {
        failover_proxy http://api1.local http://api2.local
    }

    route /auth/* {
        failover_proxy http://auth1.local http://auth2.local
    }

    # Can combine route and handle
    handle /status {
        route {
            failover_status
        }
    }
}
```

### Key Differences

| Aspect | `handle` | `route` |
|--------|----------|---------|
| **Ordering** | Requires global `order` directive | Order is explicit in config |
| **Matching** | Mutually exclusive (only one matches) | All matching routes execute |
| **Use Case** | Simple path-based routing | Complex routing with multiple directives |
| **Performance** | Slightly faster | Slightly slower (but negligible) |

## Complete Example Configuration

Here's a comprehensive example showing all features:

```caddyfile
{
    # Global configuration
    order failover_proxy before reverse_proxy
    order failover_status before respond
    order caddy_api_registrar before respond

    # Admin API
    admin :2019

    # API documentation configuration
    caddy_api_registrar {
        failover_api {
            path /caddy/failover
            title "Failover Plugin API"
            version "1.0.0"
        }
        caddy_api {
            path /caddy-admin
            title "Caddy Admin API"
            version "2.0"
        }
    }

    # Optional: Enable debug logging
    # debug
}

# HTTPS server
https://localhost:443 {
    # API Documentation endpoints
    handle /api/docs* {
        caddy_api_registrar swagger-ui
    }

    handle /api/docs/redoc* {
        caddy_api_registrar redoc
    }

    handle /api/docs/openapi.json {
        caddy_api_registrar openapi-v3.0
    }

    # Status monitoring endpoint
    handle /admin/failover/status {
        failover_status
    }

    # Admin API proxy
    handle /caddy-admin/* {
        uri strip_prefix /caddy-admin
        reverse_proxy localhost:2019
    }

    # Application API with failover
    handle /api/* {
        failover_proxy http://localhost:3000 http://docker:3000 https://api.production.com {
            # Timeouts
            fail_duration 5s
            dial_timeout 2s
            response_timeout 10s

            # Enable status tracking
            status_path /api/*

            # Skip cert verification for dev
            insecure_skip_verify

            # Health checks
            health_check http://localhost:3000 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }

            health_check http://docker:3000 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }

            health_check https://api.production.com {
                path /health
                interval 60s
                timeout 10s
                expected_status 200
            }

            # Per-upstream headers
            header_up http://localhost:3000 X-Environment local
            header_up http://docker:3000 X-Environment docker
            header_up https://api.production.com X-Environment production
            header_up https://api.production.com Host api.production.com
        }
    }

    # Authentication service with failover
    handle /auth/* {
        failover_proxy http://auth1.local:8080 http://auth2.local:8080 {
            status_path /auth/*
            fail_duration 3s

            header_up http://auth1.local:8080 X-Service auth-primary
            header_up http://auth2.local:8080 X-Service auth-backup
        }
    }

    # Default handler
    handle {
        respond "Not found" 404
    }
}

# HTTP server with redirect
http://localhost:80 {
    # Health check endpoint (doesn't redirect)
    handle /health {
        respond "OK" 200
    }

    # Redirect everything else to HTTPS
    handle {
        redir https://localhost:443{uri} permanent
    }
}
```

## Configuration Options Reference

### Failover Proxy Options

| Option | Description | Default |
|--------|-------------|---------|
| `fail_duration` | How long to remember failed upstreams | `30s` |
| `dial_timeout` | Connection timeout | `2s` |
| `response_timeout` | Response timeout | `5s` |
| `insecure_skip_verify` | Skip TLS certificate verification | `false` |
| `status_path` | Path identifier for status reporting | (auto-generated) |
| `header_up <upstream> <name> <value>` | Set upstream-specific headers | - |
| `health_check <upstream> { ... }` | Configure health checks | - |

### Health Check Options

Health checks are configured per upstream URL within the `failover_proxy` block:

```caddyfile
health_check <upstream_url> {
    path <endpoint_path>
    interval <duration>
    timeout <duration>
    expected_status <http_code>
}
```

| Option | Description | Default |
|--------|-------------|---------|
| `path` | Health check endpoint path | `/health` |
| `interval` | Check interval | `30s` |
| `timeout` | Check timeout | `5s` |
| `expected_status` | Expected HTTP status code | `200` |

**Important:** Each `health_check` directive must specify the upstream URL it applies to.

### API Registrar Formats

| Format | Description |
|--------|-------------|
| `swagger-ui` | Interactive Swagger UI interface |
| `redoc` | Clean Redoc documentation interface |
| `openapi-v3.0` | OpenAPI 3.0 JSON specification |
| `openapi-v3.1` | OpenAPI 3.1 JSON specification |

## Docker Images

### Available Images

```bash
# Standard image (failover plugin only)
docker pull ghcr.io/ejlevin1/caddy-failover:latest
docker pull ghcr.io/ejlevin1/caddy-failover:1.9.0
docker pull ghcr.io/ejlevin1/caddy-failover:1

# Loaded image (with additional plugins)
docker pull ghcr.io/ejlevin1/caddy-failover:latest-loaded
docker pull ghcr.io/ejlevin1/caddy-failover:1.9.0-loaded
docker pull ghcr.io/ejlevin1/caddy-failover:1-loaded
```

The `-loaded` variants include:
- [caddy-admin-ui](https://github.com/gsmlg-dev/caddy-admin-ui) - Web UI for Caddy admin
- [caddy-docker-proxy](https://github.com/lucaslorentz/caddy-docker-proxy) - Docker container auto-discovery

### Docker Compose Example

```yaml
version: '3.8'

services:
  caddy:
    image: ghcr.io/ejlevin1/caddy-failover:latest
    container_name: caddy-failover
    ports:
      - "80:80"
      - "443:443"
      - "2019:2019"  # Admin API
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    environment:
      - PRIMARY_HOST=host.docker.internal
      - BACKUP_HOST=api.example.com
    restart: unless-stopped

volumes:
  caddy_data:
  caddy_config:
```

### Building Local Docker Images

```bash
# Build with custom tag
./scripts/build-docker-with-tag.sh "my-tag"

# Build loaded variant
./scripts/build-docker-with-tag.sh "my-tag" loaded

# Quick development build
./scripts/docker-build-local.sh
```

## Environment Variables

The plugin supports environment variable expansion using Caddy's `{env.VARIABLE}` syntax:

```caddyfile
:443 {
    handle /api/* {
        failover_proxy http://{env.PRIMARY_HOST}:3000 https://{env.BACKUP_HOST} {
            fail_duration {env.FAIL_DURATION}

            header_up http://{env.PRIMARY_HOST}:3000 X-API-Key {env.API_KEY}
            header_up https://{env.BACKUP_HOST} X-API-Key {env.PROD_API_KEY}
        }
    }
}
```

Run with environment variables:
```bash
export PRIMARY_HOST=localhost
export BACKUP_HOST=api.example.com
export API_KEY=dev-key-123
export PROD_API_KEY=prod-key-456
export FAIL_DURATION=5s

caddy run --config Caddyfile
```

## Common Use Cases

### Development Environment with IDE Priority

Route to local IDE first, fall back to Docker, then production:

```caddyfile
:443 {
    handle /api/* {
        failover_proxy http://localhost:5000 http://docker:5000 https://api.prod.com {
            status_path /api/*
            fail_duration 2s

            header_up http://localhost:5000 X-Environment local-ide
            header_up http://docker:5000 X-Environment docker
            header_up https://api.prod.com X-Environment production
        }
    }
}
```

### Microservices with Different Health Endpoints

```caddyfile
:443 {
    # User service
    handle /user/* {
        failover_proxy http://user1:8080 http://user2:8080 {
            status_path /user/*

            health_check http://user1:8080 {
                path /actuator/health
                interval 30s
            }

            health_check http://user2:8080 {
                path /healthz
                interval 30s
            }
        }
    }

    # Order service
    handle /order/* {
        failover_proxy http://order1:8081 http://order2:8081 {
            status_path /order/*

            health_check http://order1:8081 {
                path /health/live
                interval 30s
            }
        }
    }
}
```

### Geographic Failover

```caddyfile
:443 {
    handle /api/* {
        failover_proxy https://us-east.api.com https://us-west.api.com https://eu.api.com {
            status_path /api/*
            fail_duration 10s
            response_timeout 3s

            # Different timeouts for different regions
            health_check https://us-east.api.com {
                path /health
                interval 30s
                timeout 2s
            }

            health_check https://us-west.api.com {
                path /health
                interval 30s
                timeout 3s
            }

            health_check https://eu.api.com {
                path /health
                interval 60s
                timeout 5s
            }
        }
    }
}
```

## Troubleshooting

### Enable Debug Logging

```caddyfile
{
    debug
}
```

Or run Caddy with debug flag:
```bash
caddy run --config Caddyfile --debug
```

### Common Issues

1. **Directive ordering errors**
   - Solution: Ensure global `order` directives are set correctly
   - Use `route` instead of `handle` to avoid ordering issues

2. **Status endpoint returns empty array**
   - Solution: Add `status_path` to your failover_proxy configurations

3. **Swagger UI not loading**
   - Solution: Use `/api/docs*` with wildcard to handle trailing slashes

4. **Health checks not running**
   - Solution: Verify health check configuration and endpoint accessibility
   - Ensure each `health_check` specifies its upstream URL

5. **Failover not triggering**
   - Solution: Check `fail_duration` setting and debug logs

6. **"Unknown subdirective" errors**
   - Common mistakes:
     - Using `timeout` instead of `dial_timeout` or `response_timeout`
     - Using `retry_count` or `retry_delay` (not supported)
     - Incorrect `health_check` syntax (missing upstream URL)
   - Solution: Check the Configuration Options Reference for valid directives

## Additional Documentation

For more detailed information, see the `docs/` directory:

- [Docker Usage](docs/DOCKER.md) - Detailed Docker build and deployment
- [xcaddy Usage](docs/XCADDY_USAGE.md) - Building with xcaddy
- [Directive Ordering](docs/DIRECTIVE_ORDERING.md) - Understanding Caddy directives
- [OpenAPI Design](docs/OPENAPI_DESIGN.md) - API documentation architecture
- [Semantic Versioning](docs/SEMANTIC_VERSIONING.md) - Version management
- [Git Hooks](docs/GIT_HOOKS.md) - Development workflow
- [GitHub App Setup](docs/GITHUB_APP_SETUP.md) - GitHub Apps configuration
- [Branch Protection](docs/BRANCH_PROTECTION.md) - Repository protection rules

---

## Development

### Building from Source

#### Prerequisites

- Go 1.22 or later
- xcaddy (for building Caddy with plugins)

#### Build Steps

```bash
# Clone the repository
git clone https://github.com/ejlevin1/caddy-failover.git
cd caddy-failover

# Build with xcaddy
xcaddy build --with github.com/ejlevin1/caddy-failover=.

# Or build Docker image
docker build -t caddy-failover .
```

### Testing

The plugin includes comprehensive tests with convenient test runners.

#### Quick Test Commands

```bash
# Run all tests
./scripts/test.sh all

# Run unit tests only
./scripts/test.sh unit

# Run with coverage
./scripts/test.sh coverage

# Run with race detector
./scripts/test.sh race

# Run benchmarks
./scripts/test.sh benchmark

# Run integration tests
./scripts/test.sh integration
```

#### Test Structure

```
.
├── integration_test.go         # Integration tests with caddytest
├── failover/
│   ├── failover_test.go       # Core unit tests
│   ├── failover_benchmark_test.go  # Performance benchmarks
│   └── handler_test.go        # HTTP handler tests
├── scripts/
│   └── test.sh                # Test runner script
└── test/
    ├── test.sh                # Docker-based integration tests
    ├── test-failover-logs.sh  # Test logging and health checks
    └── test-failover-status.sh # Test status endpoint
```

#### Manual Testing

```bash
# Run specific tests
go test -v -run TestFailoverProxy ./...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### License

Apache 2.0 - See [LICENSE](LICENSE) for details.
