# Caddy Failover Plugin

A Caddy plugin that provides intelligent failover between multiple upstream servers with support for mixed HTTP/HTTPS schemes.

## Features

- **Mixed-scheme failover**: Supports both HTTP and HTTPS upstreams in the same directive
- **Intelligent failover**: Automatically fails over to next upstream on connection errors or 5xx responses
- **Per-upstream headers**: Configure different headers for each upstream
- **Failure caching**: Remembers failed upstreams for configurable duration to avoid repeated attempts
- **TLS configuration**: Supports skipping certificate verification for development environments

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

Pull the pre-built image:

```bash
docker pull ghcr.io/ejlevin1/caddy-failover:latest
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
