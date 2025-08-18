#!/bin/bash

# Simple Docker build wrapper
# Usage: ./scripts/docker-build.sh [tag]
#   tag: Optional tag name (defaults to "local")
#
# Examples:
#   ./scripts/docker-build.sh          # Builds caddy-failover:local
#   ./scripts/docker-build.sh v1.0.0   # Builds caddy-failover:v1.0.0
#   ./scripts/docker-build.sh dev      # Builds caddy-failover:dev

set -e

# Get the script's directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Use provided tag or default to "local"
TAG="${1:-local}"

# Call the main build script with standard variant
exec "$SCRIPT_DIR/build-docker-with-tag.sh" "$TAG" "standard"
