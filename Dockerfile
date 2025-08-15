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

# Copy custom Caddy binary
COPY --from=builder /src/caddy /usr/bin/caddy

# Expose ports
EXPOSE 80 443 2019

# Run Caddy
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
