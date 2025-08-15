# Using with xcaddy

This plugin is designed to work seamlessly with [xcaddy](https://github.com/caddyserver/xcaddy), the official Caddy build tool.

## Installation Methods

### 1. Build from GitHub (Recommended for Production)

```bash
xcaddy build --with github.com/ejlevin1/caddy-failover
```

This downloads the latest version from GitHub and builds Caddy with the plugin.

### 2. Build from Local Source (Development)

```bash
# Clone the repository
git clone https://github.com/ejlevin1/caddy-failover.git
cd caddy-failover

# Build with local source
xcaddy build --with github.com/ejlevin1/caddy-failover=.
```

### 3. Build with Specific Version

```bash
xcaddy build --with github.com/ejlevin1/caddy-failover@v1.0.0
```

### 4. Build with Multiple Plugins

```bash
xcaddy build \
    --with github.com/ejlevin1/caddy-failover \
    --with github.com/caddyserver/transform-encoder \
    --with github.com/caddy-dns/cloudflare
```

## Development Workflow

### Quick Development Cycle

1. Make changes to `failover.go`
2. Build and test:
   ```bash
   make xcaddy-build
   ./caddy validate --config examples/basic-caddyfile --adapter caddyfile
   ```

### Running with Test Config

```bash
# Build and run in one command
make xcaddy-run

# Or manually
xcaddy build --with github.com/ejlevin1/caddy-failover=.
./caddy run --config examples/basic-caddyfile --adapter caddyfile
```

### Testing Changes

```bash
# Run Go tests
go test -v ./...

# Build and validate
xcaddy build --with github.com/ejlevin1/caddy-failover=.
./caddy list-modules | grep failover_proxy
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Install xcaddy
  run: go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

- name: Build Caddy
  run: xcaddy build --with github.com/ejlevin1/caddy-failover
```

### Docker Multi-stage Build

```dockerfile
FROM caddy:2-builder AS builder
WORKDIR /src
COPY . .
RUN xcaddy build --with github.com/ejlevin1/caddy-failover=.

FROM caddy:2-alpine
COPY --from=builder /src/caddy /usr/bin/caddy
```

## Module Path

The plugin registers as:
- **Module ID**: `http.handlers.failover_proxy`
- **Caddyfile Directive**: `failover_proxy`

## Verification

After building, verify the plugin is included:

```bash
# List all modules
./caddy list-modules

# Check for failover_proxy
./caddy list-modules | grep failover

# Validate a Caddyfile
./caddy validate --config your-caddyfile --adapter caddyfile
```

## Troubleshooting

### Plugin Not Found

If xcaddy can't find the plugin:
1. Ensure `go.mod` is in the repository root
2. Check that the module path matches: `github.com/ejlevin1/caddy-failover`
3. For local builds, use the full path: `--with github.com/ejlevin1/caddy-failover=/full/path/to/repo`

### Build Errors

1. Update Go modules:
   ```bash
   go mod tidy
   ```

2. Clear build cache:
   ```bash
   go clean -modcache
   xcaddy build --with github.com/ejlevin1/caddy-failover=.
   ```

### Directive Not Recognized

Ensure you have the order directive in your Caddyfile:
```caddyfile
{
    order failover_proxy before reverse_proxy
}
```
