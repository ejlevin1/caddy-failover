# Test Data Directory

This directory contains test configuration files used for validating the Caddy Failover plugin during CI/CD builds.

## Files

### basic.Caddyfile
A minimal Caddyfile configuration used to validate basic plugin functionality:
- Simple failover proxy setup with primary and backup servers
- Basic failover status endpoint
- Minimal timeout configurations

### complex.Caddyfile
A comprehensive Caddyfile configuration used to validate advanced features:
- Multiple service endpoints (auth, admin, API, gateway)
- Health check configurations for each upstream
- Per-upstream header customization
- Environment variable expansion
- Mixed HTTP/HTTPS upstreams
- Various timeout and failure duration settings

## Purpose

These files are used by GitHub Actions workflows to:
1. Validate that the plugin builds correctly with xcaddy
2. Ensure Caddyfile syntax remains compatible across versions
3. Test that all plugin directives are properly registered
4. Verify complex configurations parse without errors

## Important Notes

- **DO NOT DELETE** these files - they are required for CI/CD validation
- These are not functional configurations - they're syntax validation files only
- The endpoints referenced (like `auth-primary.local`) are not expected to exist
- Environment variables in `complex.Caddyfile` are set by the CI workflow

## Usage in CI/CD

The GitHub Actions workflow (`/.github/workflows/test.yml`) uses these files in the "Build and Validate Plugin" job:

```bash
# Validate basic configuration
./caddy validate --config testdata/basic.Caddyfile --adapter caddyfile

# Validate complex configuration with environment variables
export API_PRIMARY_URL=http://localhost:5031
export API_SECONDARY_URL=http://localhost:5032
./caddy validate --config testdata/complex.Caddyfile --adapter caddyfile
```

These validation steps ensure that the plugin remains compatible with Caddy's configuration parser and that all registered directives work correctly.
