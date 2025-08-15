# Docker Images

The Caddy Failover plugin is available as a Docker image with multi-architecture support.

## Available Tags

Docker images are published to GitHub Container Registry with semantic versioning:

- `ghcr.io/ejlevin1/caddy-failover:latest` - Latest stable release
- `ghcr.io/ejlevin1/caddy-failover:1` - Latest v1.x.x release
- `ghcr.io/ejlevin1/caddy-failover:1.3` - Latest v1.3.x release
- `ghcr.io/ejlevin1/caddy-failover:1.3.0` - Specific version
- `ghcr.io/ejlevin1/caddy-failover:main` - Latest build from main branch (development)

## Architecture Support

All images support multiple architectures:
- `linux/amd64` - Intel/AMD 64-bit processors
- `linux/arm64` - ARM 64-bit processors (Apple Silicon, AWS Graviton, etc.)

Docker will automatically pull the correct architecture for your platform.

## Usage

### Basic Usage

```bash
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
  ghcr.io/ejlevin1/caddy-failover:latest
```

### With Environment Variables

```bash
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -e PRIMARY_HOST=localhost \
  -e BACKUP_HOST=api.example.com \
  -e ENVIRONMENT=production \
  -v $(pwd)/Caddyfile:/etc/caddy/Caddyfile \
  ghcr.io/ejlevin1/caddy-failover:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  caddy:
    image: ghcr.io/ejlevin1/caddy-failover:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    environment:
      - PRIMARY_HOST=app
      - BACKUP_HOST=backup-app
      - ENVIRONMENT=production
    restart: unless-stopped

volumes:
  caddy_data:
  caddy_config:
```

## Version Pinning

For production deployments, it's recommended to pin to a specific version:

```yaml
# Pin to specific version
image: ghcr.io/ejlevin1/caddy-failover:1.3.0

# Pin to minor version (gets patch updates)
image: ghcr.io/ejlevin1/caddy-failover:1.3

# Pin to major version (gets minor and patch updates)
image: ghcr.io/ejlevin1/caddy-failover:1
```

## Build Attestations

All Docker images include build attestations for supply chain security. You can verify the provenance of an image:

```bash
docker trust inspect ghcr.io/ejlevin1/caddy-failover:latest
```

## Development Builds

Development builds from the main branch are available but not recommended for production:

```bash
docker run ghcr.io/ejlevin1/caddy-failover:main
```

These builds include the latest features but may be unstable.
