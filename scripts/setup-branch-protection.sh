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
    else
        echo -e "${RED}❌ Failed to add secret${NC}"
        exit 1
    fi
}

# Function to test branch protection
test_protection() {
    echo ""
    echo "Testing branch protection configuration..."
    echo ""

    # Trigger the workflow
    gh workflow run branch-protection.yml --repo="$REPO_OWNER/$REPO_NAME"

    echo "Workflow triggered. Waiting for completion..."
    sleep 5

    # Check workflow status
    RUN_ID=$(gh run list --workflow=branch-protection.yml --limit=1 --json databaseId -q '.[0].databaseId' --repo="$REPO_OWNER/$REPO_NAME")

    if [ -n "$RUN_ID" ]; then
        echo "Watching workflow run #$RUN_ID..."
        gh run watch "$RUN_ID" --repo="$REPO_OWNER/$REPO_NAME"

        # Check if successful
        STATUS=$(gh run view "$RUN_ID" --json conclusion -q .conclusion --repo="$REPO_OWNER/$REPO_NAME")

        if [ "$STATUS" = "success" ]; then
            echo -e "${GREEN}✅ Branch protection configured successfully!${NC}"
        else
            echo -e "${YELLOW}⚠️  Workflow completed with status: $STATUS${NC}"
            echo "Check the workflow logs for details:"
            echo "gh run view $RUN_ID --repo=$REPO_OWNER/$REPO_NAME"
        fi
    else
        echo -e "${YELLOW}⚠️  Could not find workflow run${NC}"
    fi
}

# Main menu
echo "What would you like to do?"
echo ""
echo "1) Create and configure Personal Access Token"
echo "2) Test existing branch protection setup"
echo "3) View current branch protection status"
echo "4) Exit"
echo ""
read -p "Select an option (1-4): " choice

case $choice in
    1)
        create_pat
        test_protection
        ;;
    2)
        test_protection
        ;;
    3)
        echo ""
        echo "Current branch protection for main branch:"
        gh api repos/$REPO_OWNER/$REPO_NAME/branches/main/protection 2>/dev/null | jq '.' || echo -e "${YELLOW}No branch protection configured${NC}"
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
