# Build stage for Caddy with custom plugin
FROM caddy:2-builder AS builder

# Copy plugin source to build context
WORKDIR /src
COPY . .

# Build Caddy with the plugin
RUN xcaddy build \
    --with github.com/ejlevin1/caddy-failover=.

# Final stage
FROM caddy:2-alpine

# Build arguments for git information
ARG GIT_COMMIT=unknown
ARG GIT_TAG=unknown
ARG GIT_BRANCH=unknown
ARG BUILD_DATE=unknown
ARG VERSION=unknown

# Copy custom Caddy binary
COPY --from=builder /src/caddy /usr/bin/caddy

# Create build info file
RUN echo "{\
  \"version\": \"${VERSION}\",\
  \"git_commit\": \"${GIT_COMMIT}\",\
  \"git_tag\": \"${GIT_TAG}\",\
  \"git_branch\": \"${GIT_BRANCH}\",\
  \"build_date\": \"${BUILD_DATE}\",\
  \"caddy_version\": \"$(caddy version | head -1)\",\
  \"plugin\": \"github.com/ejlevin1/caddy-failover\"\
}" > /etc/caddy/build-info.json && \
    chmod 644 /etc/caddy/build-info.json

# Add labels for OCI image spec
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/ejlevin1/caddy-failover" \
      org.opencontainers.image.title="Caddy with Failover Plugin" \
      org.opencontainers.image.description="Caddy web server with intelligent failover plugin"

# Expose ports
EXPOSE 80 443 2019

# Run Caddy
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
