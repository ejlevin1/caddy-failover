#!/bin/bash

# Build Docker image with custom tag
# Usage: ./scripts/build-docker-with-tag.sh "tag-name" [variant]
#   tag-name: The tag to apply to the image (required)
#   variant: "standard" (default) or "loaded" (with additional plugins)
#
# Examples:
#   ./scripts/build-docker-with-tag.sh "v1.0.0"
#   ./scripts/build-docker-with-tag.sh "dev-build"
#   ./scripts/build-docker-with-tag.sh "v1.0.0" loaded
#   ./scripts/build-docker-with-tag.sh "latest" standard

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ ${NC}$1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check for help flag
if [ "$1" = "--help" ] || [ "$1" = "-h" ] || [ -z "$1" ]; then
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        echo "Docker Image Build Script"
        echo ""
    else
        print_error "Error: Tag name is required"
    fi
    echo ""
    echo "Usage: $0 <tag-name> [variant]"
    echo ""
    echo "Examples:"
    echo "  $0 v1.0.0           # Build standard image with tag v1.0.0"
    echo "  $0 dev-build        # Build standard image with tag dev-build"
    echo "  $0 v1.0.0 loaded    # Build loaded variant with tag v1.0.0"
    echo "  $0 latest standard  # Build standard image with tag latest"
    echo ""
    echo "Variants:"
    echo "  standard (default) - Caddy with failover plugin only"
    echo "  loaded            - Caddy with failover, admin-ui, and docker-proxy plugins"
    exit 1
fi

TAG="$1"
VARIANT="${2:-standard}"

# Validate variant
if [ "$VARIANT" != "standard" ] && [ "$VARIANT" != "loaded" ]; then
    print_error "Error: Invalid variant '$VARIANT'"
    echo "Valid variants are: standard, loaded"
    exit 1
fi

# Set image name based on variant
IMAGE_NAME="caddy-failover"
if [ "$VARIANT" = "loaded" ]; then
    FULL_TAG="${IMAGE_NAME}:${TAG}-loaded"
    TARGET="loaded"
else
    FULL_TAG="${IMAGE_NAME}:${TAG}"
    TARGET="standard"
fi

# Get the script's directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

echo "========================================="
echo "Docker Image Build Script"
echo "========================================="
echo ""
print_info "Configuration:"
echo "  • Project: $PROJECT_ROOT"
echo "  • Variant: $VARIANT"
echo "  • Target:  $TARGET"
echo "  • Tag:     $FULL_TAG"
echo ""

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker info &> /dev/null; then
    print_error "Docker daemon is not running"
    echo "Please start Docker and try again"
    exit 1
fi

# Get git information for build args
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
VERSION="${GIT_TAG:-$TAG}"

print_info "Building Docker image..."
echo ""

# Build the Docker image with the specified target
if docker build \
    --target "$TARGET" \
    --build-arg VERSION="$VERSION" \
    --build-arg GIT_TAG="$GIT_TAG" \
    -t "$FULL_TAG" \
    -f Dockerfile \
    .; then

    print_success "Docker image built successfully!"
    echo ""

    # Show image details
    print_info "Image details:"
    docker images "$IMAGE_NAME:$TAG*" --format "table {{.Repository}}:{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"
    echo ""

    # Show build info if the image was built successfully
    print_info "Verifying build..."
    if docker run --rm "$FULL_TAG" caddy version &> /dev/null; then
        print_success "Image verification passed"
        echo ""

        # Display Caddy version
        print_info "Caddy version in image:"
        docker run --rm "$FULL_TAG" caddy version
        echo ""

        # Display build info if available
        if docker run --rm "$FULL_TAG" cat /etc/caddy/build-info.json &> /dev/null; then
            print_info "Build information:"
            docker run --rm "$FULL_TAG" cat /etc/caddy/build-info.json | jq '.' 2>/dev/null || \
                docker run --rm "$FULL_TAG" cat /etc/caddy/build-info.json
            echo ""
        fi
    else
        print_warning "Could not verify image (this might be normal for some configurations)"
    fi

    print_success "Image is ready to use!"
    echo ""
    echo "To run the image:"
    echo "  docker run -p 80:80 -p 443:443 -v \$(pwd)/Caddyfile:/etc/caddy/Caddyfile $FULL_TAG"
    echo ""
    echo "To push to a registry:"
    echo "  docker tag $FULL_TAG <registry>/$FULL_TAG"
    echo "  docker push <registry>/$FULL_TAG"
    echo ""

    # Also tag as latest-variant if building a versioned tag
    if [[ "$TAG" =~ ^v[0-9] ]]; then
        if [ "$VARIANT" = "loaded" ]; then
            LATEST_TAG="${IMAGE_NAME}:latest-loaded"
        else
            LATEST_TAG="${IMAGE_NAME}:latest"
        fi

        print_info "Also tagging as $LATEST_TAG..."
        docker tag "$FULL_TAG" "$LATEST_TAG"
        print_success "Tagged as $LATEST_TAG"
        echo ""
    fi

else
    print_error "Docker build failed"
    exit 1
fi
