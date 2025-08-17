# GitHub App Setup for Automated Releases

This guide explains how to set up a GitHub App to allow the release workflow to commit to the protected main branch.

## Step 1: Create GitHub App

1. Go to [GitHub Settings > Developer settings > GitHub Apps](https://github.com/settings/apps)
2. Click "New GitHub App"
3. Fill in the following:
   - **GitHub App name**: `caddy-failover-release-bot` (or similar)
   - **Homepage URL**: Your repo URL
   - **Webhook**: Uncheck "Active"
   - **Permissions**:
     - Repository permissions:
       - Contents: Read & Write
       - Pull requests: Read & Write (if needed)
       - Metadata: Read
       - Actions: Read (optional)
   - **Where can this GitHub App be installed?**: Only on this account

4. Click "Create GitHub App"
5. Note the **App ID** (you'll need this)

## Step 2: Generate Private Key

1. In your GitHub App settings, scroll to "Private keys"
2. Click "Generate a private key"
3. Save the downloaded `.pem` file

## Step 3: Install the App

1. In your GitHub App settings, click "Install App"
2. Choose your account
3. Select "Only select repositories"
4. Choose `caddy-failover`
5. Click "Install"

## Step 4: Add Secrets to Repository

1. Go to your repository's Settings > Secrets and variables > Actions
2. Add the following secrets:
   - `SEMANTIC_RELEASE_APP_ID`: The App ID from Step 1
   - `SEMANTIC_RELEASE_APP_PRIVATE_KEY`: The entire contents of the `.pem` file from Step 2

## Step 5: Configure Branch Protection

1. Go to Settings > Branches
2. Edit the protection rule for `main`
3. If "Restrict who can push to matching branches" is enabled:
   - Add your GitHub App to the list of actors who can push
   - The app name will be something like `caddy-failover-release-bot[bot]`

## Step 6: Test

Push a commit with a conventional commit message (e.g., `feat: new feature`) to trigger a release.

## Troubleshooting

- If the token generation fails, check that both secrets are properly set
- Ensure the private key includes the full PEM content including headers
- Verify the app has the correct permissions and is installed on the repository
- Check that the app is listed in branch protection bypass rules if needed
