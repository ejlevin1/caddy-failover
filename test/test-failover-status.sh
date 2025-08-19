#!/bin/bash

# Test script for failover status endpoint
# This script sets up a test Caddy server and verifies the status endpoint

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================="
echo "Failover Status Endpoint Test"
echo "========================================="
echo ""

# Check if Caddy is installed
if ! command -v caddy &> /dev/null; then
    echo -e "${RED}❌ Caddy is not installed${NC}"
    echo "Please install Caddy first: https://caddyserver.com/docs/install"
    exit 1
fi

# Build the plugin
echo -e "${BLUE}Building caddy-failover plugin...${NC}"
go build -v

# Create a test Caddyfile
echo -e "${BLUE}Creating test Caddyfile...${NC}"
cat > test-status.Caddyfile << 'EOF'
{
    order failover_proxy before reverse_proxy
    order failover_status before respond
    admin :2019
    debug
}

:9090 {
    # Status endpoint - this should return JSON with path info
    handle /admin/failover/status {
        failover_status
    }

    # Test path 1: /auth/*
    handle /auth/* {
        failover_proxy http://localhost:5051 https://httpbin.org/anything {
            fail_duration 3s

            health_check http://localhost:5051 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }
        }
    }

    # Test path 2: /admin/* (excluding /admin/failover/status)
    handle /admin/* {
        failover_proxy http://localhost:5041 http://localhost:5042 https://httpbin.org/anything {
            fail_duration 3s

            health_check http://localhost:5041 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }
        }
    }

    # Test path 3: /api/*
    handle /api/* {
        failover_proxy http://localhost:5021 https://httpbin.org/anything {
            fail_duration 3s

            health_check http://localhost:5021 {
                path /health
                interval 30s
                timeout 5s
                expected_status 200
            }
        }
    }

    # Test path 4: /gateway/*
    handle /gateway/* {
        failover_proxy http://localhost:5021 https://httpbin.org/anything {
            fail_duration 3s

            health_check http://localhost:5021 {
                path /gateway/health/ready
                interval 30s
                timeout 5s
                expected_status 200
            }

            health_check https://httpbin.org/anything {
                path /status/200
                interval 60s
                timeout 10s
                expected_status 200
            }
        }
    }

    # Root handler
    handle {
        respond "Test server running" 200
    }
}
EOF

# Start Caddy in the background
echo -e "${BLUE}Starting Caddy server...${NC}"
caddy start --config test-status.Caddyfile --adapter caddyfile

# Wait for Caddy to start
echo "Waiting for Caddy to start..."
sleep 3

# Function to test the status endpoint
test_status_endpoint() {
    echo ""
    echo -e "${BLUE}Testing failover status endpoint...${NC}"
    echo ""

    # Call the status endpoint
    RESPONSE=$(curl -s -X GET 'http://localhost:9090/admin/failover/status' \
        --header 'Accept: application/json' 2>/dev/null || echo "FAILED")

    if [ "$RESPONSE" = "FAILED" ]; then
        echo -e "${RED}❌ Failed to connect to status endpoint${NC}"
        return 1
    fi

    # Pretty print the response
    echo "Response from /admin/failover/status:"
    echo "----------------------------------------"
    echo "$RESPONSE" | python -m json.tool 2>/dev/null || echo "$RESPONSE"
    echo "----------------------------------------"
    echo ""

    # Validate the response
    echo -e "${BLUE}Validating response...${NC}"

    # Check if response is valid JSON
    if echo "$RESPONSE" | python -c "import sys, json; json.load(sys.stdin)" 2>/dev/null; then
        echo -e "${GREEN}✓ Valid JSON response${NC}"
    else
        echo -e "${RED}✗ Invalid JSON response${NC}"
        return 1
    fi

    # Check for expected paths
    PATHS_FOUND=0

    # Check for /auth/* path
    if echo "$RESPONSE" | grep -q '"/auth/\*"'; then
        echo -e "${GREEN}✓ Found /auth/* path${NC}"
        ((PATHS_FOUND++))
    else
        echo -e "${YELLOW}⚠ /auth/* path not found or has incorrect format${NC}"
    fi

    # Check for /admin/* path
    if echo "$RESPONSE" | grep -q '"/admin/\*"'; then
        echo -e "${GREEN}✓ Found /admin/* path${NC}"
        ((PATHS_FOUND++))
    else
        echo -e "${YELLOW}⚠ /admin/* path not found or has incorrect format${NC}"
    fi

    # Check for /api/* path
    if echo "$RESPONSE" | grep -q '"/api/\*"'; then
        echo -e "${GREEN}✓ Found /api/* path${NC}"
        ((PATHS_FOUND++))
    else
        echo -e "${YELLOW}⚠ /api/* path not found or has incorrect format${NC}"
    fi

    # Check for /gateway/* path
    if echo "$RESPONSE" | grep -q '"/gateway/\*"'; then
        echo -e "${GREEN}✓ Found /gateway/* path${NC}"
        ((PATHS_FOUND++))
    else
        echo -e "${YELLOW}⚠ /gateway/* path not found or has incorrect format${NC}"
    fi

    # Check for no "auto:" prefixes
    if echo "$RESPONSE" | grep -q '"auto:'; then
        echo -e "${RED}✗ Found 'auto:' prefix in paths (this should not happen)${NC}"
        echo "Problematic entries:"
        echo "$RESPONSE" | grep -o '"path":"[^"]*"' | grep "auto:"
        return 1
    else
        echo -e "${GREEN}✓ No 'auto:' prefixes found${NC}"
    fi

    # Check for duplicates by counting unique vs total paths
    TOTAL_PATHS=$(echo "$RESPONSE" | grep -o '"path":"[^"]*"' | wc -l | tr -d ' ')
    UNIQUE_PATHS=$(echo "$RESPONSE" | grep -o '"path":"[^"]*"' | sort -u | wc -l | tr -d ' ')

    if [ "$TOTAL_PATHS" = "$UNIQUE_PATHS" ]; then
        echo -e "${GREEN}✓ No duplicate paths found (${TOTAL_PATHS} paths total)${NC}"
    else
        echo -e "${RED}✗ Found duplicate paths: ${TOTAL_PATHS} total, ${UNIQUE_PATHS} unique${NC}"
        echo "Duplicate paths:"
        echo "$RESPONSE" | grep -o '"path":"[^"]*"' | sort | uniq -d
    fi

    echo ""
    if [ $PATHS_FOUND -ge 3 ]; then
        echo -e "${GREEN}✅ Status endpoint test PASSED${NC}"
        return 0
    else
        echo -e "${RED}❌ Status endpoint test FAILED - only found $PATHS_FOUND/4 expected paths${NC}"
        return 1
    fi
}

# Run the test
test_status_endpoint
TEST_RESULT=$?

# Test with specific host header (like in your example)
echo ""
echo -e "${BLUE}Testing with Host header...${NC}"
curl -s -X GET 'http://localhost:9090/admin/failover/status' \
    --header 'Accept: application/json' \
    --header 'Host: dev.portal.the-commons.app' | python -m json.tool 2>/dev/null | head -50

# Clean up
echo ""
echo -e "${BLUE}Stopping Caddy...${NC}"
caddy stop

# Remove test file
rm -f test-status.Caddyfile

echo ""
echo "========================================="
if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
else
    echo -e "${RED}Some tests failed. Please check the output above.${NC}"
fi
echo "========================================="

exit $TEST_RESULT
