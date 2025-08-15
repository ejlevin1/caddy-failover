#!/bin/bash

# Test script for health check functionality

set -e

echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

echo "Creating test Caddyfile with health checks..."
cat > test-healthcheck-caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    debug
}

:8080 {
    handle /test {
        # Configure with health checks
        failover_proxy http://httpbin.org https://httpbin.org {
            fail_duration 10s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify

            # Health check for first upstream
            health_check http://httpbin.org {
                path /status/200
                interval 5s
                timeout 3s
                expected_status 200
            }

            # Health check for second upstream
            health_check https://httpbin.org {
                path /status/200
                interval 5s
                timeout 3s
                expected_status 200
            }
        }
    }

    handle /unhealthy {
        # Test with one unhealthy upstream
        failover_proxy http://localhost:9999 https://httpbin.org {
            fail_duration 10s
            dial_timeout 2s
            response_timeout 5s

            # This will fail (server doesn't exist)
            health_check http://localhost:9999 {
                path /health
                interval 5s
                timeout 2s
                expected_status 200
            }

            # This should succeed
            health_check https://httpbin.org {
                path /status/200
                interval 5s
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

echo "Starting Caddy container with health checks..."
docker run --rm -d \
    --name caddy-healthcheck-test \
    -v $(pwd)/test-healthcheck-caddyfile:/etc/caddy/Caddyfile \
    -p 8081:8080 \
    caddy-failover:test \
    caddy run --config /etc/caddy/Caddyfile --adapter caddyfile --environ --debug

echo "Waiting for Caddy to start and perform initial health checks..."
sleep 10

echo "Checking Caddy logs for health check activity..."
echo "========================================="
docker logs caddy-healthcheck-test 2>&1 | grep -E "(health check|upstream became)" || true
echo "========================================="

echo ""
echo "Running tests..."
echo "=================="

# Test 1: Basic proxy with healthy upstreams
echo -n "Test 1 - Proxy with healthy upstreams: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/test/get)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-healthcheck-test
    docker stop caddy-healthcheck-test
    exit 1
fi

# Test 2: Proxy with one unhealthy upstream (should skip to healthy one)
echo -n "Test 2 - Skip unhealthy upstream: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/unhealthy/get)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-healthcheck-test
    docker stop caddy-healthcheck-test
    exit 1
fi

# Test 3: Health endpoint
echo -n "Test 3 - Health check endpoint: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/health)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker stop caddy-healthcheck-test
    exit 1
fi

echo ""
echo "Waiting to see periodic health checks..."
sleep 10

echo ""
echo "Final health check logs:"
echo "========================================="
docker logs caddy-healthcheck-test 2>&1 | grep -E "(health check|upstream became)" | tail -20 || true
echo "========================================="

echo ""
echo "Stopping Caddy container..."
docker stop caddy-healthcheck-test

echo ""
echo "✅ All health check tests passed successfully!"
