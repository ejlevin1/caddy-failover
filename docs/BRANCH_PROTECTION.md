# Branch Protection Setup

This repository uses automated branch protection to ensure code quality and prevent accidental direct pushes to the main branch.

## Automated Setup

The easiest way to configure branch protection is using our setup script:

```bash
./scripts/setup-branch-protection.sh
```

This script will:
1. Guide you through creating a Personal Access Token (PAT)
2. Add it as a repository secret
3. Test the branch protection configuration

## Manual Setup

If you prefer to set up manually:

### Step 1: Create a Personal Access Token

1. Go to [GitHub Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token (classic)"
3. Name it: `caddy-failover-branch-protection`
4. Select these scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `admin:repo_hook` (Full control of repository hooks)
5. Click "Generate token"
6. **Copy the token immediately** (it won't be shown again!)

### Step 2: Add the PAT as a Repository Secret

Using GitHub CLI:
```bash
echo "your-token-here" | gh secret set ADMIN_PAT
```

Or via GitHub UI:
1. Go to repository Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `ADMIN_PAT`
4. Value: Your PAT from Step 1
5. Click "Add secret"

### Step 3: Trigger the Workflow

The branch protection workflow will run automatically on the next push to main, or you can trigger it manually:

```bash
gh workflow run branch-protection.yml
```

## Branch Protection Rules

Once configured, the main branch will have these protections:

- ✅ **Require pull request reviews** (1 approval required)
- ✅ **Dismiss stale PR approvals** when new commits are pushed
- ✅ **Require review from CODEOWNERS**
- ✅ **Require status checks to pass**: `test`, `build`, `commitlint`
- ✅ **Require branches to be up to date** before merging
- ✅ **Require conversation resolution** before merging
- ❌ **No force pushes allowed**
- ❌ **No branch deletions allowed**

### Bypass for Repository Owner

The repository owner (`ejlevin1`) and the `semantic-release-bot` can bypass PR requirements for emergency fixes and automated releases.

## Troubleshooting

### Permission Denied (403)

If you see a 403 error, it means the token doesn't have sufficient permissions:

1. Verify your PAT has both `repo` and `admin:repo_hook` scopes
2. Make sure the `ADMIN_PAT` secret is properly set
3. Check that you're the repository owner or have admin access

### Workflow Not Found

If the workflow can't be found:

1. Make sure you're on the main branch
2. Check that `.github/workflows/branch-protection.yml` exists
3. Push any pending changes to trigger workflow discovery

### Testing Your Setup

To verify branch protection is working:

```bash
# Check current protection status
gh api repos/$(gh repo view --json nameWithOwner -q .nameWithOwner)/branches/main/protection

# Or use the setup script
./scripts/setup-branch-protection.sh
# Select option 3
```

## GitHub App Alternative

For organizations or multiple repositories, consider using a GitHub App instead of a PAT:

1. [Create a GitHub App](https://docs.github.com/en/apps/creating-github-apps)
2. Grant it `Administration: write` and `Contents: write` permissions
3. Install it on your repository
4. Use the app's token in the workflow

This approach is more secure and doesn't require personal tokens.

## Security Notes

- **Never commit your PAT** to the repository
- PATs should have minimal required scopes
- Rotate PATs regularly (recommended: every 90 days)
- Use GitHub Apps for production/organization use
- Review branch protection settings periodically

## Related Documentation

- [GitHub Branch Protection API](https://docs.github.com/en/rest/branches/branch-protection)
- [GitHub Actions Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
