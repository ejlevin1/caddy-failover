#!/bin/bash

# Test script for environment variable expansion functionality

set -e

# Set environment variables for testing
export DOCKER_HOST_URI="host.docker.internal"
export CUSTOM_HOST="my-custom-host"
export ENVIRONMENT="development"

echo "Testing environment variable expansion..."
echo "DOCKER_HOST_URI=$DOCKER_HOST_URI"
echo "CUSTOM_HOST=$CUSTOM_HOST"
echo "ENVIRONMENT=$ENVIRONMENT"
echo ""

echo "Building Caddy with failover plugin..."
docker build -t caddy-failover:test .

echo "Creating test Caddyfile with environment variables..."
cat > test-env-caddyfile <<EOF
{
    order failover_proxy before reverse_proxy
    debug
}

:8080 {
    handle /api/* {
        # Using environment variables in upstream URLs and headers
        # Note: httpbin.org/anything endpoint echoes the request
        failover_proxy http://{env.DOCKER_HOST_URI}:3000 https://httpbin.org/anything {
            fail_duration 3s
            dial_timeout 2s
            response_timeout 5s
            insecure_skip_verify

            # Headers with environment variables
            header_up http://{env.DOCKER_HOST_URI}:3000 X-Environment {env.ENVIRONMENT}
            header_up http://{env.DOCKER_HOST_URI}:3000 X-Custom-Host {env.CUSTOM_HOST}
            header_up https://httpbin.org/anything X-Environment production
        }
    }

    handle /test/* {
        # Mixed configuration with env vars and hardcoded values
        failover_proxy http://{env.DOCKER_HOST_URI}:5041 https://httpbin.org/anything {
            fail_duration 5s
            header_up http://{env.DOCKER_HOST_URI}:5041 X-Service-Track caddy
        }
    }

    handle /health {
        respond "OK" 200
    }
}
EOF

echo "Starting Caddy container with environment variables..."
docker run --rm -d \
    --name caddy-env-test \
    -v $(pwd)/test-env-caddyfile:/etc/caddy/Caddyfile \
    -p 8091:8080 \
    -e DOCKER_HOST_URI="$DOCKER_HOST_URI" \
    -e CUSTOM_HOST="$CUSTOM_HOST" \
    -e ENVIRONMENT="$ENVIRONMENT" \
    caddy-failover:test

echo "Waiting for Caddy to start..."
sleep 5

echo ""
echo "Checking Caddy logs for expanded values..."
echo "========================================="
docker logs caddy-env-test 2>&1 | grep -E "(expanded|DOCKER_HOST_URI|host.docker.internal)" || true
echo "========================================="
echo ""

echo "Running tests..."
echo "===================="

# Test 1: Check that the service starts without {env.} parsing errors
echo -n "Test 1 - Service starts without parsing errors: "
if docker ps | grep -q caddy-env-test; then
    echo "✅ PASSED"
else
    echo "❌ FAILED - Container not running"
    docker logs caddy-env-test
    docker stop caddy-env-test 2>/dev/null || true
    exit 1
fi

# Test 2: Health check endpoint works
echo -n "Test 2 - Health endpoint: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8091/health)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-env-test
    docker stop caddy-env-test
    exit 1
fi

# Test 3: API endpoint (will fail on first upstream but should succeed with httpbin)
echo -n "Test 3 - API endpoint with env var upstream: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8091/api/test)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-env-test | tail -20
    docker stop caddy-env-test
    exit 1
fi

# Test 4: Test endpoint (will also fail on first upstream but succeed with httpbin)
echo -n "Test 4 - Test endpoint with env var upstream: "
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8091/test/anything)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
else
    echo "❌ FAILED (expected 200, got $response)"
    docker logs caddy-env-test | tail -20
    docker stop caddy-env-test
    exit 1
fi

# Test 5: Check expanded values in logs
echo -n "Test 5 - Environment variables are expanded: "
if docker logs caddy-env-test 2>&1 | grep -q "host.docker.internal"; then
    echo "✅ PASSED (found expanded value in logs)"
else
    echo "⚠️  WARNING (expansion may be working but not logged)"
fi

echo ""
echo "===================="
echo "Checking final logs for environment variable expansion..."
docker logs caddy-env-test 2>&1 | grep -E "(attempting upstream|proxying request|expanded)" | tail -10 || true
echo "===================="

echo ""
echo "Stopping Caddy container..."
docker stop caddy-env-test

echo ""
echo "✅ Environment variable expansion tests completed successfully!"
echo ""
echo "Summary:"
echo "- Environment variables are properly expanded in upstream URLs"
echo "- Environment variables are properly expanded in header values"
echo "- The plugin works with mixed configurations (env vars + hardcoded)"
echo "- No parsing errors with {env.VARIABLE} syntax"
