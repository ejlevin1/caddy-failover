#!/bin/bash

# Test script to verify directive ordering fix

set -e

echo "Testing failover_status directive ordering issue"
echo "================================================"

# Build the Docker image
echo "Building Docker image..."
docker build -t caddy-failover:ordering-test .

# Test 1: Without order directive (should fail)
echo ""
echo "Test 1: Configuration WITHOUT order directive (expected to fail)..."
cat > test-no-order.Caddyfile <<EOF
:8080 {
    handle /admin/failover/status {
        failover_status
    }
}
EOF

echo "Testing configuration validation..."
if docker run --rm -v $(pwd)/test-no-order.Caddyfile:/etc/caddy/Caddyfile caddy-failover:ordering-test caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1 | grep -q "not an ordered HTTP handler"; then
    echo "✅ Test 1 PASSED: Correctly failed without order directive"
else
    echo "❌ Test 1 FAILED: Should have failed without order directive"
fi

# Test 2: With order directive (should succeed)
echo ""
echo "Test 2: Configuration WITH order directive (expected to succeed)..."
cat > test-with-order.Caddyfile <<EOF
{
    order failover_status before respond
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }
}
EOF

echo "Testing configuration validation..."
if docker run --rm -v $(pwd)/test-with-order.Caddyfile:/etc/caddy/Caddyfile caddy-failover:ordering-test caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1 | grep -q "Valid configuration"; then
    echo "✅ Test 2 PASSED: Configuration is valid with order directive"
else
    echo "❌ Test 2 FAILED: Configuration should be valid with order directive"
    docker run --rm -v $(pwd)/test-with-order.Caddyfile:/etc/caddy/Caddyfile caddy-failover:ordering-test caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
fi

# Test 3: Using route instead of handle (should succeed)
echo ""
echo "Test 3: Using route instead of handle (expected to succeed)..."
cat > test-with-route.Caddyfile <<EOF
:8080 {
    route /admin/failover/status {
        failover_status
    }
}
EOF

echo "Testing configuration validation..."
if docker run --rm -v $(pwd)/test-with-route.Caddyfile:/etc/caddy/Caddyfile caddy-failover:ordering-test caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1 | grep -q "Valid configuration"; then
    echo "✅ Test 3 PASSED: Configuration is valid with route directive"
else
    echo "❌ Test 3 FAILED: Configuration should be valid with route directive"
    docker run --rm -v $(pwd)/test-with-route.Caddyfile:/etc/caddy/Caddyfile caddy-failover:ordering-test caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
fi

# Test 4: Full working example
echo ""
echo "Test 4: Full working example with failover_status endpoint..."
cat > test-full.Caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }

    handle /api/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
        }
    }

    handle {
        respond "OK" 200
    }
}
EOF

echo "Starting test container..."
docker run --rm -d \
    --name caddy-ordering-test \
    -v $(pwd)/test-full.Caddyfile:/etc/caddy/Caddyfile \
    -p 8083:8080 \
    caddy-failover:ordering-test

echo "Waiting for container to start..."
sleep 3

echo "Testing status endpoint..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8083/admin/failover/status)
if [ "$response" = "200" ]; then
    echo "✅ Test 4 PASSED: Status endpoint returns 200"
else
    echo "❌ Test 4 FAILED: Status endpoint returned $response instead of 200"
fi

echo "Stopping test container..."
docker stop caddy-ordering-test

echo ""
echo "Cleaning up test files..."
rm -f test-*.Caddyfile

echo ""
echo "================================================"
echo "Directive ordering tests completed!"
