#!/bin/bash

# Test script for OpenAPI endpoints
# This script validates that all OpenAPI documentation endpoints work correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================="
echo "OpenAPI Endpoints Test"
echo "========================================="
echo ""

# Configuration
# For private-test example, the port is 9444
CADDY_PORT="${CADDY_PORT:-9444}"
CADDY_HOST="${CADDY_HOST:-localhost}"
BASE_URL="https://${CADDY_HOST}:${CADDY_PORT}"

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local expected_content=$2
    local description=$3

    echo -e "${BLUE}Testing: ${description}${NC}"
    echo "  Endpoint: ${endpoint}"

    # Make request with curl (ignore SSL for localhost)
    response=$(curl -s -k -w "\n%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        if echo "$body" | grep -q "$expected_content" 2>/dev/null; then
            echo -e "  ${GREEN}✓ Success${NC} (HTTP $http_code)"
            return 0
        else
            echo -e "  ${YELLOW}⚠ Response OK but content validation failed${NC}"
            echo "  Expected to find: '$expected_content'"
            return 1
        fi
    else
        echo -e "  ${RED}✗ Failed${NC} (HTTP $http_code)"
        return 1
    fi
}

# Function to test JSON endpoint
test_json_endpoint() {
    local endpoint=$1
    local json_path=$2
    local description=$3

    echo -e "${BLUE}Testing: ${description}${NC}"
    echo "  Endpoint: ${endpoint}"

    # Make request and parse JSON
    response=$(curl -s -k -w "\n%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        # Try to parse JSON and check specific path
        if echo "$body" | jq -e "$json_path" > /dev/null 2>&1; then
            value=$(echo "$body" | jq -r "$json_path")
            echo -e "  ${GREEN}✓ Success${NC} (HTTP $http_code)"
            echo "    JSON path '$json_path' = '$value'"
            return 0
        else
            echo -e "  ${YELLOW}⚠ Invalid JSON or path not found${NC}"
            return 1
        fi
    else
        echo -e "  ${RED}✗ Failed${NC} (HTTP $http_code)"
        return 1
    fi
}

# Check if jq is installed for JSON parsing
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}Warning: jq is not installed. JSON validation will be limited.${NC}"
    echo "Install jq for better JSON testing: brew install jq"
    echo ""
fi

# Check if Caddy is running
echo -e "${BLUE}Checking Caddy server...${NC}"
if curl -s -k "${BASE_URL}/caddy/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Caddy is running${NC}"
else
    echo -e "${RED}✗ Caddy is not running or not accessible at ${BASE_URL}${NC}"
    echo "Please start Caddy with the OpenAPI configuration first."
    exit 1
fi
echo ""

# Test counters
total_tests=0
passed_tests=0

# Test Swagger UI endpoints
echo -e "${YELLOW}=== Swagger UI Tests ===${NC}"
echo ""

test_endpoint "/caddy/openapi/" "swagger-ui" "Swagger UI"
[ $? -eq 0 ] && ((passed_tests++))
((total_tests++))

test_endpoint "/caddy/openapi/" "SwaggerUIBundle" "Swagger UI JavaScript loaded"
[ $? -eq 0 ] && ((passed_tests++))
((total_tests++))

echo ""

# Test Redoc UI endpoint
echo -e "${YELLOW}=== Redoc UI Tests ===${NC}"
echo ""

test_endpoint "/caddy/openapi/redoc" "redoc" "Redoc UI"
[ $? -eq 0 ] && ((passed_tests++))
((total_tests++))

test_endpoint "/caddy/openapi/redoc" "spec-url" "Redoc spec URL configured"
[ $? -eq 0 ] && ((passed_tests++))
((total_tests++))

echo ""

# Test OpenAPI JSON endpoints
echo -e "${YELLOW}=== OpenAPI JSON Tests ===${NC}"
echo ""

if command -v jq &> /dev/null; then
    test_json_endpoint "/caddy/openapi/openapi.json" ".openapi" "OpenAPI 3.0 JSON"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))

    test_json_endpoint "/caddy/openapi/openapi.json" ".info.title" "OpenAPI Info Title"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))

    test_json_endpoint "/caddy/openapi/openapi.json" ".paths" "OpenAPI Paths"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))

    test_json_endpoint "/caddy/openapi/openapi-3.1.json" ".openapi" "OpenAPI 3.1 JSON"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))
else
    test_endpoint "/caddy/openapi/openapi.json" "openapi" "OpenAPI 3.0 JSON (basic check)"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))

    test_endpoint "/caddy/openapi/openapi-3.1.json" "openapi" "OpenAPI 3.1 JSON (basic check)"
    [ $? -eq 0 ] && ((passed_tests++))
    ((total_tests++))
fi

echo ""

# Test failover status endpoint (should be documented in OpenAPI)
echo -e "${YELLOW}=== Failover Status Endpoint ===${NC}"
echo ""

test_endpoint "/caddy/failover/status" "[" "Failover Status API"
[ $? -eq 0 ] && ((passed_tests++))
((total_tests++))

echo ""

# Summary
echo "========================================="
echo -e "${BLUE}Test Summary${NC}"
echo "========================================="
echo -e "Total tests: ${total_tests}"
echo -e "Passed: ${GREEN}${passed_tests}${NC}"
echo -e "Failed: ${RED}$((total_tests - passed_tests))${NC}"

if [ $passed_tests -eq $total_tests ]; then
    echo ""
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
