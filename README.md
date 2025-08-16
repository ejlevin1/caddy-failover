# Caddy Failover Plugin

A Caddy plugin that provides intelligent failover between multiple upstream servers with support for mixed HTTP/HTTPS schemes.

## Features

- **Mixed-scheme failover**: Supports both HTTP and HTTPS upstreams in the same directive
- **Intelligent failover**: Automatically fails over to next upstream on connection errors or 5xx responses
- **Health checks**: Optional per-upstream health checks to proactively detect unhealthy servers
- **Per-upstream headers**: Configure different headers for each upstream
- **Failure caching**: Remembers failed upstreams for configurable duration to avoid repeated attempts
- **TLS configuration**: Supports skipping certificate verification for development environments
- **Path base support**: Upstreams can have different base paths that are preserved in routing
- **Environment variables**: Supports environment variable expansion in upstream URLs and header values
- **Debug logging**: Comprehensive logging for troubleshooting upstream selection

## Installation

### Using xcaddy

Build Caddy with this plugin using [xcaddy](https://github.com/caddyserver/xcaddy):

```bash
xcaddy build --with github.com/ejlevin1/caddy-failover
```

Or from a local clone:

```bash
xcaddy build --with github.com/ejlevin1/caddy-failover=.
```

### Using Docker

Pull the pre-built image with semantic versioning tags:

```bash
# Latest version
docker pull ghcr.io/ejlevin1/caddy-failover:latest

# Specific version
docker pull ghcr.io/ejlevin1/caddy-failover:1.6.1

# Major version (gets latest 1.x.x)
docker pull ghcr.io/ejlevin1/caddy-failover:1
```

Or build your own:

```bash
docker build -t caddy-failover .
```

## Usage

### Caddyfile Configuration

First, add the global order directive to tell Caddy where to place the `failover_proxy` handler:

```caddyfile
{
    order failover_proxy before reverse_proxy
}
```

Then use the `failover_proxy` directive in your site blocks:

```caddyfile
https://localhost:443 {
    handle /api/* {
        failover_proxy http://localhost:3000 https://api.example.com {
            fail_duration 3s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify

            # Headers for HTTP upstream
            header_up http://localhost:3000 X-Source local
            header_up http://localhost:3000 Host api.example.com

            # Headers for HTTPS upstream
            header_up https://api.example.com X-Source remote
        }
    }
}
```

### Configuration Options

- `fail_duration` - How long to remember failed upstreams (default: 30s)
- `dial_timeout` - Connection timeout (default: 2s)
- `response_timeout` - Response timeout (default: 5s)
- `insecure_skip_verify` - Skip TLS certificate verification
- `header_up <upstream> <name> <value>` - Set upstream-specific headers
- `health_check <upstream> { ... }` - Configure health checks for an upstream
- `status_path <path>` - Set the path identifier for status reporting

#### Health Check Options

Configure health checks for each upstream to proactively detect unhealthy servers:

```caddyfile
health_check <upstream_url> {
    path /health          # Health check endpoint (default: /health)
    interval 30s          # Check interval (default: 30s)
    timeout 5s            # Check timeout (default: 5s)
    expected_status 200   # Expected HTTP status (default: 200)
}
```

### Path Base Support

Upstreams can have different base paths. The plugin automatically preserves and combines the upstream base path with the incoming request path:

```caddyfile
:8080 {
    handle {
        # Request to /gateway will be proxied to:
        # - http://test.com/gateway (no base path)
        # - http://test2.com/path/gateway (with /path base)
        failover_proxy http://test.com http://test2.com/path/
    }
}
```

### X-Forwarded-Proto Header

The plugin correctly sets the `X-Forwarded-Proto` header based on the actual inbound protocol (HTTP or HTTPS), not hardcoded. It also preserves any existing `X-Forwarded-Proto` header from upstream proxies.

### Debug Logging

The plugin provides detailed debug logging to help troubleshoot failover behavior. To enable debug logging in Caddy, use:

```bash
# Start Caddy with debug logging
caddy run --config Caddyfile --adapter caddyfile --environ --debug

# Or set the log level in your Caddyfile
{
    debug
}
```

The plugin logs:
- Which upstream is being attempted
- The full target URL being proxied to
- Success/failure of each upstream attempt
- When upstreams are skipped due to previous failures

## Examples

### IDE-First Development

Route to local IDE first, fall back to remote development server:

```caddyfile
{
    order failover_proxy before reverse_proxy
}

:443 {
    handle /admin/* {
        failover_proxy http://localhost:5041 https://dev.example.com {
            fail_duration 3s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify

            header_up http://localhost:5041 X-Environment local-ide
            header_up https://dev.example.com X-Environment remote-dev
        }
    }
}
```

### Multi-Tier Failover

Try local, then Docker, then production:

```caddyfile
{
    order failover_proxy before reverse_proxy
}

:443 {
    handle /api/* {
        failover_proxy http://localhost:3000 http://api:3000 https://api.production.com {
            fail_duration 5s
            dial_timeout 2s
            response_timeout 10s

            header_up http://localhost:3000 X-Tier local
            header_up http://api:3000 X-Tier docker
            header_up https://api.production.com X-Tier production
        }
    }
}
```

### With Health Checks

Configure health checks to proactively detect unhealthy upstreams:

```caddyfile
{
    order failover_proxy before reverse_proxy
}

:443 {
    handle /api/* {
        failover_proxy http://primary.local https://backup.cloud {
            fail_duration 10s

            # Health check for primary
            health_check http://primary.local {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }

            # Health check for backup
            health_check https://backup.cloud {
                path /status
                interval 60s
                timeout 10s
                expected_status 204
            }
        }
    }
}
```

### Path Base Support

Upstreams can have different base paths that are preserved:

```caddyfile
{
    order failover_proxy before reverse_proxy
}

:443 {
    handle /gateway/* {
        # Request to /gateway/api becomes:
        # - http://service1.local/gateway/api
        # - http://service2.local/v2/gateway/api
        failover_proxy http://service1.local http://service2.local/v2 {
            fail_duration 5s
        }
    }
}
```

### Environment Variables

The plugin supports environment variable expansion in upstream URLs and header values using Caddy's standard `{env.VARIABLE_NAME}` syntax:

```caddyfile
{
    order failover_proxy before reverse_proxy
}

:443 {
    handle /api/* {
        # Environment variables in upstream URLs
        failover_proxy http://{env.PRIMARY_HOST}:3000 https://{env.BACKUP_HOST} {
            fail_duration 5s

            # Environment variables in header values
            header_up http://{env.PRIMARY_HOST}:3000 X-Environment {env.ENVIRONMENT}
            header_up http://{env.PRIMARY_HOST}:3000 X-API-Key {env.API_KEY}
            header_up https://{env.BACKUP_HOST} X-Environment production
        }
    }
}
```

Set the environment variables when running Caddy:

```bash
export PRIMARY_HOST=localhost
export BACKUP_HOST=api.example.com
export ENVIRONMENT=development
export API_KEY=secret-key-123

caddy run --config Caddyfile
```

Or with Docker:

```bash
docker run -d \
    -e PRIMARY_HOST=host.docker.internal \
    -e BACKUP_HOST=api.example.com \
    -e ENVIRONMENT=development \
    -e API_KEY=secret-key-123 \
    -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
    -p 443:443 \
    ghcr.io/ejlevin1/caddy-failover:latest
```

### Status API

Monitor the health and status of all failover proxies via REST API:

**Important**: The `failover_status` directive requires proper ordering. Choose one of these approaches:

**Option 1: Use global order directive (recommended)**
```caddyfile
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:443 {
    # Status endpoint
    handle /admin/failover/status {
        failover_status
    }

    # Proxies with status tracking
    handle /auth/* {
        failover_proxy http://auth1.local http://auth2.local {
            # status_path is recommended for clear path identification
            # If not specified, an auto-generated identifier will be used
            status_path /auth/*
            health_check http://auth1.local {
                path /health
                interval 30s
            }
        }
    }
}
```

**Option 2: Use route instead of handle (no order directive needed)**
```caddyfile
:443 {
    # Status endpoint using route (doesn't require ordering)
    route /admin/failover/status {
        failover_status
    }
}
```

The status endpoint returns JSON with the current state of all upstreams:

```json
[
    {
        "path": "/auth/*",
        "failover_proxies": [
            {
                "host": "http://auth1.local",
                "status": "UP",
                "health_check_enabled": true,
                "last_check": "2024-01-15T10:30:45Z",
                "response_time_ms": 125
            },
            {
                "host": "http://auth2.local",
                "status": "DOWN",
                "health_check_enabled": false,
                "last_failure": "2024-01-15T10:29:30Z"
            }
        ]
    }
]
```

Status values:
- `UP` - Upstream is healthy and accepting requests
- `DOWN` - Upstream failed recently and is in failure cache
- `UNHEALTHY` - Health check is failing for this upstream

## Building from Source

### Prerequisites

- Go 1.22 or later
- xcaddy (optional, for building Caddy with plugins)

### Build Steps

1. Clone the repository:
```bash
git clone https://github.com/ejlevin1/caddy-failover.git
cd caddy-failover
```

2. Build with xcaddy:
```bash
xcaddy build --with github.com/ejlevin1/caddy-failover=.
```

3. Or build the Docker image:
```bash
docker build -t caddy-failover .
```

## Testing

Run the test suite:

```bash
./test/test.sh
```

Or use the Makefile:

```bash
make test
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
