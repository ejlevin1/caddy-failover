#!/bin/bash

# Test script for status API functionality

set -e

echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

echo "Creating test Caddyfile with status endpoint..."
cat > test-status-caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
    debug
}

:8080 {
    # Status endpoint
    handle /api/failover/status {
        failover_status
    }

    handle /auth/* {
        failover_proxy http://localhost:3000 https://httpbin.org/anything {
            fail_duration 10s
            dial_timeout 2s
            response_timeout 5s
            status_path /auth/*

            # Health check for local (will fail)
            health_check http://localhost:3000 {
                path /health
                interval 5s
                timeout 2s
                expected_status 200
            }

            # Health check for httpbin (should succeed)
            health_check https://httpbin.org/anything {
                path /status/200
                interval 5s
                timeout 3s
                expected_status 200
            }
        }
    }

    handle /api/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
            status_path /api/*

            health_check http://httpbin.org/anything {
                path /status/200
                interval 10s
                timeout 3s
                expected_status 200
            }
        }
    }

    handle /health {
        respond "OK" 200
    }
}
EOF

echo "Starting Caddy container with status API..."
docker run --rm -d \
    --name caddy-status-test \
    -v $(pwd)/test-status-caddyfile:/etc/caddy/Caddyfile \
    -p 8082:8080 \
    caddy-failover:test \
    caddy run --config /etc/caddy/Caddyfile --adapter caddyfile --environ

echo "Waiting for Caddy to start and perform initial health checks..."
sleep 8

echo ""
echo "Testing status API endpoint..."
echo "========================================"
echo "GET http://localhost:8082/api/failover/status"
echo ""
response=$(curl -s http://localhost:8082/api/failover/status)
echo "$response" | python3 -m json.tool || echo "$response"
echo "========================================"

echo ""
echo "Making a request to trigger failover..."
curl -s -o /dev/null http://localhost:8082/auth/test
sleep 2

echo ""
echo "Checking status after request..."
echo "========================================"
echo "GET http://localhost:8082/api/failover/status"
echo ""
response=$(curl -s http://localhost:8082/api/failover/status)
echo "$response" | python3 -m json.tool || echo "$response"
echo "========================================"

echo ""
echo "Testing that proxied endpoints still work..."
echo -n "Test 1 - Auth endpoint: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/auth/test)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
fi

echo -n "Test 2 - API endpoint: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/api/test)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
fi

echo -n "Test 3 - Health endpoint: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8082/health)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
fi

echo ""
echo "Stopping Caddy container..."
docker stop caddy-status-test

echo ""
echo "✅ Status API test completed successfully!"
