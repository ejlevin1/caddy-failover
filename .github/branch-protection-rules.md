# Branch Protection Rules

This document describes the branch protection rules configured for this repository.

## Automated Configuration

The branch protection rules can be automatically configured by running the GitHub Action workflow:
`.github/workflows/branch-protection.yml`

To apply the rules:
1. Push this workflow to the main branch
2. Go to Actions tab in GitHub
3. Select "Configure Branch Protection" workflow
4. Click "Run workflow"

## Manual Configuration

If automatic configuration fails or you prefer manual setup, configure these settings in GitHub:

### Navigate to Settings → Branches

Add a branch protection rule for `main` with these settings:

### Required Settings
- ✅ **Require a pull request before merging**
  - Required approving reviews: 1
  - Dismiss stale pull request approvals when new commits are pushed
  - Require review from CODEOWNERS
  - Do not require approval of the most recent reviewable push
  - **Allow specified actors to bypass required pull requests**: `ejlevin1` (repository owner)

- ✅ **Require status checks to pass before merging**
  - Required status checks:
    - `test`
    - `build`
    - `commitlint`
  - Require branches to be up to date before merging

- ✅ **Require conversation resolution before merging**

### Restrictions
- ✅ **Do not allow force pushes**
- ✅ **Do not allow deletions**

### Optional Settings (can be enabled based on needs)
- ⬜ Require signed commits
- ⬜ Require linear history
- ⬜ Include administrators
- ⬜ Restrict who can push to matching branches

## Bypass Settings

The following users can bypass pull request requirements:
- **`ejlevin1`** (repository owner): For emergency fixes and maintenance
- **`semantic-release-bot`**: For automated version bumps and changelog updates

This allows:
- Owner to merge directly to main without reviews when necessary
- Semantic-release to automatically commit version updates
- Smooth CI/CD pipeline without manual intervention

## Required GitHub Permissions

For the automated configuration to work:
- The workflow uses standard GitHub token permissions
- Repository settings access is required (usually available to repository admins)

## Status Checks

The following CI checks must pass before merging:
1. **test** - Runs the test suite
2. **build** - Builds the Caddy binary with the plugin
3. **commitlint** - Validates commit messages follow conventional format

These checks are defined in separate workflow files and are referenced by the branch protection rules.

## Semantic Versioning Integration

This repository uses semantic-release for automated versioning:
- Commits must follow Conventional Commits format
- Version bumps are determined by commit types (feat, fix, etc.)
- Releases are automatically created when merging to main
- The `semantic-release-bot` can bypass protection rules to commit release updates

See [SEMANTIC_VERSIONING.md](../SEMANTIC_VERSIONING.md) for detailed commit guidelines.
