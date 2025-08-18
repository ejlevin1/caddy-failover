#!/bin/bash

# Quick local Docker build script
# Usage: ./scripts/docker-build-local.sh [variant]
#   variant: "standard" (default) or "loaded"
#
# This script builds a local development image with timestamp tags

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the script's directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate a timestamp tag
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null | sed 's/[^a-zA-Z0-9-]/-/g' || echo "nobranch")
TAG="local-${BRANCH}-${TIMESTAMP}"

# Get variant
VARIANT="${1:-standard}"

echo -e "${BLUE}ℹ${NC} Building local development image..."
echo "  • Branch: $BRANCH"
echo "  • Tag: $TAG"
echo "  • Variant: $VARIANT"
echo ""

# Call the main build script
"$SCRIPT_DIR/build-docker-with-tag.sh" "$TAG" "$VARIANT"

# Also tag as 'local-dev' for easy reference
if [ "$VARIANT" = "loaded" ]; then
    DEV_TAG="caddy-failover:local-dev-loaded"
else
    DEV_TAG="caddy-failover:local-dev"
fi

echo -e "${BLUE}ℹ${NC} Tagging as $DEV_TAG for easy reference..."
docker tag "caddy-failover:$TAG" "$DEV_TAG"
echo -e "${GREEN}✓${NC} Tagged as $DEV_TAG"
echo ""
echo "Quick run commands:"
echo "  docker run --rm -p 8080:80 $DEV_TAG caddy version"
echo "  docker run --rm -p 8080:80 -v \$(pwd)/Caddyfile:/etc/caddy/Caddyfile $DEV_TAG"
