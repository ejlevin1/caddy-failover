# Git Hooks Setup Guide

This repository uses git hooks to enforce code quality and conventional commit standards. We use a combination of **pre-commit** (Python-based but language agnostic) and **cocogitto** (Rust-based, single binary) to avoid Node.js dependencies.

## Quick Setup

Run the setup script to install all necessary tools and hooks:

```bash
./scripts/setup-hooks.sh
```

This will:
1. Install pre-commit framework
2. Install cocogitto (conventional commits tool)
3. Install required Go tools (gosec)
4. Configure git hooks
5. Run initial validation

## Tools Used

### 1. Pre-commit Framework
- **What**: Multi-language pre-commit hook manager
- **Why**: Language agnostic, extensive plugin ecosystem
- **Installation**: Via pip or Homebrew
- **Config**: `.pre-commit-config.yaml`

### 2. Cocogitto
- **What**: Rust-based conventional commits toolbox
- **Why**: Single binary, no Node.js/npm required, fast
- **Installation**: Direct binary download
- **Config**: `cog.toml`

## Hooks Configured

### Commit-msg Hook
- **Validates conventional commit format** using cocogitto
- Runs on every commit
- Ensures commits follow: `<type>(<scope>): <subject>`

### Pre-commit Hooks
- **Go formatting** (`go fmt`)
- **Go vetting** (`go vet`)
- **Go module tidying** (`go mod tidy`)
- **Go tests** (short mode)
- **Security scanning** (gosec)
- **Secret detection** (detect-secrets)
- **File formatting** (trailing whitespace, EOF, line endings)

### Pre-push Hook
- Runs tests
- Validates formatting
- Runs go vet

## Manual Installation

If you prefer manual installation:

### 1. Install Pre-commit

```bash
# macOS
brew install pre-commit

# Linux/Unix (via pip)
pip3 install --user pre-commit

# Windows
pip install --user pre-commit
```

### 2. Install Cocogitto

Download the appropriate binary from [releases](https://github.com/cocogitto/cocogitto/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/cocogitto/cocogitto/releases/download/6.1.0/cocogitto-6.1.0-aarch64-apple-darwin.tar.gz | tar -xz
sudo mv cog /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/cocogitto/cocogitto/releases/download/6.1.0/cocogitto-6.1.0-x86_64-apple-darwin.tar.gz | tar -xz
sudo mv cog /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/cocogitto/cocogitto/releases/download/6.1.0/cocogitto-6.1.0-x86_64-unknown-linux-musl.tar.gz | tar -xz
sudo mv cog /usr/local/bin/
```

### 3. Install Go Tools

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### 4. Install Hooks

```bash
# Install pre-commit hooks
pre-commit install --install-hooks
pre-commit install --hook-type commit-msg

# Install cocogitto hooks
cog install-hook --all
```

## Usage

### Making Commits

#### Option 1: Interactive (Recommended)
```bash
cog commit
# Follow the interactive prompts
```

#### Option 2: Standard Git
```bash
git commit -m "feat: add new feature"
git commit -m "fix: resolve bug"
git commit -m "docs: update README"
```

### Commit Types

| Type | Description | Version Impact |
|------|-------------|----------------|
| `feat` | New feature | Minor bump |
| `fix` | Bug fix | Patch bump |
| `perf` | Performance improvement | Patch bump |
| `docs` | Documentation only | No release |
| `style` | Code style (formatting) | No release |
| `refactor` | Code refactoring | No release |
| `test` | Test changes | No release |
| `build` | Build system changes | No release |
| `ci` | CI/CD changes | No release |
| `chore` | Maintenance tasks | No release |
| `revert` | Revert previous commit | Varies |

### Examples

```bash
# Simple commits
git commit -m "feat: add retry mechanism"
git commit -m "fix: resolve timeout issue"
git commit -m "docs: update installation guide"

# With scope
git commit -m "feat(api): add new endpoint"
git commit -m "fix(failover): correct retry logic"

# Breaking change
git commit -m "feat!: change config format

BREAKING CHANGE: Configuration now uses TOML instead of YAML"
```

## Testing Hooks

### Test All Hooks
```bash
pre-commit run --all-files
```

### Test Specific Hook
```bash
pre-commit run go-fmt --all-files
pre-commit run check-commit-msg --all-files
```

### Bypass Hooks (Emergency Only)
```bash
git commit --no-verify -m "emergency: fix critical issue"
```

## Troubleshooting

### "cog: command not found"
- Ensure `~/.local/bin` or `/usr/local/bin` is in your PATH
- Run `./scripts/setup-hooks.sh` to reinstall

### "pre-commit: command not found"
- Install pre-commit: `pip3 install --user pre-commit`
- Ensure Python pip bin directory is in PATH

### Commit Rejected
- Check the error message for the specific format issue
- Use `cog commit` for interactive help
- Ensure format: `<type>(<scope>): <subject>`

### Hook Not Running
```bash
# Reinstall hooks
pre-commit install --install-hooks
cog install-hook --all
```

## Benefits

1. **No Node.js Required**: Uses Rust (cocogitto) and Python (pre-commit)
2. **Fast Validation**: Cocogitto is very fast being a Rust binary
3. **Comprehensive Checks**: Format, security, tests, and commit messages
4. **CI/CD Integration**: Same rules enforced locally and in CI
5. **Developer Friendly**: Interactive mode with `cog commit`

## Maintenance

### Update Hook Versions
```bash
# Update pre-commit hooks
pre-commit autoupdate

# Update cocogitto (check for new version)
# Then update COG_VERSION in scripts/setup-hooks.sh
```

### Skip Hooks for Specific Files
Add to `.pre-commit-config.yaml`:
```yaml
exclude: '^(vendor/|third_party/)'
```

## Resources

- [Pre-commit Documentation](https://pre-commit.com/)
- [Cocogitto Documentation](https://docs.cocogitto.io/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Project Semantic Versioning Guide](./SEMANTIC_VERSIONING.md)
