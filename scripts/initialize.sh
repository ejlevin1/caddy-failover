#!/bin/bash

# Development Environment Initialization Script for caddy-failover
# This script checks for required tools and provides installation instructions
#
# Supported Systems: macOS, Linux (Ubuntu/Debian, RHEL/CentOS/Fedora, Arch)
# Note: Windows/WSL is not tested and may require adjustments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Track if any tools are missing
MISSING_TOOLS=0
WARNINGS=0

# Detect OS and package manager
OS_TYPE="unknown"
PKG_MANAGER="unknown"
PKG_INSTALL_CMD="unknown"

detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        OS_TYPE="macos"
        if command -v brew &> /dev/null; then
            PKG_MANAGER="brew"
            PKG_INSTALL_CMD="brew install"
        else
            PKG_MANAGER="none"
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS_TYPE="linux"
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            if [[ "$ID" == "ubuntu" ]] || [[ "$ID" == "debian" ]]; then
                PKG_MANAGER="apt"
                PKG_INSTALL_CMD="sudo apt-get install -y"
            elif [[ "$ID" == "fedora" ]] || [[ "$ID" == "centos" ]] || [[ "$ID" == "rhel" ]]; then
                PKG_MANAGER="dnf"
                PKG_INSTALL_CMD="sudo dnf install -y"
            elif [[ "$ID" == "arch" ]] || [[ "$ID" == "manjaro" ]]; then
                PKG_MANAGER="pacman"
                PKG_INSTALL_CMD="sudo pacman -S --noconfirm"
            elif [[ "$ID" == "opensuse"* ]]; then
                PKG_MANAGER="zypper"
                PKG_INSTALL_CMD="sudo zypper install -y"
            elif [[ "$ID" == "alpine" ]]; then
                PKG_MANAGER="apk"
                PKG_INSTALL_CMD="sudo apk add"
            fi
        fi
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "win32" ]]; then
        OS_TYPE="windows"
        print_color $YELLOW "⚠️  Windows detected. This script is not tested on Windows/WSL."
        print_color $YELLOW "   You may need to adjust commands for your environment."
        print_color $YELLOW "   Consider using WSL2 with Ubuntu for best compatibility."
        echo ""
    elif [[ "$OSTYPE" == "freebsd"* ]]; then
        OS_TYPE="freebsd"
        PKG_MANAGER="pkg"
        PKG_INSTALL_CMD="sudo pkg install -y"
    fi
}

# Function to print colored output
print_color() {
    color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to get OS-specific install command
get_install_cmd() {
    local tool=$1
    local brew_pkg=${2:-$tool}
    local apt_pkg=${3:-$tool}
    local dnf_pkg=${4:-$tool}
    local pacman_pkg=${5:-$tool}

    case "$PKG_MANAGER" in
        brew)
            echo "brew install $brew_pkg"
            ;;
        apt)
            echo "sudo apt-get update && sudo apt-get install -y $apt_pkg"
            ;;
        dnf)
            echo "sudo dnf install -y $dnf_pkg"
            ;;
        pacman)
            echo "sudo pacman -S --noconfirm $pacman_pkg"
            ;;
        zypper)
            echo "sudo zypper install -y $tool"
            ;;
        apk)
            echo "sudo apk add $tool"
            ;;
        pkg)
            echo "sudo pkg install -y $tool"
            ;;
        none)
            if [ "$OS_TYPE" = "macos" ]; then
                echo "Install Homebrew first: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            else
                echo "Manual installation required - check package manager documentation"
            fi
            ;;
        *)
            echo "Manual installation required for your system"
            ;;
    esac
}

# Function to check if a command exists
check_command() {
    local cmd=$1
    local name=$2
    local brew_pkg=${3:-$cmd}
    local apt_pkg=${4:-$cmd}
    local dnf_pkg=${5:-$cmd}
    local pacman_pkg=${6:-$cmd}
    local required=${7:-true}

    # Check if command exists in PATH
    local cmd_path=""
    if command -v "$cmd" &> /dev/null; then
        cmd_path=$(command -v "$cmd")
    elif [ -n "$GOPATH" ] && [ -x "$GOPATH/bin/$cmd" ]; then
        # Check in GOPATH/bin for Go tools
        cmd_path="$GOPATH/bin/$cmd"
    elif [ -x "$(go env GOPATH 2>/dev/null)/bin/$cmd" ]; then
        # Check in default GOPATH/bin
        cmd_path="$(go env GOPATH)/bin/$cmd"
    fi

    if [ -n "$cmd_path" ]; then
        local version=""
        # Special handling for different version commands
        case "$cmd" in
            go)
                version=$(go version 2>&1 | head -n1 || echo "version unknown")
                ;;
            git)
                version=$(git --version 2>&1 | head -n1 || echo "version unknown")
                ;;
            docker)
                version=$(docker --version 2>&1 | head -n1 || echo "version unknown")
                ;;
            python3|python)
                version=$($cmd --version 2>&1 | head -n1 || echo "version unknown")
                ;;
            gofmt)
                # gofmt doesn't have a version flag, it's part of Go
                version="included with Go $(go version | awk '{print $3}')"
                ;;
            xcaddy)
                # Use the full path if not in PATH
                version=$("$cmd_path" version 2>&1 | head -n1 || echo "version unknown")
                ;;
            *)
                # Try --version first, then -v, then version
                # Use the full path if command is not in PATH
                local cmd_to_run="$cmd"
                if [ -n "$cmd_path" ] && ! command -v "$cmd" &> /dev/null; then
                    cmd_to_run="$cmd_path"
                fi
                version=$($cmd_to_run --version 2>&1 | head -n1 || $cmd_to_run -v 2>&1 | head -n1 || $cmd_to_run version 2>&1 | head -n1 || echo "version unknown")
                ;;
        esac
        print_color $GREEN "✅ $name found: $version"
        # If it's a Go tool found in GOPATH but not in PATH, just inform the user
        if [[ "$cmd_path" == *"GOPATH"* ]] || [[ "$cmd_path" == */go/bin/* ]]; then
            if ! echo "$PATH" | grep -q "$(dirname "$cmd_path")"; then
                print_color $CYAN "   ℹ️  Located in $(dirname "$cmd_path")"
                print_color $CYAN "      To add to PATH: export PATH=\$PATH:$(dirname "$cmd_path")"
            fi
        fi
        return 0
    else
        if [ "$required" = true ]; then
            print_color $RED "❌ $name NOT found (REQUIRED)"
            MISSING_TOOLS=$((MISSING_TOOLS + 1))
        else
            print_color $YELLOW "⚠️  $name NOT found (OPTIONAL)"
            WARNINGS=$((WARNINGS + 1))
        fi
        local install_cmd=$(get_install_cmd "$cmd" "$brew_pkg" "$apt_pkg" "$dnf_pkg" "$pacman_pkg")
        print_color $CYAN "   Installation: $install_cmd"
        return 1
    fi
}

# Function to check Go version
check_go_version() {
    if command -v go &> /dev/null; then
        local go_version=$(go version | awk '{print $3}' | sed 's/go//')
        local required_version="1.22"

        # Compare versions (simple comparison, works for most cases)
        if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" = "$required_version" ]; then
            print_color $GREEN "✅ Go version $go_version (meets minimum $required_version)"
        else
            print_color $YELLOW "⚠️  Go version $go_version (minimum required: $required_version)"
            print_color $CYAN "   Update Go: https://go.dev/doc/install"
            WARNINGS=$((WARNINGS + 1))
        fi
    fi
}

# Function to check Docker
check_docker() {
    if command -v docker &> /dev/null; then
        if docker info &> /dev/null; then
            local version=$(docker --version)
            print_color $GREEN "✅ Docker found and running: $version"
        else
            print_color $YELLOW "⚠️  Docker found but daemon not running"
            print_color $CYAN "   Start Docker Desktop or run: sudo systemctl start docker"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        print_color $YELLOW "⚠️  Docker NOT found (RECOMMENDED for integration tests)"
        if [ "$OS_TYPE" = "macos" ]; then
            print_color $CYAN "   Installation: https://docs.docker.com/desktop/install/mac-install/"
        elif [ "$OS_TYPE" = "linux" ]; then
            print_color $CYAN "   Installation: https://docs.docker.com/engine/install/"
        else
            print_color $CYAN "   Installation: https://docs.docker.com/get-docker/"
        fi
        WARNINGS=$((WARNINGS + 1))
    fi
}

# Detect OS before anything else
detect_os

# Main header
print_color $BLUE "========================================="
print_color $BLUE "  Caddy Failover Development Setup Check"
print_color $BLUE "========================================="
echo ""

# System Information
print_color $YELLOW "🖥️  System Information:"
print_color $YELLOW "----------------------"
print_color $CYAN "OS Type: $OS_TYPE"
print_color $CYAN "Package Manager: $PKG_MANAGER"
if [ "$OS_TYPE" = "linux" ] && [ -f /etc/os-release ]; then
    . /etc/os-release
    print_color $CYAN "Distribution: $NAME $VERSION"
fi
echo ""

# Core Requirements
print_color $YELLOW "📦 Core Requirements:"
print_color $YELLOW "--------------------"
# Go has different package names across systems
if [ "$OS_TYPE" = "linux" ]; then
    check_command "go" "Go" "go" "golang" "golang" "go" true
else
    check_command "go" "Go" "go" "golang" "golang" "go" true
fi
check_go_version
print_color $CYAN "   📝 Used for: Building and testing the Caddy plugin"
echo ""
check_command "git" "Git" "git" "git" "git" "git" true
print_color $CYAN "   📝 Used for: Version control and collaboration"
echo ""

# Build Tools
print_color $YELLOW "🔨 Build Tools:"
print_color $YELLOW "---------------"
# xcaddy is typically installed via go install
check_command "xcaddy" "xcaddy" "xcaddy" "xcaddy" "xcaddy" "xcaddy" false
print_color $CYAN "   📝 Used for: Building Caddy server with custom plugins"
echo ""
check_command "make" "Make" "make" "make" "make" "make" false
print_color $CYAN "   📝 Used for: Build automation and task running"
echo ""

# Container & Testing Tools
print_color $YELLOW "🧪 Container & Testing Tools:"
print_color $YELLOW "-----------------------------"
check_command "curl" "curl" "curl" "curl" "curl" "curl" true
print_color $CYAN "   📝 Used for: Testing HTTP endpoints and downloading files"
echo ""
check_docker
print_color $CYAN "   📝 Used for: Running containerized tests and building Docker images"
echo ""
# Docker Compose can be standalone or part of Docker Desktop
if command -v docker &> /dev/null && docker compose version &> /dev/null 2>&1; then
    version=$(docker compose version 2>&1 | head -n1)
    print_color $GREEN "✅ Docker Compose found (Docker plugin): $version"
    print_color $CYAN "   📝 Used for: Orchestrating multi-container test environments"
elif command -v docker-compose &> /dev/null; then
    version=$(docker-compose --version 2>&1 | head -n1)
    print_color $GREEN "✅ Docker Compose found (standalone): $version"
    print_color $CYAN "   📝 Used for: Orchestrating multi-container test environments"
else
    print_color $YELLOW "⚠️  Docker Compose NOT found (RECOMMENDED for integration tests)"
    print_color $CYAN "   Installation: Included with Docker Desktop or install docker-compose-plugin"
    print_color $CYAN "   📝 Used for: Orchestrating multi-container test environments"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""
check_command "jq" "jq (JSON processor)" "jq" "jq" "jq" "jq" false
print_color $CYAN "   📝 Used for: Parsing JSON output in test scripts"
echo ""

# Code Quality Tools
print_color $YELLOW "✨ Code Quality Tools:"
print_color $YELLOW "----------------------"
check_command "gofmt" "gofmt" "go" "golang" "golang" "go" true
print_color $CYAN "   📝 Used for: Formatting Go code to standard style"
echo ""
check_command "golangci-lint" "golangci-lint" "golangci-lint" "golangci-lint" "golangci-lint" "golangci-lint" false
if ! command -v golangci-lint &> /dev/null && [ -z "$(find $(go env GOPATH 2>/dev/null)/bin -name golangci-lint 2>/dev/null)" ]; then
    print_color $CYAN "   Alternative: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
fi
print_color $CYAN "   📝 Used for: Comprehensive Go code linting and static analysis"
echo ""
check_command "yamllint" "yamllint" "yamllint" "yamllint" "yamllint" "yamllint" false
if ! command -v yamllint &> /dev/null; then
    print_color $CYAN "   Alternative: pip3 install --user yamllint"
fi
print_color $CYAN "   📝 Used for: Validating YAML files (configs, workflows)"
echo ""

# Git Hooks & CI Tools
print_color $YELLOW "🔗 Git Hooks & CI Tools:"
print_color $YELLOW "------------------------"
check_command "python3" "Python 3" "python3" "python3" "python3" "python" false
print_color $CYAN "   📝 Used for: Running pre-commit hooks and various scripts"
echo ""
check_command "pip3" "pip3" "python3" "python3-pip" "python3-pip" "python-pip" false
print_color $CYAN "   📝 Used for: Installing Python packages (pre-commit, yamllint)"
echo ""
# Check for pre-commit
if command -v pre-commit &> /dev/null; then
    version=$(pre-commit --version 2>&1 | head -n1)
    print_color $GREEN "✅ pre-commit found: $version"
    print_color $CYAN "   📝 Used for: Automated code checks before git commits"
else
    print_color $YELLOW "⚠️  pre-commit NOT found (OPTIONAL - for Git hooks)"
    if command -v pip3 &> /dev/null; then
        print_color $CYAN "   Installation: pip3 install --user pre-commit"
    elif command -v pip &> /dev/null; then
        print_color $CYAN "   Installation: pip install --user pre-commit"
    else
        print_color $CYAN "   Installation: Install Python first, then: pip install pre-commit"
    fi
    print_color $CYAN "   📝 Used for: Automated code checks before git commits"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Additional Development Tools
print_color $YELLOW "🔧 Additional Tools:"
print_color $YELLOW "--------------------"
check_command "gh" "GitHub CLI" "gh" "gh" "gh" "github-cli" false
print_color $CYAN "   📝 Used for: Creating pull requests and managing GitHub repos"
echo ""
# Check for other useful tools
check_command "watch" "watch (file monitoring)" "watch" "procps-ng" "procps-ng" "procps" false
print_color $CYAN "   📝 Used for: Monitoring file changes and command output"
echo ""
check_command "tree" "tree (directory viewer)" "tree" "tree" "tree" "tree" false
print_color $CYAN "   📝 Used for: Visualizing directory structures"
echo ""

# Check for Go modules
print_color $YELLOW "📦 Go Modules Check:"
print_color $YELLOW "--------------------"
if [ -f "go.mod" ]; then
    print_color $GREEN "✅ go.mod found"

    # Check if modules are downloaded
    if go mod verify &> /dev/null; then
        print_color $GREEN "✅ Go modules verified"
    else
        print_color $YELLOW "⚠️  Go modules need updating"
        print_color $CYAN "   Run: go mod download && go mod verify"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    print_color $RED "❌ go.mod not found"
    print_color $CYAN "   Ensure you're in the project root directory"
    MISSING_TOOLS=$((MISSING_TOOLS + 1))
fi

# Check for pre-commit hooks configuration
if [ -f ".pre-commit-config.yaml" ]; then
    print_color $GREEN "✅ pre-commit config found"

    # Check if hooks are installed
    if [ -f ".git/hooks/pre-commit" ] && grep -q "pre-commit" ".git/hooks/pre-commit" 2>/dev/null; then
        print_color $GREEN "✅ pre-commit hooks installed"
    else
        print_color $YELLOW "⚠️  pre-commit hooks not installed"
        if command -v pre-commit &> /dev/null; then
            print_color $CYAN "   Run: pre-commit install"
        else
            print_color $CYAN "   Install pre-commit first, then run: pre-commit install"
        fi
        WARNINGS=$((WARNINGS + 1))
    fi
fi
echo ""

# Install missing Go tools
print_color $YELLOW "🔄 Installing/Updating Go Tools:"
print_color $YELLOW "---------------------------------"

# Install xcaddy if missing
if ! command -v xcaddy &> /dev/null; then
    print_color $CYAN "Installing xcaddy..."
    if go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest; then
        print_color $GREEN "✅ xcaddy installed successfully"
        # Add GOPATH/bin to PATH reminder
        print_color $YELLOW "   Note: Ensure $(go env GOPATH)/bin is in your PATH"
    else
        print_color $RED "❌ Failed to install xcaddy"
    fi
fi

# Install golangci-lint if missing and Go is available
if ! command -v golangci-lint &> /dev/null && command -v go &> /dev/null; then
    print_color $CYAN "Installing golangci-lint..."
    if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin; then
        print_color $GREEN "✅ golangci-lint installed successfully"
    else
        print_color $YELLOW "⚠️  Could not auto-install golangci-lint, use: brew install golangci-lint"
    fi
fi
echo ""

# Environment checks
print_color $YELLOW "🌍 Environment Checks:"
print_color $YELLOW "----------------------"

# Check GOPATH
if [ -z "$GOPATH" ]; then
    GOPATH=$(go env GOPATH 2>/dev/null || echo "")
fi

if [ -n "$GOPATH" ]; then
    print_color $GREEN "✅ GOPATH set: $GOPATH"

    # Check if GOPATH/bin is in PATH
    if echo "$PATH" | grep -q "$GOPATH/bin"; then
        print_color $GREEN "✅ $GOPATH/bin is in PATH"
    else
        print_color $YELLOW "⚠️  $GOPATH/bin is not in PATH"
        print_color $CYAN "   Add to your shell profile: export PATH=\$PATH:$GOPATH/bin"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    print_color $YELLOW "⚠️  GOPATH not set (using default)"
fi
echo ""

# Quick Actions
if [ $MISSING_TOOLS -gt 0 ] || [ $WARNINGS -gt 0 ]; then
    print_color $YELLOW "🚀 Quick Setup Commands:"
    print_color $YELLOW "------------------------"

    case "$OS_TYPE" in
        macos)
            if [ "$PKG_MANAGER" = "none" ]; then
                print_color $CYAN "# Install Homebrew first:"
                print_color $CYAN '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
                echo ""
            fi
            print_color $CYAN "# Install all recommended tools:"
            print_color $CYAN "brew install go git jq yamllint gh golangci-lint curl make"
            ;;
        linux)
            case "$PKG_MANAGER" in
                apt)
                    print_color $CYAN "# Update package list and install tools:"
                    print_color $CYAN "sudo apt-get update"
                    print_color $CYAN "sudo apt-get install -y golang git curl jq make yamllint"
                    print_color $CYAN "# For GitHub CLI:"
                    print_color $CYAN "curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo tee /usr/share/keyrings/githubcli-archive-keyring.gpg > /dev/null"
                    print_color $CYAN "echo \"deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null"
                    print_color $CYAN "sudo apt update && sudo apt install gh"
                    ;;
                dnf)
                    print_color $CYAN "# Install tools:"
                    print_color $CYAN "sudo dnf install -y golang git curl jq make yamllint gh"
                    ;;
                pacman)
                    print_color $CYAN "# Install tools:"
                    print_color $CYAN "sudo pacman -S --noconfirm go git curl jq make yamllint github-cli"
                    ;;
                *)
                    print_color $CYAN "# Install tools using your package manager"
                    print_color $CYAN "# Required: go, git, curl"
                    print_color $CYAN "# Optional: jq, make, yamllint, gh"
                    ;;
            esac
            ;;
        freebsd)
            print_color $CYAN "# Install tools:"
            print_color $CYAN "sudo pkg install -y go git curl jq gmake yamllint gh"
            ;;
        *)
            print_color $YELLOW "# Manual installation required for your system"
            print_color $YELLOW "# Visit https://go.dev/doc/install for Go installation"
            ;;
    esac
    echo ""

    print_color $CYAN "# Install Go tools (all systems):"
    print_color $CYAN "go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest"
    print_color $CYAN "# Install golangci-lint:"
    print_color $CYAN "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
    echo ""

    print_color $CYAN "# Download Go dependencies:"
    print_color $CYAN "go mod download && go mod verify"
    echo ""

    print_color $CYAN "# Add Go binaries to PATH:"
    if [ "$OS_TYPE" = "macos" ]; then
        print_color $CYAN "# Add to ~/.zshrc or ~/.bash_profile:"
    else
        print_color $CYAN "# Add to ~/.bashrc or ~/.profile:"
    fi
    print_color $CYAN 'export PATH=$PATH:$(go env GOPATH)/bin'
    echo ""
fi

# Summary
print_color $BLUE "========================================="
print_color $BLUE "              Summary"
print_color $BLUE "========================================="

if [ $MISSING_TOOLS -eq 0 ]; then
    print_color $GREEN "✅ All required tools are installed!"
else
    print_color $RED "❌ $MISSING_TOOLS required tool(s) missing"
fi

if [ $WARNINGS -gt 0 ]; then
    print_color $YELLOW "⚠️  $WARNINGS optional tool(s) missing or need attention"
fi

if [ $MISSING_TOOLS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    print_color $GREEN "🎉 Your development environment is fully configured!"
    echo ""
    print_color $CYAN "Next steps:"
    print_color $CYAN "  1. Run tests: ./scripts/test.sh all"
    print_color $CYAN "  2. Build Caddy: xcaddy build --with github.com/ejlevin1/caddy-failover=."
    print_color $CYAN "  3. Run integration tests: ./test/test-openapi-docker.sh"
else
    echo ""
    print_color $YELLOW "Please install missing tools and re-run this script to verify."
fi

echo ""
exit $MISSING_TOOLS
