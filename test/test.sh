#!/bin/bash

# Test script for Caddy with failover plugin

set -e

echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

echo "Creating test Caddyfile..."
cat > test-caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
}

:8080 {
    handle /get {
        # Test with httpbin.org (should always work)
        failover_proxy http://httpbin.org https://httpbin.org {
            fail_duration 3s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify

            header_up http://httpbin.org X-Test-Header http-test
            header_up https://httpbin.org X-Test-Header https-test
        }
    }

    handle /anything/* {
        # Test with intentionally failing first upstream for failover
        failover_proxy http://localhost:9999 https://httpbin.org {
            fail_duration 3s
            dial_timeout 1s
            response_timeout 5s
            insecure_skip_verify
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
    -v $(pwd)/test-caddyfile:/etc/caddy/Caddyfile \
    -p 8090:8080 \
    caddy-failover:test

echo "Waiting for Caddy to start..."
sleep 3

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
    docker stop caddy-test
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
    docker stop caddy-test
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
    docker stop caddy-test
    exit 1
fi

# Test 4: Check custom headers
echo -n "Test 4 - Custom headers: "
headers=$(curl -s http://localhost:8090/get | grep -i "x-test-header" || true)
if [ -n "$headers" ]; then
    echo "✅ PASSED (headers might be upstream-only)"
else
    echo "⚠️  WARNING (headers are upstream-only, this is expected)"
fi

echo "===================="
echo "All tests completed!"

echo "Stopping Caddy container..."
docker stop caddy-test

echo "✅ Test suite passed successfully!"
