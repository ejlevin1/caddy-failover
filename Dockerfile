# ARG for using pre-built binaries from cache
# Can be set to "builder" (default) or a registry URL for cached builder image
ARG BUILDER_CACHE_IMAGE=builder
ARG BUILDER_LOADED_CACHE_IMAGE=builder-loaded

# ============================================================================
# Stage: builder
# Build Caddy with the failover plugin only
# ============================================================================
FROM caddy:2-builder AS builder

# Copy plugin source to build context
WORKDIR /src
COPY . .

# Build Caddy with the plugin
RUN xcaddy build \
    --with github.com/ejlevin1/caddy-failover=.

# ============================================================================
# Stage: builder-loaded
# Build Caddy with failover plugin plus additional plugins
# ============================================================================
FROM caddy:2-builder AS builder-loaded

# Copy plugin source to build context
WORKDIR /src
COPY . .

# Build Caddy with failover plugin plus additional plugins
RUN xcaddy build \
    --with github.com/ejlevin1/caddy-failover=. \
    --with github.com/gsmlg-dev/caddy-admin-ui \
    --with github.com/lucaslorentz/caddy-docker-proxy/v2

# ============================================================================
# Stage: builder-cache
# Cacheable builder stage that can be pulled from registry
# ============================================================================
FROM ${BUILDER_CACHE_IMAGE} AS builder-cache

# ============================================================================
# Stage: builder-loaded-cache
# Cacheable builder stage for loaded variant
# ============================================================================
FROM ${BUILDER_LOADED_CACHE_IMAGE} AS builder-loaded-cache

# ============================================================================
# Stage: git-info
# Generate build-info.json files for both standard and loaded variants
# ============================================================================
FROM caddy:2-alpine AS git-info
RUN apk add --no-cache git jq

WORKDIR /workspace
COPY .git .git

# Copy a built caddy binary to get version info
COPY --from=builder-cache /src/caddy /tmp/caddy

# Build arguments that may be provided by CI/CD
ARG VERSION=unknown
ARG GIT_TAG=unknown

# Generate git metadata and build-info files
RUN VERSION="${VERSION}" && \
    GIT_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')" && \
    GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')" && \
    GIT_TAG="${GIT_TAG:-$(git describe --tags --exact-match 2>/dev/null || echo 'unknown')}" && \
    BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo 'unknown')" && \
    CADDY_VERSION="$(/tmp/caddy version | head -1)" && \
    echo "{\"version\":\"$VERSION\",\"git_commit\":\"$GIT_COMMIT\",\"git_tag\":\"$GIT_TAG\",\"git_branch\":\"$GIT_BRANCH\",\"build_date\":\"$BUILD_DATE\",\"caddy_version\":\"$CADDY_VERSION\",\"plugin\":\"github.com/ejlevin1/caddy-failover\"}" | jq . > /build-info-standard.json && \
    echo "{\"version\":\"$VERSION\",\"git_commit\":\"$GIT_COMMIT\",\"git_tag\":\"$GIT_TAG\",\"git_branch\":\"$GIT_BRANCH\",\"build_date\":\"$BUILD_DATE\",\"caddy_version\":\"$CADDY_VERSION\",\"plugins\":[\"github.com/ejlevin1/caddy-failover\",\"github.com/gsmlg-dev/caddy-admin-ui\",\"github.com/lucaslorentz/caddy-docker-proxy/v2\"]}" | jq . > /build-info-loaded.json

# ============================================================================
# Stage: loaded
# Final stage for loaded image (with additional plugins)
# ============================================================================
FROM caddy:2-alpine AS loaded

# Copy custom Caddy binary with all plugins (from cache or local build)
COPY --from=builder-loaded-cache /src/caddy /usr/bin/caddy

# Copy build info
COPY --from=git-info /build-info-loaded.json /etc/caddy/build-info.json
RUN chmod 644 /etc/caddy/build-info.json

# Copy license file for compliance
COPY LICENSE /usr/share/licenses/caddy-failover/LICENSE

# Add labels for OCI image spec
LABEL org.opencontainers.image.source="https://github.com/ejlevin1/caddy-failover" \
      org.opencontainers.image.title="Caddy with Failover Plugin and Additional Modules" \
      org.opencontainers.image.description="Caddy web server with failover, admin UI, and Docker proxy plugins" \
      org.opencontainers.image.licenses="MIT"

# Expose ports
EXPOSE 80 443 2019

# Run Caddy
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]

# ============================================================================
# Stage: standard (default)
# Final stage for standard image - this is the default target
# ============================================================================
FROM caddy:2-alpine AS standard

# Copy custom Caddy binary (from cache or local build)
COPY --from=builder-cache /src/caddy /usr/bin/caddy

# Copy build info
COPY --from=git-info /build-info-standard.json /etc/caddy/build-info.json
RUN chmod 644 /etc/caddy/build-info.json

# Copy license file for compliance
COPY LICENSE /usr/share/licenses/caddy-failover/LICENSE

# Add labels for OCI image spec
LABEL org.opencontainers.image.source="https://github.com/ejlevin1/caddy-failover" \
      org.opencontainers.image.title="Caddy with Failover Plugin" \
      org.opencontainers.image.description="Caddy web server with intelligent failover plugin" \
      org.opencontainers.image.licenses="MIT"

# Expose ports
EXPOSE 80 443 2019

# Run Caddy
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
