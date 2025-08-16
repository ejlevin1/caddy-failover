#!/bin/bash

# Test runner script for caddy-failover plugin
# This script provides convenient commands for running different types of tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
VERBOSE=""
COVERAGE=""
RACE=""

# Function to print colored output
print_color() {
    color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to print usage
usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    unit        Run unit tests
    integration Run integration tests
    benchmark   Run benchmark tests
    all         Run all tests (unit + integration)
    coverage    Run tests with coverage report
    race        Run tests with race detector
    quick       Run quick tests (exclude integration)
    status      Test the failover status endpoint manually
    help        Show this help message

Options:
    -v, --verbose    Run with verbose output (shows detailed test output and status JSON)
    -c, --coverage   Generate coverage report
    -r, --race       Enable race detector
    -h, --help       Show this help message

Examples:
    $0 unit                    # Run unit tests
    $0 unit -v                 # Run unit tests with verbose output
    $0 integration             # Run integration tests
    $0 integration -v          # Run integration tests with status endpoint output
    $0 coverage                # Run all tests with coverage
    $0 race                    # Run all tests with race detector
    $0 benchmark              # Run benchmark tests
    $0 all -v -c              # Run all tests with verbose output and coverage

EOF
}

# Parse command line arguments
COMMAND=${1:-help}
shift || true

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -c|--coverage)
            COVERAGE="yes"
            shift
            ;;
        -r|--race)
            RACE="-race"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            print_color $RED "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Function to generate formatted coverage report
generate_coverage_report() {
    print_color $GREEN "\n========================================="
    print_color $GREEN "         CODE COVERAGE REPORT"
    print_color $GREEN "========================================="

    # Generate coverage statistics
    print_color $YELLOW "\n📊 Coverage Summary:"
    echo "-------------------"

    # Get total coverage percentage
    TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')

    # Color code the total coverage
    COVERAGE_VALUE=${TOTAL_COVERAGE%\%}
    # Remove decimal for integer comparison
    COVERAGE_INT=${COVERAGE_VALUE%.*}

    if [[ "$COVERAGE_INT" -ge 80 ]]; then
        print_color $GREEN "✅ Total Coverage: $TOTAL_COVERAGE"
    elif [[ "$COVERAGE_INT" -ge 60 ]]; then
        print_color $YELLOW "⚠️  Total Coverage: $TOTAL_COVERAGE"
    else
        print_color $RED "❌ Total Coverage: $TOTAL_COVERAGE"
    fi

    # Show per-package coverage
    print_color $YELLOW "\n📦 Package Coverage:"
    echo "-------------------"
    go list ./... | while read -r package; do
        # Get coverage for this package
        PACKAGE_COV=$(go tool cover -func=coverage.out | grep "^$package" | tail -1 | awk '{print $3}')
        if [[ ! -z "$PACKAGE_COV" ]]; then
            PACKAGE_NAME=$(basename $package)
            printf "%-30s %s\n" "$PACKAGE_NAME:" "$PACKAGE_COV"
        fi
    done

    # Show per-file coverage with color coding (excluding test helpers)
    print_color $YELLOW "\n📄 File Coverage Details:"
    echo "-------------------------"
    go tool cover -func=coverage.out | grep -v "total:" | grep -v "test_helpers.go" | while IFS=$'\t' read -r file func coverage; do
        if [[ ! -z "$coverage" && "$file" != "" ]]; then
            # Extract just the filename without path
            filename=$(basename "$file")
            cov_value=${coverage%\%}

            # Color code based on coverage
            # Remove decimal for integer comparison
            cov_int=${cov_value%.*}
            if [[ "$cov_int" -ge 80 ]]; then
                printf "  ${GREEN}%-40s %s${NC}\n" "$filename" "$coverage"
            elif [[ "$cov_int" -ge 60 ]]; then
                printf "  ${YELLOW}%-40s %s${NC}\n" "$filename" "$coverage"
            elif [[ "$cov_int" -gt 0 ]]; then
                printf "  ${RED}%-40s %s${NC}\n" "$filename" "$coverage"
            else
                printf "  ${RED}%-40s %s ⚠️${NC}\n" "$filename" "$coverage"
            fi
        fi
    done | sort -t: -k2 -rn | head -20  # Show top 20 files by coverage

    # Show uncovered functions (excluding test helpers)
    print_color $YELLOW "\n🔍 Functions with No Coverage:"
    echo "-------------------------------"
    go tool cover -func=coverage.out | grep "0.0%" | grep -v "test_helpers.go" | while read -r line; do
        file=$(echo "$line" | awk '{print $1}')
        func=$(echo "$line" | awk '{print $2}')
        filename=$(basename "$file")
        printf "  ❌ %-30s %s\n" "$filename:" "$func"
    done | head -10  # Show first 10 uncovered functions

    # Generate HTML report
    print_color $YELLOW "\n📊 Generating HTML Coverage Report..."
    go tool cover -html=coverage.out -o coverage.html

    # Try to open the HTML report in browser (if available)
    if command -v open &> /dev/null; then
        print_color $GREEN "✅ Opening coverage report in browser..."
        open coverage.html
    elif command -v xdg-open &> /dev/null; then
        print_color $GREEN "✅ Opening coverage report in browser..."
        xdg-open coverage.html
    else
        print_color $GREEN "✅ HTML coverage report saved to: coverage.html"
    fi

    # Generate coverage badge info
    print_color $YELLOW "\n🏷️  Coverage Badge Info:"
    echo "----------------------"
    if [[ "$COVERAGE_INT" -ge 80 ]]; then
        echo "Badge Color: Green (Excellent)"
    elif [[ "$COVERAGE_INT" -ge 60 ]]; then
        echo "Badge Color: Yellow (Good)"
    else
        echo "Badge Color: Red (Needs Improvement)"
    fi
    echo "Coverage: $TOTAL_COVERAGE"

    # Provide recommendations
    print_color $YELLOW "\n💡 Recommendations:"
    echo "-------------------"
    if [[ "$COVERAGE_INT" -lt 60 ]]; then
        echo "• Consider adding more unit tests to improve coverage"
        echo "• Focus on testing critical business logic first"
        echo "• Run 'go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out'"
    elif [[ "$COVERAGE_INT" -lt 80 ]]; then
        echo "• Good coverage! Consider targeting 80% or higher"
        echo "• Review uncovered functions for testing opportunities"
    else
        echo "• Excellent coverage! Keep maintaining high standards"
        echo "• Consider adding integration tests for complex scenarios"
    fi

    print_color $GREEN "\n========================================="
    print_color $GREEN "        END OF COVERAGE REPORT"
    print_color $GREEN "========================================="
}

# Function to run unit tests
run_unit_tests() {
    print_color $GREEN "=== Running Unit Tests ==="

    local cmd="go test $VERBOSE $RACE"

    if [[ "$COVERAGE" == "yes" ]]; then
        cmd="$cmd -coverprofile=coverage.out -covermode=atomic"
    fi

    # Run tests excluding integration tests
    cmd="$cmd -short ./..."

    print_color $YELLOW "Command: $cmd"
    eval $cmd

    if [[ "$COVERAGE" == "yes" ]]; then
        generate_coverage_report
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_color $GREEN "=== Running Integration Tests ==="

    # Check if xcaddy is installed
    if ! command -v xcaddy &> /dev/null; then
        print_color $YELLOW "xcaddy not found. Installing..."
        go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

        # Add GOPATH/bin to PATH if not already there
        export PATH=$PATH:$(go env GOPATH)/bin

        # Check again after installation
        if ! command -v xcaddy &> /dev/null; then
            print_color $RED "Failed to install xcaddy. Please install manually:"
            print_color $YELLOW "  go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest"
            print_color $YELLOW "  export PATH=\$PATH:\$(go env GOPATH)/bin"
            return 1
        fi
    fi

    # Build Caddy with the plugin
    print_color $YELLOW "Building Caddy with failover plugin..."
    xcaddy build --with github.com/ejlevin1/caddy-failover=.

    local cmd="go test $VERBOSE $RACE -run Integration ./..."

    print_color $YELLOW "Command: $cmd"
    eval $cmd

    # Show the status endpoint output if verbose mode is enabled
    if [[ "$VERBOSE" == "-v" ]]; then
        print_color $GREEN "\n=== Status Endpoint Output Demo ==="
        print_color $YELLOW "Showing actual JSON response from failover status endpoint..."
        go test -v -run TestDisplayStatusEndpoint ./... 2>&1 | grep -B 2 -A 100 "FAILOVER STATUS" || true
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    print_color $GREEN "=== Running Benchmark Tests ==="

    local cmd="go test -bench=. -benchmem -run=^$ $VERBOSE ./..."

    print_color $YELLOW "Command: $cmd"
    eval $cmd
}

# Function to run all tests
run_all_tests() {
    run_unit_tests
    echo ""
    run_integration_tests
    echo ""
    run_benchmark_tests
}

# Function to run quick tests (exclude slow/integration tests)
run_quick_tests() {
    print_color $GREEN "=== Running Quick Tests (excluding integration) ==="

    local cmd="go test $VERBOSE $RACE -short ./..."

    print_color $YELLOW "Command: $cmd"
    eval $cmd
}


# Function to test failover status endpoint
test_status_endpoint() {
    print_color $GREEN "=== Testing Failover Status Endpoint ==="

    # Check if test Caddyfile exists
    if [[ ! -f "testdata/basic.Caddyfile" ]]; then
        print_color $RED "Error: testdata/basic.Caddyfile not found"
        exit 1
    fi

    # Check if xcaddy is installed
    if ! command -v xcaddy &> /dev/null; then
        print_color $YELLOW "xcaddy not found. Installing..."
        go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

        # Add GOPATH/bin to PATH if not already there
        export PATH=$PATH:$(go env GOPATH)/bin

        # Check again after installation
        if ! command -v xcaddy &> /dev/null; then
            print_color $RED "Failed to install xcaddy. Please install manually:"
            print_color $YELLOW "  go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest"
            print_color $YELLOW "  export PATH=\$PATH:\$(go env GOPATH)/bin"
            return 1
        fi
    fi

    # Build Caddy with the plugin
    print_color $YELLOW "Building Caddy with failover plugin..."
    xcaddy build --with github.com/ejlevin1/caddy-failover=.

    # Start Caddy
    print_color $YELLOW "Starting Caddy..."
    ./caddy run --config testdata/basic.Caddyfile --adapter caddyfile &
    CADDY_PID=$!

    # Wait for Caddy to start
    sleep 2

    # Test the status endpoint
    print_color $YELLOW "Testing /admin/failover/status endpoint..."
    curl -s http://localhost:8080/admin/failover/status | jq .

    # Stop Caddy
    print_color $YELLOW "Stopping Caddy..."
    kill $CADDY_PID
    wait $CADDY_PID 2>/dev/null || true
}

# Function to check dependencies
check_dependencies() {
    local missing_deps=()

    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_color $RED "Error: Missing required dependencies: ${missing_deps[*]}"
        print_color $YELLOW "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Main execution
check_dependencies

case $COMMAND in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    benchmark)
        run_benchmark_tests
        ;;
    all)
        run_all_tests
        ;;
    coverage)
        COVERAGE="yes"
        print_color $GREEN "=== Running Tests with Coverage Analysis ==="
        run_unit_tests
        ;;
    race)
        RACE="-race"
        run_all_tests
        ;;
    quick)
        run_quick_tests
        ;;
    status)
        test_status_endpoint
        ;;
    help)
        usage
        ;;
    *)
        print_color $RED "Unknown command: $COMMAND"
        usage
        exit 1
        ;;
esac

print_color $GREEN "✅ Done!"
