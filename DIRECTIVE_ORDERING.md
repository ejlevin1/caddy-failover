# Directive Ordering in Caddy Failover Plugin

## The Issue

When using the `failover_status` directive within a `handle` block, you may encounter this error:

```
Error: adapting config using caddyfile: parsing caddyfile tokens for 'handle':
directive 'failover_status' is not an ordered HTTP handler
```

This occurs because Caddy needs to know the order in which to execute directives within a `handle` block.

## Solutions

### Solution 1: Global Order Directive (Recommended)

Add the order directives to your global options block:

```caddyfile
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }

    handle /api/* {
        failover_proxy http://backend1:8080 http://backend2:8080 {
            fail_duration 10s
        }
    }
}
```

### Solution 2: Use `route` Instead of `handle`

The `route` directive doesn't require ordering:

```caddyfile
:8080 {
    route /admin/failover/status {
        failover_status
    }

    route /api/* {
        failover_proxy http://backend1:8080 http://backend2:8080 {
            fail_duration 10s
        }
    }
}
```

## Why This Happens

Caddy's `handle` directive requires all contained directives to have a defined order. This ensures predictable request processing. The `failover_proxy` and `failover_status` directives are custom handlers that need to be explicitly ordered.

The `route` directive, on the other hand, executes directives in the order they appear in the Caddyfile, so no explicit ordering is needed.

## Best Practices

1. **Always include order directives** in your global options when using `failover_proxy` and `failover_status`
2. **Use consistent ordering** across your configuration
3. **Consider using `route`** for simpler configurations where execution order is clear

## Full Example

Here's a complete working example with proper ordering:

```caddyfile
{
    # Required ordering for failover directives
    order failover_proxy before reverse_proxy
    order failover_status before respond

    # Optional: enable debug logging
    debug
}

# Main site
:443 {
    # Admin status endpoint
    handle /admin/failover/status {
        failover_status
    }

    # API with failover
    handle /api/* {
        failover_proxy http://api1.local:8080 http://api2.local:8080 {
            fail_duration 30s
            dial_timeout 2s
            response_timeout 5s

            health_check http://api1.local:8080 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }

            health_check http://api2.local:8080 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }
        }
    }

    # Default handler
    handle {
        respond "Welcome" 200
    }
}
```

## Testing Your Configuration

Before deploying, always validate your Caddyfile:

```bash
caddy validate --config Caddyfile --adapter caddyfile
```

Or with Docker:

```bash
docker run --rm -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
    ghcr.io/ejlevin1/caddy-failover:latest \
    caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
```
