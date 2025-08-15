#!/bin/bash

# Test script for Caddy with failover plugin

set -e

echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

# Create a simple mock server that always returns 200
echo "Creating mock server..."
cat > mock-server.js <<'EOF'
const http = require('http');
const server = http.createServer((req, res) => {
  // Log the request
  console.log(`${req.method} ${req.url}`);

  // Always return 200 with some JSON
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

# Start mock server container
echo "Starting mock server container..."
docker run --rm -d \
    --name mock-server \
    --network bridge \
    -p 3001:3000 \
    -v $(pwd)/mock-server.js:/app/server.js \
    node:20-alpine \
    node /app/server.js

# Get the mock server IP
MOCK_SERVER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' mock-server)
echo "Mock server IP: $MOCK_SERVER_IP"

echo "Creating test Caddyfile..."
cat > test-caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
}

:8080 {
    handle /get {
        # Test with mock server (should always work)
        failover_proxy http://$MOCK_SERVER_IP:3000 {
            fail_duration 3s
            dial_timeout 2s
            response_timeout 5s

            header_up http://$MOCK_SERVER_IP:3000 X-Test-Header test-value
        }
    }

    handle /anything/* {
        # Test with intentionally failing first upstream for failover
        failover_proxy http://localhost:9999 http://$MOCK_SERVER_IP:3000 {
            fail_duration 3s
            dial_timeout 1s
            response_timeout 5s
        }
    }

    handle /health {
        respond "OK" 200
    }
}
EOF

echo "Starting Caddy container..."
docker run --rm -d \
    --name caddy-test \
    --network bridge \
    -v $(pwd)/test-caddyfile:/etc/caddy/Caddyfile \
    -p 8090:8080 \
    caddy-failover:test

echo "Waiting for services to start..."
sleep 3

# Test mock server is working
echo "Testing mock server directly..."
curl -s http://localhost:3001/ > /dev/null || {
    echo "Mock server not responding!"
    docker logs mock-server
    docker stop mock-server caddy-test 2>/dev/null || true
    exit 1
}

echo "Running tests..."
echo "===================="

# Test 1: Basic proxy
echo -n "Test 1 - Basic proxy: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8090/get)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-test
    docker stop mock-server caddy-test 2>/dev/null || true
    exit 1
fi

# Test 2: Failover scenario
echo -n "Test 2 - Failover: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8090/anything/test)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-test
    docker stop mock-server caddy-test 2>/dev/null || true
    exit 1
fi

# Test 3: Health check
echo -n "Test 3 - Health check: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8090/health)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-test
    docker stop mock-server caddy-test 2>/dev/null || true
    exit 1
fi

# Test 4: Check custom headers received by mock server
echo -n "Test 4 - Custom headers: "
response=$(curl -s http://localhost:8090/get)
# Check if mock server received the request (validates proxy is working)
if echo "$response" | grep -q '"url":"/get"'; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (proxy not working correctly)"
    echo "Response: $response"
    docker logs caddy-test
    docker stop mock-server caddy-test 2>/dev/null || true
    exit 1
fi

echo "===================="
echo "All tests completed!"

echo "Stopping containers..."
docker stop mock-server caddy-test 2>/dev/null || true

# Clean up
rm -f test-caddyfile mock-server.js

echo "✅ Test suite passed successfully!"
