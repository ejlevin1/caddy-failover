#!/bin/bash

# Test script for verifying status tracking with and without status_path

set -e

echo "Testing failover_status tracking behavior"
echo "=========================================="

# Build the Docker image
echo "Building Docker image..."
docker build -t caddy-failover:status-test .

# Test 1: Without status_path (should auto-detect from handle path)
echo ""
echo "Test 1: Auto-detection of path from handle block..."
cat > test-auto-path.Caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
    debug
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }

    handle /admin/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
            # Note: no status_path specified - should auto-detect from handle
        }
    }

    handle /api/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
            # Note: no status_path specified - should auto-detect from handle
        }
    }
}
EOF

echo "Starting container with auto-path detection..."
docker run --rm -d \
    --name caddy-status-test \
    -v $(pwd)/test-auto-path.Caddyfile:/etc/caddy/Caddyfile \
    -p 8091:8080 \
    caddy-failover:status-test

echo "Waiting for container to start..."
sleep 3

echo "Checking status endpoint (auto-detected paths)..."
response=$(curl -s http://localhost:8091/admin/failover/status)
echo "Response: $response"

if [ "$response" = "[]" ] || [ "$response" = "null" ]; then
    echo "❌ Test 1 FAILED: No proxies registered (auto-detection failed)"
    docker logs caddy-status-test 2>&1 | tail -20
else
    echo "$response" | python3 -m json.tool
    echo "✅ Test 1 PASSED: Proxies registered with auto-detected paths"
fi

docker stop caddy-status-test

# Test 2: With explicit status_path
echo ""
echo "Test 2: Explicit status_path configuration..."
cat > test-explicit-path.Caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
    debug
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }

    handle /admin/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
            status_path /admin/*
        }
    }

    handle /api/* {
        failover_proxy http://httpbin.org/anything https://httpbin.org/anything {
            fail_duration 10s
            status_path /api/*
        }
    }
}
EOF

echo "Starting container with explicit status_path..."
docker run --rm -d \
    --name caddy-status-test \
    -v $(pwd)/test-explicit-path.Caddyfile:/etc/caddy/Caddyfile \
    -p 8091:8080 \
    caddy-failover:status-test

echo "Waiting for container to start..."
sleep 3

echo "Checking status endpoint (explicit paths)..."
response=$(curl -s http://localhost:8091/admin/failover/status)
echo "Response: $response"

if [ "$response" = "[]" ] || [ "$response" = "null" ]; then
    echo "❌ Test 2 FAILED: No proxies registered even with explicit status_path"
    docker logs caddy-status-test 2>&1 | tail -20
else
    echo "$response" | python3 -m json.tool
    echo "✅ Test 2 PASSED: Proxies registered with explicit paths"
fi

docker stop caddy-status-test

# Test 3: Test with health checks
echo ""
echo "Test 3: Status with health checks enabled..."
cat > test-health.Caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
}

:8080 {
    handle /admin/failover/status {
        failover_status
    }

    handle /admin/* {
        failover_proxy http://httpbin.org/anything http://localhost:9999 {
            fail_duration 10s
            status_path /admin/*

            health_check http://httpbin.org/anything {
                path /status/200
                interval 5s
                timeout 2s
                expected_status 200
            }

            health_check http://localhost:9999 {
                path /health
                interval 5s
                timeout 2s
                expected_status 200
            }
        }
    }
}
EOF

echo "Starting container with health checks..."
docker run --rm -d \
    --name caddy-status-test \
    -v $(pwd)/test-health.Caddyfile:/etc/caddy/Caddyfile \
    -p 8091:8080 \
    caddy-failover:status-test

echo "Waiting for health checks to run..."
sleep 8

echo "Checking status endpoint with health check data..."
response=$(curl -s http://localhost:8091/admin/failover/status)
echo "$response" | python3 -m json.tool || echo "$response"

docker stop caddy-status-test

echo ""
echo "Cleaning up test files..."
rm -f test-*.Caddyfile

echo ""
echo "=========================================="
echo "Status tracking tests completed!"
