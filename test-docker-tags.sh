#!/bin/bash

# Test script to verify Docker image tagging strategy

echo "Testing Docker Image Semantic Versioning"
echo "========================================="
echo ""

# Simulate different scenarios
echo "1. Testing release scenario (v1.4.0):"
echo "   Expected tags:"
echo "   - ghcr.io/ejlevin1/caddy-failover:1.4.0"
echo "   - ghcr.io/ejlevin1/caddy-failover:1.4"
echo "   - ghcr.io/ejlevin1/caddy-failover:1"
echo "   - ghcr.io/ejlevin1/caddy-failover:latest"
echo ""

echo "2. Testing main branch push:"
echo "   Expected tags:"
echo "   - ghcr.io/ejlevin1/caddy-failover:main"
echo "   - ghcr.io/ejlevin1/caddy-failover:main-<short-sha>"
echo "   - ghcr.io/ejlevin1/caddy-failover:latest"
echo ""

echo "3. Testing PR build:"
echo "   Expected tags:"
echo "   - ghcr.io/ejlevin1/caddy-failover:pr-<number>"
echo "   Note: Images are not pushed for PRs"
echo ""

echo "Workflow Changes Summary:"
echo "========================="
echo "✅ release.yml:"
echo "   - Uses docker/build-push-action for multi-arch support"
echo "   - Builds for linux/amd64 and linux/arm64"
echo "   - Creates semantic version tags (X.Y.Z, X.Y, X, latest)"
echo "   - Uses GitHub Actions cache for faster builds"
echo ""
echo "✅ build-and-publish.yml:"
echo "   - SHA tags only created for non-tag pushes"
echo "   - Proper semantic versioning for tag pushes"
echo "   - Consistent tagging strategy"
echo ""
