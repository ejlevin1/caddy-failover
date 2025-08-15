#!/bin/bash

# Setup script for git hooks
# This script installs pre-commit and cocogitto for conventional commits

set -e

echo "🚀 Setting up git hooks for caddy-failover..."
echo ""

# Check OS
OS="$(uname -s)"
ARCH="$(uname -m)"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install pre-commit
install_precommit() {
    echo "📦 Installing pre-commit..."

    if command_exists pre-commit; then
        echo "✅ pre-commit is already installed"
    else
        # Check if we're in a virtual environment
        if [ -n "$VIRTUAL_ENV" ]; then
            echo "   Detected virtual environment, installing without --user flag"
            if command_exists pip; then
                pip install pre-commit
            elif command_exists pip3; then
                pip3 install pre-commit
            else
                echo "❌ Could not install pre-commit in virtual environment."
                exit 1
            fi
        elif command_exists brew; then
            brew install pre-commit
        elif command_exists pip3; then
            pip3 install --user pre-commit
        elif command_exists pip; then
            pip install --user pre-commit
        else
            echo "❌ Could not install pre-commit. Please install Python pip or Homebrew first."
            echo "   Visit: https://pre-commit.com/#install"
            exit 1
        fi
        echo "✅ pre-commit installed successfully"
    fi
}

# Install cocogitto
install_cocogitto() {
    echo "📦 Installing cocogitto (conventional commits tool)..."

    if command_exists cog; then
        echo "✅ cocogitto is already installed"
    else
        # Use cargo if available
        if command_exists cargo; then
            echo "   Installing cocogitto via cargo..."
            cargo install cocogitto
            echo "✅ cocogitto installed via cargo"
            return
        fi

        # Otherwise try homebrew on macOS
        if [ "$OS" = "Darwin" ] && command_exists brew; then
            echo "   Installing cocogitto via homebrew..."
            brew install cocogitto
            echo "✅ cocogitto installed via homebrew"
            return
        fi

        # Fall back to downloading binary
        COG_VERSION="6.3.0"

        case "$OS" in
            Linux)
                case "$ARCH" in
                    x86_64)
                        COG_URL="https://github.com/cocogitto/cocogitto/releases/download/${COG_VERSION}/cocogitto-${COG_VERSION}-x86_64-unknown-linux-musl.tar.gz"
                        ;;
                    *)
                        echo "❌ Unsupported architecture: $ARCH for Linux"
                        echo "   Please install via cargo: cargo install cocogitto"
                        exit 1
                        ;;
                esac
                ;;
            Darwin)
                # Only x86_64 binary available for macOS
                COG_URL="https://github.com/cocogitto/cocogitto/releases/download/${COG_VERSION}/cocogitto-${COG_VERSION}-x86_64-apple-darwin.tar.gz"
                if [ "$ARCH" = "arm64" ]; then
                    echo "⚠️  Note: Installing x86_64 binary on ARM64 Mac (will run via Rosetta 2)"
                fi
                ;;
            *)
                echo "❌ Unsupported OS: $OS"
                echo "   Please install cocogitto manually: https://docs.cocogitto.io/installation"
                exit 1
                ;;
        esac

        # Download and install
        echo "   Downloading from: $COG_URL"
        TEMP_DIR=$(mktemp -d)
        curl -L "$COG_URL" -o "$TEMP_DIR/cocogitto.tar.gz"
        tar -xzf "$TEMP_DIR/cocogitto.tar.gz" -C "$TEMP_DIR"

        # Try to install to /usr/local/bin or ~/.local/bin
        if [ -w "/usr/local/bin" ]; then
            mv "$TEMP_DIR/cog" /usr/local/bin/
            echo "✅ cocogitto installed to /usr/local/bin/cog"
        else
            mkdir -p ~/.local/bin
            mv "$TEMP_DIR/cog" ~/.local/bin/
            echo "✅ cocogitto installed to ~/.local/bin/cog"
            echo "   Make sure ~/.local/bin is in your PATH"
        fi

        rm -rf "$TEMP_DIR"
    fi
}

# Install Go tools
install_go_tools() {
    echo "📦 Installing Go tools..."

    if ! command_exists go; then
        echo "❌ Go is not installed. Please install Go first: https://golang.org/dl/"
        exit 1
    fi

    # Install gosec if not present
    if ! command_exists gosec; then
        echo "   Installing gosec..."
        go install github.com/securego/gosec/v2/cmd/gosec@latest
    fi

    echo "✅ Go tools ready"
}

# Main installation
echo "1️⃣  Installing pre-commit framework..."
install_precommit

echo ""
echo "2️⃣  Installing cocogitto..."
install_cocogitto

echo ""
echo "3️⃣  Installing Go tools..."
install_go_tools

echo ""
echo "4️⃣  Installing pre-commit hooks..."
pre-commit install --install-hooks
pre-commit install --hook-type commit-msg

echo ""
echo "5️⃣  Installing cocogitto hooks..."
cog install-hook --all

echo ""
echo "6️⃣  Running initial checks..."
pre-commit run --all-files || true

echo ""
echo "✨ Setup complete!"
echo ""
echo "📝 Commit message format:"
echo "   <type>(<scope>): <subject>"
echo ""
echo "   Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
echo ""
echo "   Examples:"
echo "     feat: add new failover strategy"
echo "     fix(api): resolve connection timeout"
echo "     docs: update installation guide"
echo ""
echo "💡 Tips:"
echo "   - Use 'cog commit' for interactive commit creation"
echo "   - Use 'git commit' as normal (hooks will validate)"
echo "   - Run 'pre-commit run --all-files' to test all hooks"
echo ""
