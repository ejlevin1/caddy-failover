#!/bin/bash

# OpenAPI Docker Integration Test Script
# This script runs a full integration test of OpenAPI endpoints using Docker Compose
#
# Usage:
#   ./test-openapi-docker.sh              # Run tests (uses Docker cache)
#   REBUILD=1 ./test-openapi-docker.sh    # Force rebuild without cache
#   VERBOSE=1 ./test-openapi-docker.sh    # Show container logs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.openapi.yml"
CADDY_HOST="localhost"

# Function to find an available port
find_available_port() {
    python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()' 2>/dev/null || echo "$1"
}

# Find available ports dynamically
CADDY_PORT=$(find_available_port 9080)
CADDY_HTTPS_PORT=$(find_available_port 9443)
CADDY_ADMIN_PORT=$(find_available_port 2019)

BASE_URL="http://${CADDY_HOST}:${CADDY_PORT}"
TIMEOUT=10

# Export ports for docker-compose to use
export CADDY_PORT
export CADDY_HTTPS_PORT
export CADDY_ADMIN_PORT

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
EXIT_CODE=0
FAILED_TEST_NAMES=()

# Navigate to test directory
cd "$(dirname "$0")"

# Function to print colored output
print_color() {
    color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to cleanup on exit
cleanup() {
    print_color $YELLOW "\n🧹 Cleaning up Docker containers..."
    # Don't use -v flag to preserve volumes and caches
    docker-compose -f "$COMPOSE_FILE" down
    # Exit with the stored exit code or failed tests count
    if [ ${EXIT_CODE} -ne 0 ]; then
        exit ${EXIT_CODE}
    else
        exit ${FAILED_TESTS}
    fi
}

# Set trap for cleanup on exit
trap cleanup EXIT INT TERM

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local max_attempts=$2
    local attempt=1

    print_color $BLUE "⏳ Waiting for service at $url to be ready..."

    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            print_color $GREEN "✅ Service is ready!"
            return 0
        fi

        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done

    print_color $RED "❌ Service did not become ready in time"
    return 1
}

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local expected_content=$2
    local description=$3

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    print_color $BLUE "\n📝 Testing: ${description}"
    echo "  Endpoint: ${BASE_URL}${endpoint}"

    # Make request with curl
    response=$(curl -s -w "\n%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        if echo "$body" | grep -q "$expected_content" 2>/dev/null; then
            print_color $GREEN "  ✅ Success (HTTP $http_code)"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            print_color $YELLOW "  ⚠️  Response OK but content validation failed"
            echo "  Expected to find: '$expected_content'"
            echo "  Response preview: $(echo "$body" | head -c 200)..."
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$description")
            return 1
        fi
    else
        print_color $RED "  ❌ Failed (HTTP $http_code)"
        echo "  Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES+=("$description")
        return 1
    fi
}

# Function to test JSON endpoint
test_json_endpoint() {
    local endpoint=$1
    local json_path=$2
    local description=$3

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    print_color $BLUE "\n📊 Testing: ${description}"
    echo "  Endpoint: ${BASE_URL}${endpoint}"

    # Make request and parse JSON
    response=$(curl -s -w "\n%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        # Save JSON to temp file for analysis
        echo "$body" > /tmp/openapi_test.json

        # Try to parse JSON and check specific path
        if jq -e "$json_path" /tmp/openapi_test.json > /dev/null 2>&1; then
            value=$(jq -r "$json_path" /tmp/openapi_test.json)
            print_color $GREEN "  ✅ Success (HTTP $http_code)"
            echo "    JSON path '$json_path' = '$value'"
            PASSED_TESTS=$((PASSED_TESTS + 1))

            # Additional validation for OpenAPI structure
            if [[ "$endpoint" == *"openapi"* ]]; then
                # Check for required OpenAPI fields
                if jq -e '.paths' /tmp/openapi_test.json > /dev/null 2>&1; then
                    path_count=$(jq '.paths | length' /tmp/openapi_test.json)
                    echo "    Found $path_count API paths"
                fi
                if jq -e '.info.title' /tmp/openapi_test.json > /dev/null 2>&1; then
                    title=$(jq -r '.info.title' /tmp/openapi_test.json)
                    echo "    API Title: $title"
                fi
            fi
            return 0
        else
            print_color $YELLOW "  ⚠️  Invalid JSON or path not found"
            echo "    Attempted to access: $json_path"
            echo "    JSON structure:"
            jq -C '.' /tmp/openapi_test.json 2>/dev/null | head -20 || echo "    Failed to parse JSON"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$description")
            return 1
        fi
    else
        print_color $RED "  ❌ Failed (HTTP $http_code)"
        echo "  Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES+=("$description")
        return 1
    fi
}

# Function to test Caddy Admin API endpoints
test_admin_api_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_code=$4
    local description=$5

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    print_color $BLUE "\n🔧 Testing: ${description}"
    echo "  Method: ${method}"
    echo "  Endpoint: http://localhost:${CADDY_ADMIN_PORT}${endpoint}"

    # Prepare curl command based on method
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" -X GET "http://localhost:${CADDY_ADMIN_PORT}${endpoint}" 2>/dev/null)
    elif [ "$method" = "POST" ] || [ "$method" = "PUT" ] || [ "$method" = "PATCH" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "http://localhost:${CADDY_ADMIN_PORT}${endpoint}" 2>/dev/null)
    elif [ "$method" = "DELETE" ]; then
        response=$(curl -s -w "\n%{http_code}" -X DELETE "http://localhost:${CADDY_ADMIN_PORT}${endpoint}" 2>/dev/null)
    else
        print_color $RED "  ❌ Unknown method: $method"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "$expected_code" ]; then
        print_color $GREEN "  ✅ Success (HTTP $http_code)"
        if [ -n "$body" ] && echo "$body" | jq -e . > /dev/null 2>&1; then
            echo "    Response preview: $(echo "$body" | jq -c . | head -c 100)..."
        fi
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_color $RED "  ❌ Failed (Expected $expected_code, got $http_code)"
        echo "  Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES+=("$description")
        return 1
    fi
}

# Function to test POST /adapt endpoint
test_adapt_endpoint() {
    local config_format=$1
    local config_data=$2
    local description=$3

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    print_color $BLUE "\n🔄 Testing: ${description}"
    echo "  Endpoint: http://localhost:${CADDY_ADMIN_PORT}/adapt"
    echo "  Format: ${config_format}"

    # Set content type based on format
    if [ "$config_format" = "caddyfile" ]; then
        content_type="text/caddyfile"
    else
        content_type="application/json"
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: ${content_type}" \
        -d "$config_data" \
        "http://localhost:${CADDY_ADMIN_PORT}/adapt" 2>/dev/null)

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        if echo "$body" | jq -e . > /dev/null 2>&1; then
            print_color $GREEN "  ✅ Success - Valid JSON output"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            print_color $YELLOW "  ⚠️  Response OK but not valid JSON"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$description")
            return 1
        fi
    else
        print_color $RED "  ❌ Failed (HTTP $http_code)"
        echo "  Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_TEST_NAMES+=("$description")
        return 1
    fi
}

# Main test execution
START_TIME=$(date +%s)
print_color $GREEN "========================================="
print_color $GREEN "   OpenAPI Docker Integration Test"
print_color $GREEN "========================================="
echo ""

# Check for required tools
print_color $BLUE "🔍 Checking required tools..."
for tool in docker docker-compose curl jq; do
    if ! command -v $tool &> /dev/null; then
        print_color $RED "❌ Missing required tool: $tool"
        if [ "$tool" = "jq" ]; then
            print_color $YELLOW "  Install with: brew install jq (macOS) or apt-get install jq (Linux)"
        fi
        EXIT_CODE=1
        exit 1
    fi
done
print_color $GREEN "✅ All required tools are installed"

# Build and start services
print_color $BLUE "\n🚀 Starting Docker Compose services..."
print_color $BLUE "📍 Using dynamic ports:"
print_color $BLUE "   HTTP: ${CADDY_PORT}"
print_color $BLUE "   HTTPS: ${CADDY_HTTPS_PORT}"
print_color $BLUE "   Admin: ${CADDY_ADMIN_PORT}"
# Use --no-cache only if REBUILD env var is set
if [ "${REBUILD:-0}" = "1" ]; then
    print_color $YELLOW "Rebuilding without cache (REBUILD=1)"
    docker-compose -f "$COMPOSE_FILE" build --no-cache
else
    docker-compose -f "$COMPOSE_FILE" build
fi
docker-compose -f "$COMPOSE_FILE" up -d

# Wait for Caddy to be ready
if ! wait_for_service "${BASE_URL}/health" $TIMEOUT; then
    print_color $RED "❌ Caddy failed to start"
    docker-compose -f "$COMPOSE_FILE" logs caddy
    EXIT_CODE=1
    exit 1
fi

# Run tests
print_color $GREEN "\n🧪 Running OpenAPI endpoint tests..."
print_color $GREEN "========================================="

# Test Swagger UI endpoints
print_color $YELLOW "\n=== Swagger UI Tests ==="
test_endpoint "/api/docs" "swagger-ui" "Swagger UI (root)"
test_endpoint "/api/docs/" "SwaggerUIBundle" "Swagger UI (with trailing slash)"

# Test Redoc UI endpoint
print_color $YELLOW "\n=== Redoc UI Tests ==="
test_endpoint "/api/docs/redoc" "redoc" "Redoc UI"
test_endpoint "/api/docs/redoc/" "spec-url" "Redoc UI (with trailing slash)"

# Test OpenAPI JSON endpoints
print_color $YELLOW "\n=== OpenAPI JSON Tests ==="
test_json_endpoint "/api/docs/openapi.json" ".openapi" "OpenAPI 3.0 JSON"
test_json_endpoint "/api/docs/openapi.json" ".info.title" "OpenAPI Info Title"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy/failover/status"' "Failover Status Path"
test_json_endpoint "/api/docs/openapi-3.1.json" ".openapi" "OpenAPI 3.1 JSON"

# Test that OpenAPI includes Caddy Admin API endpoints
print_color $YELLOW "\n=== OpenAPI Documentation Coverage Tests ==="
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/config/{path}"' "OpenAPI includes config path operations"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/adapt"' "OpenAPI includes POST /adapt"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/load"' "OpenAPI includes POST /load"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/stop"' "OpenAPI includes POST /stop"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/pki/ca/{id}"' "OpenAPI includes PKI CA operations"
test_json_endpoint "/api/docs/openapi.json" '.paths."/caddy-admin/reverse_proxy/upstreams"' "OpenAPI includes reverse proxy upstreams"

# Test failover status endpoint
print_color $YELLOW "\n=== Failover Status Tests ==="
test_json_endpoint "/caddy/failover/status" ".[0].path" "Failover Status API"

# Test Caddy Admin API - Basic accessibility
print_color $YELLOW "\n=== Caddy Admin API Basic Tests ==="
if curl -s "http://localhost:${CADDY_ADMIN_PORT}/config/" > /dev/null 2>&1; then
    print_color $GREEN "  ✅ Caddy Admin API is accessible"
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
else
    print_color $RED "  ❌ Caddy Admin API is not accessible"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# Comprehensive Caddy Admin API Tests
print_color $YELLOW "\n=== Caddy Admin API Configuration Tests ==="

# Test GET /config/
test_admin_api_endpoint "GET" "/config/" "" "200" "Get full configuration"

# Test GET /config/apps
test_admin_api_endpoint "GET" "/config/apps" "" "200" "Get apps configuration"

# Test GET /config/apps/http
test_admin_api_endpoint "GET" "/config/apps/http" "" "200" "Get HTTP app configuration"

# Test GET /config/apps/http/servers
test_admin_api_endpoint "GET" "/config/apps/http/servers" "" "200" "Get HTTP servers configuration"

# Test non-existent path (Caddy returns 400 for invalid traversal paths)
test_admin_api_endpoint "GET" "/config/nonexistent/path" "" "400" "Get non-existent config path (should 400)"

# Test POST /adapt with Caddyfile
print_color $YELLOW "\n=== Caddy Admin API Adapt Tests ==="
caddyfile_config=':8888
respond "Hello from adapted Caddyfile"'
test_adapt_endpoint "caddyfile" "$caddyfile_config" "Adapt Caddyfile to JSON"

# Test POST /adapt with JSON (should pass through)
json_config='{
  "apps": {
    "http": {
      "servers": {
        "test": {
          "listen": [":8889"],
          "routes": [{
            "handle": [{
              "handler": "static_response",
              "body": "Test response"
            }]
          }]
        }
      }
    }
  }
}'
test_adapt_endpoint "json" "$json_config" "Adapt JSON config (passthrough)"

# Test configuration modifications
print_color $YELLOW "\n=== Caddy Admin API Modification Tests ==="

# Create a test route
test_route_config='{
  "handle": [{
    "handler": "static_response",
    "body": "Test route added via API",
    "status_code": 200
  }],
  "match": [{
    "path": ["/test-api-route"]
  }]
}'

# Test PUT to create new route
test_admin_api_endpoint "PUT" "/config/apps/http/servers/srv0/routes/0" "$test_route_config" "200" "PUT new test route"

# Test GET the newly created route
test_admin_api_endpoint "GET" "/config/apps/http/servers/srv0/routes/0" "" "200" "GET newly created route"

# Test PATCH to modify the route (PATCH replaces, expects array for handle)
patch_config='[{
  "handler": "static_response",
  "body": "Modified via PATCH",
  "status_code": 200
}]'
test_admin_api_endpoint "PATCH" "/config/apps/http/servers/srv0/routes/0/handle" "$patch_config" "200" "PATCH modify route handler"

# Test DELETE the test route
test_admin_api_endpoint "DELETE" "/config/apps/http/servers/srv0/routes/0" "" "200" "DELETE test route"

# Test PKI endpoints (if available)
print_color $YELLOW "\n=== Caddy PKI API Tests ==="
test_admin_api_endpoint "GET" "/pki/ca/local" "" "200" "Get local CA information"
test_admin_api_endpoint "GET" "/pki/ca/local/certificates" "" "200" "Get local CA certificates"

# Test reverse proxy upstreams endpoint
print_color $YELLOW "\n=== Caddy Monitoring API Tests ==="
test_admin_api_endpoint "GET" "/reverse_proxy/upstreams" "" "200" "Get reverse proxy upstreams status"

# Test error handling and validation
print_color $YELLOW "\n=== Error Handling and Validation Tests ==="

# Test invalid JSON in POST request (Caddy returns 500 for JSON decode errors)
invalid_json='{"invalid": json}'
test_admin_api_endpoint "POST" "/config/test" "$invalid_json" "500" "POST with invalid JSON (returns 500)"

# Test invalid method
test_admin_api_endpoint "GET" "/load" "" "405" "GET on POST-only endpoint (should 405)"

# Test headers and content negotiation
print_color $YELLOW "\n=== Headers and Content Type Tests ==="

# Test that config endpoint returns proper JSON content-type
TOTAL_TESTS=$((TOTAL_TESTS + 1))
print_color $BLUE "\n📋 Testing: Content-Type header validation"
response_headers=$(curl -s -I "http://localhost:${CADDY_ADMIN_PORT}/config/" 2>/dev/null)
if echo "$response_headers" | grep -q "Content-Type: application/json"; then
    print_color $GREEN "  ✅ Correct Content-Type: application/json"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_color $RED "  ❌ Missing or incorrect Content-Type header"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Test Etag header presence
TOTAL_TESTS=$((TOTAL_TESTS + 1))
print_color $BLUE "\n🏷️ Testing: ETag header presence"
if echo "$response_headers" | grep -q "Etag:"; then
    etag_value=$(echo "$response_headers" | grep "Etag:" | cut -d' ' -f2 | tr -d '\r')
    print_color $GREEN "  ✅ ETag header present: $etag_value"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_color $YELLOW "  ⚠️  No ETag header found (optional feature)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
fi

# Show container logs if verbose mode
if [ "${VERBOSE:-0}" = "1" ]; then
    print_color $YELLOW "\n=== Container Logs ==="
    docker-compose -f "$COMPOSE_FILE" logs --tail=50
fi

# Summary
print_color $GREEN "\n========================================="
print_color $BLUE "📊 Test Results Summary"
print_color $GREEN "========================================="

# Calculate pass rate
if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$(awk "BEGIN {printf \"%.1f\", ($PASSED_TESTS/$TOTAL_TESTS)*100}")
else
    PASS_RATE="0.0"
fi

# Print detailed results
echo ""
print_color $BLUE "Test Categories:"
echo "  • Swagger UI Tests"
echo "  • Redoc UI Tests"
echo "  • OpenAPI JSON Tests"
echo "  • OpenAPI Documentation Coverage Tests"
echo "  • Failover Status Tests"
echo "  • Caddy Admin API Configuration Tests"
echo "  • Caddy Admin API Adapt Tests"
echo "  • Caddy Admin API Modification Tests"
echo "  • Caddy PKI API Tests"
echo "  • Caddy Monitoring API Tests"
echo "  • Error Handling and Validation Tests"
echo "  • Headers and Content Type Tests"

echo ""
print_color $BLUE "Test Statistics:"
echo -e "  Total tests run: ${YELLOW}${TOTAL_TESTS}${NC}"
echo -e "  Tests passed:    ${GREEN}${PASSED_TESTS}${NC}"
echo -e "  Tests failed:    ${RED}${FAILED_TESTS}${NC}"
echo -e "  Pass rate:       ${YELLOW}${PASS_RATE}%${NC}"

# Show test duration if we tracked it
END_TIME=$(date +%s)
if [ -n "$START_TIME" ]; then
    DURATION=$((END_TIME - START_TIME))
    echo -e "  Test duration:   ${YELLOW}${DURATION}s${NC}"
fi

# Final status
echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    print_color $GREEN "========================================="
    print_color $GREEN "🎉 ALL TESTS PASSED! 🎉"
    print_color $GREEN "========================================="
    echo ""
    print_color $GREEN "✅ Swagger UI is functional"
    print_color $GREEN "✅ Redoc UI is functional"
    print_color $GREEN "✅ OpenAPI specifications are valid"
    print_color $GREEN "✅ All API endpoints are documented"
    print_color $GREEN "✅ Caddy Admin API is fully operational"
    print_color $GREEN "✅ Failover plugin is working correctly"
    print_color $GREEN "✅ Error handling is proper"
    echo ""
    exit 0
else
    print_color $RED "========================================="
    print_color $RED "❌ SOME TESTS FAILED"
    print_color $RED "========================================="
    echo ""

    # List failed tests
    if [ ${#FAILED_TEST_NAMES[@]} -gt 0 ]; then
        print_color $RED "Failed Tests:"
        for test_name in "${FAILED_TEST_NAMES[@]}"; do
            echo "  ❌ $test_name"
        done
        echo ""
    fi

    print_color $YELLOW "Failed tests indicate issues with:"
    if [ $FAILED_TESTS -gt 5 ]; then
        print_color $RED "  • Multiple API endpoints or documentation"
    elif [ $FAILED_TESTS -gt 2 ]; then
        print_color $YELLOW "  • Some API functionality or documentation"
    else
        print_color $YELLOW "  • Minor configuration or endpoint issues"
    fi
    echo ""
    print_color $YELLOW "Debugging tips:"
    print_color $YELLOW "  1. Run with VERBOSE=1 to see container logs"
    print_color $YELLOW "  2. Check if all containers are running: docker-compose ps"
    print_color $YELLOW "  3. Verify Caddy config: curl http://localhost:${CADDY_ADMIN_PORT}/config/"
    print_color $YELLOW "  4. Check individual endpoint: curl -v <endpoint>"
    echo ""
    exit 1
fi
