# Semantic Versioning Guide

This repository uses **Semantic Versioning** (SemVer) with automated releases powered by `semantic-release`.

## Overview

- Versions follow the format: `MAJOR.MINOR.PATCH` (e.g., `1.2.3`)
- Releases are automatically created when commits are pushed to the `main` branch
- Version bumps are determined by commit message prefixes

## Commit Message Format

All commits must follow the **Conventional Commits** specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Commit Types and Version Bumps

| Type | Description | Version Bump | Example |
|------|-------------|--------------|---------|
| `feat` | New feature | Minor (0.X.0) | `feat: add support for custom headers` |
| `fix` | Bug fix | Patch (0.0.X) | `fix: resolve connection timeout issue` |
| `perf` | Performance improvement | Patch (0.0.X) | `perf: optimize upstream selection` |
| `docs` | Documentation only | No release | `docs: update installation guide` |
| `style` | Code style changes | No release | `style: format code with gofmt` |
| `refactor` | Code refactoring | No release | `refactor: simplify error handling` |
| `test` | Test changes | No release | `test: add unit tests for failover` |
| `build` | Build system changes | No release | `build: update Go version to 1.22` |
| `ci` | CI/CD changes | No release | `ci: add semantic release workflow` |
| `chore` | Maintenance tasks | No release | `chore: update dependencies` |
| `revert` | Revert previous commit | Varies | `revert: feat: add custom headers` |

### Breaking Changes

To trigger a **major version bump** (X.0.0), include `BREAKING CHANGE:` in the commit body or footer:

```bash
feat: change configuration format

BREAKING CHANGE: The configuration format has changed from YAML to JSON.
Users must migrate their existing configuration files.
```

Or use the `!` suffix:
```bash
feat!: remove deprecated API endpoints
```

## Examples

### Patch Release (Bug Fix)
```bash
git commit -m "fix: correct header forwarding logic"
# Results in: 1.0.0 → 1.0.1
```

### Minor Release (New Feature)
```bash
git commit -m "feat: add TLS configuration options"
# Results in: 1.0.1 → 1.1.0
```

### Major Release (Breaking Change)
```bash
git commit -m "feat!: restructure plugin API

BREAKING CHANGE: Plugin API has been completely redesigned.
All existing plugins must be updated to use the new API."
# Results in: 1.1.0 → 2.0.0
```

### Multiple Commits in One Release
When multiple commits are pushed:
- The highest severity change determines the version bump
- All changes are included in the release notes

Example:
```bash
fix: resolve memory leak
feat: add retry mechanism
docs: update README
# Results in: Minor release (due to feat)
```

## Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make changes and commit with conventional format**
   ```bash
   git commit -m "feat: add new failover strategy"
   ```

3. **Create a Pull Request**
   - PR title should follow conventional commit format
   - All commits will be validated by commitlint

4. **Merge to main**
   - Once merged, semantic-release automatically:
     - Analyzes commits
     - Determines version bump
     - Creates GitHub release
     - Updates CHANGELOG.md
     - Tags the release
     - Builds and publishes artifacts

## Branch Protection and Semantic Release

The repository is configured with branch protection rules that work with semantic-release:

- **Required checks**: `test`, `build`, `commitlint`
- **Bypass users**: `ejlevin1` (owner), `semantic-release-bot`
- **Automated releases**: The bot can push directly to main for release commits

## GitHub App Setup (Optional)

For better security and to avoid circular builds, you can set up a GitHub App:

1. Create a GitHub App with these permissions:
   - Contents: Write
   - Issues: Write
   - Pull requests: Write
   - Metadata: Read

2. Install the app on your repository

3. Add these secrets to your repository:
   - `SEMANTIC_RELEASE_APP_ID`: Your app's ID
   - `SEMANTIC_RELEASE_APP_PRIVATE_KEY`: Your app's private key

Without a GitHub App, the workflow will fall back to using `GITHUB_TOKEN`.

## Release Assets

Each release automatically includes:
- Binary artifacts for multiple platforms (Linux, macOS, Windows)
- Docker images tagged with the version
- Updated CHANGELOG.md
- GitHub release with release notes

## Troubleshooting

### Commits not triggering releases
- Ensure commits follow conventional format
- Check that commits are on the `main` branch
- Verify GitHub Actions are enabled

### Version not bumping as expected
- Check commit message format
- For breaking changes, ensure `BREAKING CHANGE:` is in the body/footer
- Review semantic-release logs in GitHub Actions

### Branch protection blocking releases
- Ensure `semantic-release-bot` is in the bypass list
- Verify GitHub App has correct permissions (if using)

## Resources

- [Semantic Versioning Specification](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [semantic-release Documentation](https://semantic-release.gitbook.io/)
