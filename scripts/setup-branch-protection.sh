#!/bin/bash

# Setup script for automated branch protection
set -e

echo "========================================="
echo "Branch Protection Automation Setup"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}❌ GitHub CLI (gh) is not installed${NC}"
    echo "Please install it first: https://cli.github.com/"
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}⚠️  You need to authenticate with GitHub CLI${NC}"
    gh auth login
fi

echo -e "${GREEN}✅ GitHub CLI authenticated${NC}"
echo ""

# Get repository info
REPO_OWNER=$(gh repo view --json owner -q .owner.login)
REPO_NAME=$(gh repo view --json name -q .name)

echo "Repository: $REPO_OWNER/$REPO_NAME"
echo ""

# Function to create PAT
create_pat() {
    echo "Creating Personal Access Token..."
    echo ""
    echo "This script will guide you through creating a PAT with the necessary permissions."
    echo ""
    echo -e "${YELLOW}Steps:${NC}"
    echo "1. Opening GitHub settings in your browser..."
    echo "2. Click 'Generate new token (classic)'"
    echo "3. Name it: 'caddy-failover-branch-protection'"
    echo "4. Select scopes:"
    echo "   ✓ repo (Full control of private repositories)"
    echo "   ✓ admin:repo_hook (Full control of repository hooks)"
    echo "5. Click 'Generate token'"
    echo "6. Copy the token (it won't be shown again!)"
    echo ""

    # Open GitHub token creation page
    if [[ "$OSTYPE" == "darwin"* ]]; then
        open "https://github.com/settings/tokens/new?description=caddy-failover-branch-protection&scopes=repo,admin:repo_hook"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        xdg-open "https://github.com/settings/tokens/new?description=caddy-failover-branch-protection&scopes=repo,admin:repo_hook"
    else
        echo "Please open this URL in your browser:"
        echo "https://github.com/settings/tokens/new?description=caddy-failover-branch-protection&scopes=repo,admin:repo_hook"
    fi

    echo ""
    read -sp "Paste your Personal Access Token here: " PAT
    echo ""

    if [ -z "$PAT" ]; then
        echo -e "${RED}❌ No token provided${NC}"
        exit 1
    fi

    # Add the PAT as a repository secret
    echo ""
    echo "Adding PAT as repository secret..."
    echo "$PAT" | gh secret set ADMIN_PAT --repo="$REPO_OWNER/$REPO_NAME"

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ PAT successfully added as ADMIN_PAT secret${NC}"

        # Now configure branch protection with the PAT
        echo ""
        echo "Now configuring branch protection with your new PAT..."
        configure_protection "$PAT"
    else
        echo -e "${RED}❌ Failed to add secret${NC}"
        exit 1
    fi
}

# Function to configure branch protection directly
configure_protection() {
    echo ""
    echo "Configuring branch protection directly..."
    echo ""

    # Ask for PAT if not provided
    if [ -z "$1" ]; then
        echo "Note: Branch protection requires your PAT with admin permissions."
        read -sp "Enter your ADMIN_PAT: " PAT
        echo ""
    else
        PAT="$1"
    fi

    if [ -z "$PAT" ]; then
        echo -e "${RED}❌ No token provided${NC}"
        return 1
    fi

    echo "Applying branch protection rules to main branch..."

    # Configure branch protection using the PAT
    RESPONSE=$(curl -s -X PUT \
        -H "Authorization: token $PAT" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/branches/main/protection" \
        -d '{
          "required_status_checks": {
            "strict": true,
            "contexts": ["test", "build", "commitlint"]
          },
          "enforce_admins": false,
          "required_pull_request_reviews": {
            "required_approving_review_count": 1,
            "dismiss_stale_reviews": true,
            "require_code_owner_reviews": true,
            "require_last_push_approval": false,
            "bypass_pull_request_allowances": {
              "users": ["ejlevin1", "semantic-release-bot"],
              "teams": [],
              "apps": []
            }
          },
          "restrictions": null,
          "allow_force_pushes": false,
          "allow_deletions": false,
          "block_creations": false,
          "required_conversation_resolution": true,
          "lock_branch": false,
          "allow_fork_syncing": false
        }')

    # Check if successful
    if echo "$RESPONSE" | grep -q '"url"'; then
        echo -e "${GREEN}✅ Branch protection configured successfully!${NC}"
        echo ""
        echo "Protection rules applied:"
        echo "  ✓ Require pull request reviews (1 approval)"
        echo "  ✓ Dismiss stale reviews on new commits"
        echo "  ✓ Require CODEOWNERS review"
        echo "  ✓ Require status checks: test, build, commitlint"
        echo "  ✓ Require branches to be up to date"
        echo "  ✓ Require conversation resolution"
        echo "  ✓ No force pushes allowed"
        echo "  ✓ No deletions allowed"
        return 0
    else
        echo -e "${RED}❌ Failed to configure branch protection${NC}"
        echo "Error response:"
        echo "$RESPONSE" | python -m json.tool 2>/dev/null || echo "$RESPONSE"
        return 1
    fi
}

# Main menu
echo "What would you like to do?"
echo ""
echo "1) Create PAT and configure branch protection"
echo "2) Configure branch protection (with existing PAT)"
echo "3) View current branch protection status"
echo "4) Exit"
echo ""
read -p "Select an option (1-4): " choice

case $choice in
    1)
        create_pat
        ;;
    2)
        configure_protection
        ;;
    3)
        echo ""
        echo "Current branch protection for main branch:"
        gh api repos/$REPO_OWNER/$REPO_NAME/branches/main/protection 2>/dev/null | python -m json.tool 2>/dev/null || echo -e "${YELLOW}No branch protection configured${NC}"
        ;;
    4)
        echo "Exiting..."
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid option${NC}"
        exit 1
        ;;
esac

echo ""
echo "========================================="
echo "Setup complete!"
echo "========================================="
