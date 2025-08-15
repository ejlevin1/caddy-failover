#!/bin/bash

# Test script to verify failover warning logs and health check user agent

set -e

echo "Testing failover warning logs and health check user agent..."
echo "=================================================="

# Build the plugin
echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

# Create a mock server that logs headers
echo "Creating mock server with header logging..."
cat > mock-server-with-headers.js <<'EOF'
const http = require('http');

const server = http.createServer((req, res) => {
  // Log request with headers
  console.log(`[${new Date().toISOString()}] ${req.method} ${req.url}`);
  console.log(`User-Agent: ${req.headers['user-agent'] || 'not set'}`);

  // Health check endpoint
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('OK');
    return;
  }

  // Normal response
  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({
    method: req.method,
    url: req.url,
    headers: req.headers,
    timestamp: new Date().toISOString()
  }));
});

const port = process.env.PORT || 3000;
server.listen(port, () => {
  console.log(`Mock server listening on port ${port}`);
});
EOF

# Start mock server
echo "Starting mock server..."
docker run --rm -d \
    --name mock-server-headers \
    --network bridge \
    -p 3002:3000 \
    -v $(pwd)/mock-server-with-headers.js:/app/server.js \
    node:20-alpine \
    node /app/server.js

# Get mock server IP
MOCK_SERVER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' mock-server-headers)
echo "Mock server IP: $MOCK_SERVER_IP"

# Create Caddyfile with health checks and failover
cat > test-failover-logs.Caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    debug
}

:8080 {
    handle /test {
        # First upstream will always fail, forcing failover
        failover_proxy http://localhost:9999 http://$MOCK_SERVER_IP:3000 {
            fail_duration 5s
            dial_timeout 1s
            response_timeout 2s

            # Health check for the mock server
            health_check http://$MOCK_SERVER_IP:3000 {
                path /health
                interval 2s
                timeout 1s
                expected_status 200
            }
        }
    }
}
EOF

# Start Caddy
echo "Starting Caddy..."
docker run --rm -d \
    --name caddy-failover-test \
    --network bridge \
    -v $(pwd)/test-failover-logs.Caddyfile:/etc/caddy/Caddyfile \
    -p 8091:8080 \
    caddy-failover:test

echo "Waiting for services to start and health checks to run..."
sleep 4

echo ""
echo "Test 1: Checking health check User-Agent..."
echo "-------------------------------------------"
docker logs mock-server-headers 2>&1 | grep "User-Agent:" | head -3
echo ""
echo "✅ Health check User-Agent should be: Caddy-failover-health-check/1.0"

echo ""
echo "Test 2: Testing failover warning log..."
echo "----------------------------------------"
echo "Making request that will trigger failover..."
curl -s http://localhost:8091/test > /dev/null

echo ""
echo "Caddy logs showing failover warning:"
docker logs caddy-failover-test 2>&1 | grep -E "(failing over|upstream failed)" | tail -5
echo ""
echo "✅ Should see 'failing over to alternate upstream' warning"

echo ""
echo "Test 3: Verify request succeeded despite first upstream failure..."
echo "-------------------------------------------------------------------"
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8091/test)
if [ "$response" = "200" ]; then
    echo "✅ Request succeeded with failover (HTTP $response)"
else
    echo "❌ Request failed (HTTP $response)"
    docker logs caddy-failover-test
fi

echo ""
echo "Cleaning up..."
docker stop mock-server-headers caddy-failover-test 2>/dev/null || true
rm -f test-failover-logs.Caddyfile mock-server-with-headers.js

echo ""
echo "=================================================="
echo "✅ Failover logging tests completed!"
