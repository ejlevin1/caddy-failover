# Contributing to Caddy Failover Plugin

Thank you for your interest in contributing!

## Development Setup

1. **Prerequisites**
   - Go 1.22 or later
   - Docker (for testing)
   - xcaddy (for building): `go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest`

2. **Clone the repository**
   ```bash
   git clone https://github.com/ejlevin1/caddy-failover.git
   cd caddy-failover
   ```

3. **Set up git hooks** (REQUIRED)
   ```bash
   ./scripts/setup-hooks.sh
   ```
   This installs pre-commit hooks and conventional commit validation.

4. **Build with xcaddy**
   ```bash
   make xcaddy-build
   # or
   xcaddy build --with github.com/ejlevin1/caddy-failover=.
   ```

4. **Run tests**
   ```bash
   make test
   make docker-test
   ```

## Making Changes

### Plugin Development

The plugin is in `failover.go`. Key areas:

- `FailoverProxy` struct: Main handler structure
- `ServeHTTP` method: Request handling logic
- `tryUpstream` method: Individual upstream attempt logic
- `parseFailoverProxy` function: Caddyfile parsing

### Adding Features

1. Create a feature branch
2. Make your changes
3. Add tests if applicable
4. Update documentation
5. Submit a pull request

### Testing Changes

1. **Go tests**:
   ```bash
   go test -v -race ./...
   ```

2. **Build and validate**:
   ```bash
   make xcaddy-build
   ./caddy validate --config examples/basic-caddyfile
   ```

3. **Integration tests**:
   ```bash
   make docker-test
   ```

4. **Manual testing**:
   ```bash
   make xcaddy-run
   ```

## Commit Message Format

All commits MUST follow the Conventional Commits specification:

```
<type>(<scope>): <subject>
```

**Types**: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert

**Examples**:
- `feat: add retry mechanism`
- `fix(failover): resolve timeout issue`
- `docs: update installation guide`
- `feat!: breaking change in config format`

Use `cog commit` for interactive commit creation.

## Pull Request Process

1. Ensure all tests pass
2. Run `make verify` to check formatting
3. Commits must follow conventional format
4. Update README.md if needed
5. Update examples if behavior changes
6. Submit PR with clear description

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting: `make fmt`
- Add comments for exported functions
- Keep functions focused and small

## Testing with xcaddy

When developing, use xcaddy for quick iteration:

```bash
# Make changes to failover.go
vim failover.go

# Build and test
xcaddy build --with github.com/ejlevin1/caddy-failover=.
./caddy run --config test.Caddyfile
```

## Reporting Issues

Please use GitHub Issues to report bugs or request features. Include:
- Caddy version
- Plugin version
- Caddyfile configuration
- Error messages/logs
- Steps to reproduce

## Release Process

Releases are automated using semantic-release:

1. Merge PRs with conventional commits to `main`
2. Semantic-release automatically:
   - Determines version bump from commits
   - Creates GitHub release
   - Updates CHANGELOG.md
   - Builds and publishes artifacts
   - Tags the release

See [SEMANTIC_VERSIONING.md](./SEMANTIC_VERSIONING.md) for details.
