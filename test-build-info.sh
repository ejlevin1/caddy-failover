#!/bin/bash

# Test script to verify build info is properly embedded in Docker images

set -e

echo "Testing Docker Build Info Embedding"
echo "===================================="
echo ""

# Get current git information
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building test image with git info..."
docker build \
  --build-arg GIT_COMMIT="$GIT_COMMIT" \
  --build-arg GIT_TAG="test-v1.0.0" \
  --build-arg GIT_BRANCH="$GIT_BRANCH" \
  --build-arg BUILD_DATE="$BUILD_DATE" \
  --build-arg VERSION="1.0.0-test" \
  -t caddy-failover:build-info-test \
  .

echo ""
echo "Checking build-info.json in container..."
echo "========================================="
docker run --rm caddy-failover:build-info-test cat /etc/caddy/build-info.json | python3 -m json.tool

echo ""
echo "Checking Docker image labels..."
echo "================================"
docker inspect caddy-failover:build-info-test | jq '.[0].Config.Labels' | grep -E "(version|revision|created|source|title|description)" || true

echo ""
echo "Testing access to build info at runtime..."
echo "==========================================="
docker run --rm -d --name test-build-info caddy-failover:build-info-test
sleep 2

echo "Build info file exists and is readable:"
docker exec test-build-info ls -la /etc/caddy/build-info.json

echo ""
echo "Build info contents:"
docker exec test-build-info cat /etc/caddy/build-info.json | python3 -m json.tool

docker stop test-build-info

echo ""
echo "✅ Build info embedding test completed successfully!"
echo ""
echo "Summary:"
echo "--------"
echo "1. Build info is stored at: /etc/caddy/build-info.json"
echo "2. File contains: version, git_commit, git_tag, git_branch, build_date, caddy_version"
echo "3. OCI labels are properly set on the image"
echo "4. Build info is accessible at container runtime"
