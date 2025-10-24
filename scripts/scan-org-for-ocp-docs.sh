#!/bin/bash

# Script to scan GitHub organization repositories for OpenShift documentation links
# Usage: ./scan-org-for-ocp-docs.sh <org-name> [output-file]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to display help
show_help() {
    cat << EOF
========================================
OCP Documentation Link Scanner
========================================

DESCRIPTION:
  Scans GitHub organization repositories for OpenShift Container Platform
  documentation links and optionally fixes outdated links automatically.

USAGE:
  $0 [OPTIONS] <org-name> [output-file]

  Note: All OPTIONS must be specified BEFORE the org-name argument.

ARGUMENTS:
  org-name        GitHub organization name to scan (required)
  output-file     Output file path (default: ocp-docs-scan-results.txt)

OPTIONS:
  -h, --help      Show this help message and exit
  --fix           Run ocp-doc-checker in fix mode and create pull requests
                  for repositories with outdated documentation links.
                  This is INTERACTIVE and will prompt you before:
                    • Committing and pushing changes to each repository
                    • Creating forks (if needed)
                    • Creating pull requests
                  Note: If you already have an open PR, it will checkout that
                  branch, re-run the checker, and force-push updates if needed.
                  This ensures existing PRs stay current with latest OCP versions
  
  --force         Non-interactive mode. Automatically accepts all prompts.
                  Use with --fix to run without user interaction.
                  ⚠ WARNING: This will automatically create forks and PRs
                  for all repositories with outdated documentation links.
  
  --link-to URL   Add a tracking issue link to all created pull requests.
                  The URL will be added to the PR description as:
                  "Tracking Issue: <URL>"
                  Example: --link-to https://github.com/owner/repo/issues/18
  
  --cache-info    Display information about cached repositories
  --clear-cache   Clear the repository cache

EXAMPLES:
  # Scan an organization (read-only)
  $0 openshift-kni

  # Scan and save to custom output file
  $0 openshift-kni my-scan-results.txt

  # Scan and automatically fix + create PRs (interactive)
  $0 --fix openshift-kni

  # Scan, fix, and link all PRs to a tracking issue
  $0 --fix --link-to https://github.com/sebrandon1/ocp-doc-checker/issues/18 openshift-kni
  
  # Update existing PRs with latest OCP versions (e.g., after 4.20 release)
  # The script will checkout existing PR branches and re-run the checker
  $0 --fix openshift-kni

  # Non-interactive mode - automatically accept all prompts
  $0 --fix --force openshift-kni

  # Full automation with tracking issue
  $0 --fix --force --link-to https://github.com/owner/repo/issues/18 openshift

  # View cache information
  $0 --cache-info

  # Clear the cache
  $0 --clear-cache

CACHE:
  The script caches cloned repositories to speed up subsequent scans.
  Cache location: ~/.cache/ocp-doc-scanner

REQUIREMENTS:
  • GitHub CLI (gh) must be installed and authenticated
  • For --fix mode: ocp-doc-checker binary must be available

OUTPUT:
  Results are saved to the specified output file with details including:
    • Repositories with OCP documentation links
    • Files containing OCP docs
    • ocp-doc-checker integration status
    • Open PRs related to ocp-doc-checker
    • Summary statistics and performance metrics

EOF
}

# Check for help flag first
if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    show_help
    exit 0
fi

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed.${NC}"
    echo "Please install it from: https://cli.github.com/"
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub CLI.${NC}"
    echo "Please run: gh auth login"
    exit 1
fi

# Parse flags
FIX_MODE=false
FORCE_MODE=false
TRACKING_LINK=""
while [[ "$1" =~ ^-- ]]; do
    case "$1" in
        --clear-cache)
            echo -e "${YELLOW}Clearing cache...${NC}"
            rm -rf "${HOME}/.cache/ocp-doc-scanner"
            echo -e "${GREEN}✓ Cache cleared${NC}"
            exit 0
            ;;
        --cache-info)
            CACHE_DIR="${HOME}/.cache/ocp-doc-scanner"
            CACHE_FILE="${CACHE_DIR}/repo-cache.txt"
            
            if [ -d "$CACHE_DIR" ]; then
                cached_count=$(wc -l < "$CACHE_FILE" 2>/dev/null | tr -d ' ')
                cache_size=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)
                
                echo -e "${BLUE}Cache Information:${NC}"
                echo -e "  Location: ${CACHE_DIR}"
                echo -e "  Cached repositories: ${GREEN}${cached_count}${NC}"
                echo -e "  Cache size: ${GREEN}${cache_size}${NC}"
                echo ""
                echo -e "${YELLOW}Cached repositories:${NC}"
                while IFS='|' read -r repo commit timestamp; do
                    if [ -n "$repo" ]; then
                        date_str=$(date -r "$timestamp" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "unknown")
                        echo -e "  ${GREEN}•${NC} $repo"
                        echo -e "    Commit: ${commit:0:7}"
                        echo -e "    Cached: $date_str"
                    fi
                done < "$CACHE_FILE"
            else
                echo -e "${YELLOW}No cache found${NC}"
            fi
            exit 0
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        --force)
            FORCE_MODE=true
            shift
            ;;
        --link-to)
            if [ -z "$2" ] || [[ "$2" =~ ^-- ]]; then
                echo -e "${RED}Error: --link-to requires a URL argument${NC}"
                exit 1
            fi
            TRACKING_LINK="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
done

# Check for required parameters
if [ -z "$1" ]; then
    echo -e "${RED}Error: Organization name is required${NC}"
    echo "Usage: $0 [options] <org-name> [output-file]"
    echo ""
    echo "Options:"
    echo "  --fix           Run ocp-doc-checker in fix mode and create PRs"
    echo "  --force         Non-interactive mode (auto-accept all prompts)"
    echo "  --link-to URL   Add tracking issue link to PRs"
    echo "  --clear-cache   Clear the repository cache"
    echo "  --cache-info    Show cache information"
    echo ""
    echo "Note: All options (--fix, --force, --link-to, etc.) must come BEFORE <org-name>"
    exit 1
fi

# Check if someone put options after the org name
if [[ "$2" =~ ^-- ]]; then
    echo -e "${RED}Error: Options must come before positional arguments${NC}"
    echo -e "${YELLOW}You provided: $0 $*${NC}"
    echo ""
    echo -e "Correct format:"
    echo -e "  ${GREEN}$0 [options] <org-name> [output-file]${NC}"
    echo ""
    echo -e "For example:"
    echo -e "  ${GREEN}$0 --fix --link-to https://... openshift${NC}"
    echo -e "  ${RED}$0 --fix openshift --link-to https://...${NC}  ← Wrong!"
    exit 1
fi

ORG_NAME="$1"
OUTPUT_FILE="${2:-ocp-docs-scan-results.txt}"

# Convert OUTPUT_FILE to absolute path to avoid writing to repo directories
if [[ ! "$OUTPUT_FILE" = /* ]]; then
    OUTPUT_FILE="$(pwd)/${OUTPUT_FILE}"
fi

# OCP documentation URL patterns to search for
DOC_PATTERN="docs\.redhat\.com/en/documentation/openshift_container_platform"

# Check if fix mode requires ocp-doc-checker
if [ "$FIX_MODE" = true ]; then
    # Find ocp-doc-checker binary
    OCP_CHECKER=""
    if [ -f "./ocp-doc-checker" ]; then
        OCP_CHECKER="$(pwd)/ocp-doc-checker"
    elif [ -f "$(dirname "$0")/../ocp-doc-checker" ]; then
        OCP_CHECKER="$(cd "$(dirname "$0")/.." && pwd)/ocp-doc-checker"
    elif command -v ocp-doc-checker &> /dev/null; then
        OCP_CHECKER="$(command -v ocp-doc-checker)"
    else
        echo -e "${RED}Error: ocp-doc-checker binary not found${NC}"
        echo "Please build it first: make build"
        exit 1
    fi
    
    # Verify the binary is executable
    if [ ! -x "$OCP_CHECKER" ]; then
        echo -e "${RED}Error: ocp-doc-checker binary is not executable${NC}"
        echo "Found at: $OCP_CHECKER"
        exit 1
    fi
    
    echo -e "${GREEN}Using ocp-doc-checker: ${OCP_CHECKER}${NC}"
    
    if [ "$FORCE_MODE" = true ]; then
        echo -e "${RED}⚠ Fix mode enabled (NON-INTERACTIVE/FORCE) - will automatically accept all prompts${NC}"
    else
        echo -e "${YELLOW}⚠ Fix mode enabled (INTERACTIVE) - will prompt before each action${NC}"
    fi
    
    if [ -n "$TRACKING_LINK" ]; then
        echo -e "${BLUE}ℹ Tracking issue will be linked in all PRs: ${TRACKING_LINK}${NC}"
    fi
    echo ""
fi

# Set up cache directory
CACHE_DIR="${HOME}/.cache/ocp-doc-scanner"
CACHE_FILE="${CACHE_DIR}/repo-cache.txt"
CACHE_REPOS_DIR="${CACHE_DIR}/repos"

# Create cache directories if they don't exist
mkdir -p "$CACHE_DIR"
mkdir -p "$CACHE_REPOS_DIR"

# Create cache file if it doesn't exist
touch "$CACHE_FILE"

# Temp directory for cloning new repos
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Function to get cached commit for a repo
get_cached_commit() {
    local repo="$1"
    grep "^${repo}|" "$CACHE_FILE" 2>/dev/null | cut -d'|' -f2
}

# Function to update cache for a repo
update_cache() {
    local repo="$1"
    local commit="$2"
    local timestamp=$(date +%s)
    
    # Remove old entry if exists
    sed -i.bak "/^${repo//\//\\/}|/d" "$CACHE_FILE" 2>/dev/null
    rm -f "${CACHE_FILE}.bak"
    
    # Add new entry
    echo "${repo}|${commit}|${timestamp}" >> "$CACHE_FILE"
}

# Function to get repo directory (cached or temp)
get_repo_dir() {
    local repo="$1"
    local repo_name=$(echo "$repo" | sed 's/\//__/g')
    echo "${CACHE_REPOS_DIR}/${repo_name}"
}

# Function to prompt user for confirmation
prompt_user() {
    local message="$1"
    local response
    
    # If force mode is enabled, automatically accept all prompts
    if [ "$FORCE_MODE" = true ]; then
        echo -e "${YELLOW}${message}${NC}"
        echo -e "${BLUE}Continue? [y/N]:${NC} ${GREEN}y${NC} ${BLUE}(forced)${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}${message}${NC}"
    read -p "$(echo -e "${BLUE}Continue? [y/N]:${NC} ")" response </dev/tty
    
    case "$response" in
        [yY][eE][sS]|[yY]) 
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Function to check if user has a fork of the repo
check_fork_exists() {
    local repo="$1"
    local username=$(gh api user --jq '.login')
    local repo_name=$(basename "$repo")
    
    gh repo view "${username}/${repo_name}" &> /dev/null
    return $?
}

# Function to create a fork
create_fork() {
    local repo="$1"
    echo -e "    ${YELLOW}Creating fork of $repo...${NC}"
    
    if gh repo fork "$repo" --clone=false &> /dev/null; then
        echo -e "    ${GREEN}✓ Fork created${NC}"
        # Wait a moment for fork to be ready
        sleep 2
        return 0
    else
        echo -e "    ${RED}✗ Failed to create fork${NC}"
        return 1
    fi
}

# Function to run ocp-doc-checker and create PR if changes are made
fix_and_create_pr() {
    local repo="$1"
    local repo_dir="$2"
    local default_branch="$3"
    
    # Get the authenticated user first
    local username=$(gh api user --jq '.login')
    
    # Check if there's already an open PR from this user for OCP docs
    echo -e "    ${YELLOW}Checking for existing open PRs...${NC}"
    local existing_prs=$(gh pr list --repo "$repo" --author "$username" --state open --search "OCP documentation" --json number,title,headRefName --limit 10 2>&1)
    
    # Validate that we got valid JSON
    if ! echo "$existing_prs" | jq empty 2>/dev/null; then
        # Not valid JSON, likely an error - treat as no PRs
        existing_prs="[]"
    fi
    
    local pr_count=$(echo "$existing_prs" | jq '. | length' 2>/dev/null || echo "0")
    
    # Check for existing PR and get branch name
    local existing_pr_num=""
    local existing_branch=""
    if [ "$pr_count" -gt 0 ]; then
        # Check if any PR has our title or branch prefix
        local matching_pr=$(echo "$existing_prs" | jq -r '.[] | select(.title | contains("Update OCP documentation") or (.headRefName | startswith("update-ocp-docs-"))) | {number: .number, branch: .headRefName} | @json' 2>/dev/null | head -1)
        
        if [ -n "$matching_pr" ]; then
            existing_pr_num=$(echo "$matching_pr" | jq -r '.number')
            existing_branch=$(echo "$matching_pr" | jq -r '.branch')
            local pr_url="https://github.com/$repo/pull/$existing_pr_num"
            echo -e "    ${YELLOW}⚠ Open PR already exists: #${existing_pr_num}${NC}"
            echo -e "    ${BLUE}  $pr_url${NC}"
            echo -e "    ${BLUE}  Branch: ${existing_branch}${NC}"
            echo -e "    ${YELLOW}↻ Will checkout and update the existing PR with latest changes${NC}"
        fi
    fi
    
    # Navigate to repo directory
    cd "$repo_dir"
    
    # If there's an existing PR, checkout its branch from the fork
    if [ -n "$existing_pr_num" ]; then
        local repo_name=$(basename "$repo")
        local fork_url="https://github.com/${username}/${repo_name}.git"
        
        # Add fork as remote if not already added
        if ! git remote get-url fork &> /dev/null; then
            echo -e "    ${YELLOW}Adding fork remote...${NC}"
            git remote add fork "$fork_url"
        fi
        
        # Fetch the fork
        echo -e "    ${YELLOW}Fetching fork...${NC}"
        if ! git fetch fork "$existing_branch" > /dev/null 2>&1; then
            echo -e "    ${RED}✗ Failed to fetch branch from fork${NC}"
            cd - > /dev/null
            return 1
        fi
        
        # Checkout the existing PR branch
        echo -e "    ${YELLOW}Checking out branch: ${existing_branch}${NC}"
        if ! git checkout -B "$existing_branch" "fork/$existing_branch" > /dev/null 2>&1; then
            echo -e "    ${RED}✗ Failed to checkout branch${NC}"
            cd - > /dev/null
            return 1
        fi
        
        echo -e "    ${GREEN}✓ Checked out existing PR branch${NC}"
        
        # Clean up any scan result files that may have been committed in previous runs
        if [ -f "ocp-docs-scan-results.txt" ] || [ -f "ocp-docs-scan-results.txt.prs" ]; then
            echo -e "    ${YELLOW}Cleaning up old scan result files from previous commits...${NC}"
            git rm -f ocp-docs-scan-results.txt ocp-docs-scan-results.txt.prs 2>/dev/null || true
            rm -f ocp-docs-scan-results.txt ocp-docs-scan-results.txt.prs 2>/dev/null || true
        fi
    fi
    
    # Also clean up any scan result files in newly cloned repos
    if [ -f "ocp-docs-scan-results.txt" ] || [ -f "ocp-docs-scan-results.txt.prs" ]; then
        rm -f ocp-docs-scan-results.txt ocp-docs-scan-results.txt.prs 2>/dev/null || true
    fi
    
    echo -e "    ${YELLOW}Running ocp-doc-checker in fix mode...${NC}"
    
    # Run ocp-doc-checker on the directory
    
    # Capture the output for diagnostics and run the checker
    checker_output=$($OCP_CHECKER -dir . -fix 2>&1)
    checker_exit_code=$?
    
    # Check if there are any changes
    if ! git diff --quiet; then
        echo -e "    ${GREEN}✓ Found changes to commit${NC}"
        
        # Show a summary of changes
        echo -e "    ${BLUE}ℹ Changes summary:${NC}"
        git diff --stat | sed 's/^/      /'
        echo ""
        
        # Prompt before proceeding with commit and PR
        if ! prompt_user "    Commit and push these changes for $repo?"; then
            echo -e "    ${YELLOW}⊘ Changes skipped - not committed${NC}"
            cd - > /dev/null
            return 0
        fi
        
        # Get repo name
        local repo_name=$(basename "$repo")
        
        # Determine if we're updating an existing PR or creating a new one
        local is_updating_pr=false
        local branch_name=""
        
        if [ -n "$existing_pr_num" ]; then
            # Updating existing PR - use the existing branch
            branch_name="$existing_branch"
            is_updating_pr=true
            echo -e "    ${BLUE}ℹ Updating existing PR branch: ${branch_name}${NC}"
        else
            # Creating new PR - check fork exists and create new branch
            if ! check_fork_exists "$repo"; then
                # Prompt before creating fork
                if ! prompt_user "    Fork of $repo doesn't exist. Create fork?"; then
                    echo -e "    ${YELLOW}⊘ Fork creation skipped - cannot proceed without fork${NC}"
                    cd - > /dev/null
                    return 1
                fi
                
                if ! create_fork "$repo"; then
                    echo -e "    ${RED}✗ Cannot proceed without fork${NC}"
                    cd - > /dev/null
                    return 1
                fi
            else
                echo -e "    ${GREEN}✓ Fork already exists${NC}"
            fi
            
            # Create a new branch
            branch_name="update-ocp-docs-$(date +%Y%m%d-%H%M%S)"
            echo -e "    ${YELLOW}Creating branch: $branch_name${NC}"
            git checkout -b "$branch_name" > /dev/null 2>&1
        fi
        
        # Stage only modified tracked files (not untracked/new files)
        # This prevents accidentally committing script output files or other artifacts
        git add -u
        
        # Create commit
        local commit_msg="Update OCP documentation links to latest versions

This commit updates OpenShift Container Platform documentation URLs
to point to the latest available versions.

Updated by: ocp-doc-checker
More info: https://github.com/sebrandon1/ocp-doc-checker"
        
        git commit -m "$commit_msg" > /dev/null 2>&1
        echo -e "    ${GREEN}✓ Changes committed${NC}"
        
        # Add fork as remote if not already added
        local fork_url="https://github.com/${username}/${repo_name}.git"
        if ! git remote get-url fork &> /dev/null; then
            git remote add fork "$fork_url"
        fi
        
        # Push to fork
        if [ "$is_updating_pr" = true ]; then
            # Force push to update existing PR
            echo -e "    ${YELLOW}Force pushing to update existing PR...${NC}"
            if git push -f fork "$branch_name" > /dev/null 2>&1; then
                echo -e "    ${GREEN}✓ Force pushed to fork - PR updated${NC}"
                
                # Add a comment to the PR
                local comment="Updated with latest OCP documentation links (ran ocp-doc-checker again).

All documentation URLs now point to the latest available versions."
                
                echo -e "    ${YELLOW}Adding comment to PR...${NC}"
                if gh pr comment "$existing_pr_num" --repo "$repo" --body "$comment" > /dev/null 2>&1; then
                    echo -e "    ${GREEN}✓ Comment added to PR #${existing_pr_num}${NC}"
                fi
                
                local pr_url="https://github.com/$repo/pull/$existing_pr_num"
                echo -e "    ${GREEN}✓ Pull request updated: $pr_url${NC}"
                cd - > /dev/null
                return 0
            else
                echo -e "    ${RED}✗ Failed to push to fork${NC}"
                cd - > /dev/null
                return 1
            fi
        else
            # Regular push for new PR
            echo -e "    ${YELLOW}Pushing to fork...${NC}"
            if git push -u fork "$branch_name" > /dev/null 2>&1; then
                echo -e "    ${GREEN}✓ Pushed to fork${NC}"
                
                # Prompt before creating PR
                echo ""
                echo -e "    ${BLUE}Branch ${branch_name} has been pushed to your fork.${NC}"
                if ! prompt_user "    Create pull request for $repo?"; then
                    echo -e "    ${YELLOW}⊘ Pull request creation skipped${NC}"
                    echo -e "    ${BLUE}ℹ You can manually create a PR later from:${NC}"
                    echo -e "      ${username}:${branch_name} -> ${repo}:${default_branch}"
                    cd - > /dev/null
                    return 0
                fi
                
                # Create pull request
                echo -e "    ${YELLOW}Creating pull request...${NC}"
                local pr_title="Update OCP documentation links to latest versions"
                local pr_body="This PR updates OpenShift Container Platform documentation URLs to point to the latest available versions.

## Changes
- Automatically updated OCP documentation links using [ocp-doc-checker](https://github.com/sebrandon1/ocp-doc-checker)
- All documentation URLs now point to current versions

## Tool Information
This PR was created automatically by scanning the repository with \`ocp-doc-checker\`, which identifies and updates outdated OpenShift documentation links.

Repository: https://github.com/sebrandon1/ocp-doc-checker"

                # Add tracking issue link if provided
                if [ -n "$TRACKING_LINK" ]; then
                    pr_body="${pr_body}

---

**Tracking Issue:** ${TRACKING_LINK}"
                fi
                
                if pr_url=$(gh pr create --repo "$repo" --base "$default_branch" --head "${username}:${branch_name}" --title "$pr_title" --body "$pr_body" 2>&1); then
                    echo -e "    ${GREEN}✓ Pull request created: $pr_url${NC}"
                    echo "$pr_url" >> "${OUTPUT_FILE}.prs"
                    return 0
                else
                    echo -e "    ${RED}✗ Failed to create PR: $pr_url${NC}"
                    return 1
                fi
            else
                echo -e "    ${RED}✗ Failed to push to fork${NC}"
                return 1
            fi
        fi
    else
        # No changes were made, determine why
        if [ -n "$existing_pr_num" ]; then
            # Existing PR is already up-to-date
            echo -e "    ${GREEN}✓ Existing PR #${existing_pr_num} is already up to date - no changes needed${NC}"
            local pr_url="https://github.com/$repo/pull/$existing_pr_num"
            echo -e "    ${BLUE}  $pr_url${NC}"
        elif [ $checker_exit_code -eq 0 ]; then
            echo -e "    ${GREEN}✓ No changes needed - docs are already up to date${NC}"
        else
            # Check if there were any OCP doc URLs found
            if echo "$checker_output" | grep -q "No OCP documentation URLs found"; then
                echo -e "    ${GREEN}✓ No OCP documentation URLs found in scanned files${NC}"
            elif echo "$checker_output" | grep -q "already pointing to the latest version"; then
                echo -e "    ${GREEN}✓ All OCP documentation links are already current${NC}"
            else
                echo -e "    ${YELLOW}ℹ ocp-doc-checker completed but made no changes${NC}"
                echo -e "    ${BLUE}  (docs may already be current, or issues found were not auto-fixable)${NC}"
            fi
        fi
        cd - > /dev/null
        return 0
    fi
    
    cd - > /dev/null
}

# Start timer
SCRIPT_START_TIME=$(date +%s)

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}OCP Documentation Link Scanner${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Organization: ${GREEN}${ORG_NAME}${NC}"
echo -e "Output file: ${GREEN}${OUTPUT_FILE}${NC}"
echo -e "Cache directory: ${CACHE_DIR}"
echo -e "Started: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# Initialize output file
cat > "$OUTPUT_FILE" << EOF
OpenShift Documentation Link Scanner Results
Organization: ${ORG_NAME}
Scan Date: $(date)
Started: $(date '+%Y-%m-%d %H:%M:%S')
================================================================================

EOF

# Get all repositories in the organization
echo -e "${YELLOW}Fetching repositories from ${ORG_NAME}...${NC}"
ALL_REPOS=$(gh repo list "$ORG_NAME" --limit 1000 --json nameWithOwner,isArchived --jq '.[] | select(.isArchived == false) | .nameWithOwner')

if [ -z "$ALL_REPOS" ]; then
    echo -e "${RED}Error: No repositories found or unable to access org ${ORG_NAME}${NC}"
    exit 1
fi

# Count total repos
TOTAL_REPO_COUNT=$(echo "$ALL_REPOS" | wc -l | tr -d ' ')
echo -e "${GREEN}Found ${TOTAL_REPO_COUNT} active repositories to scan${NC}"
echo ""

# Track progress
current_repo=0
repos_with_docs=()
repos_using_checker=()
repos_with_checker_prs=()
repos_not_using_checker=()
repos_with_prs_created=()
total_files_found=0
cache_hits=0
cache_misses=0
total_repo_time=0
prs_created=0
prs_failed=0

# Search each repository
while IFS= read -r repo; do
    if [ -z "$repo" ]; then
        continue
    fi
    
    current_repo=$((current_repo + 1))
    repo_start_time=$(date +%s)
    
    echo -e "${BLUE}[$current_repo/$TOTAL_REPO_COUNT]${NC} Scanning ${YELLOW}$repo${NC}..."
    
    # Get the default branch and latest commit
    repo_info=$(gh repo view "$repo" --json defaultBranchRef 2>/dev/null)
    default_branch=$(echo "$repo_info" | jq -r '.defaultBranchRef.name' 2>/dev/null || echo "main")
    latest_commit=$(echo "$repo_info" | jq -r '.defaultBranchRef.target.oid' 2>/dev/null || echo "")
    
    # Check if we have this repo cached with the same commit
    cached_commit=$(get_cached_commit "$repo")
    repo_dir=$(get_repo_dir "$repo")
    use_cache=false
    
    if [ -n "$latest_commit" ] && [ "$cached_commit" = "$latest_commit" ] && [ -d "$repo_dir" ]; then
        echo -e "    ${GREEN}✓ Using cached version (up-to-date)${NC}"
        use_cache=true
        cache_hits=$((cache_hits + 1))
    else
        # Clone or update the repository
        cache_misses=$((cache_misses + 1))
        if [ -d "$repo_dir" ]; then
            echo -e "    ${YELLOW}↻ Updating cached repository...${NC}"
            rm -rf "$repo_dir"
        else
            echo -e "    ${YELLOW}↓ Cloning repository...${NC}"
        fi
        
        if gh repo clone "$repo" "$repo_dir" -- --depth 1 --quiet 2>/dev/null; then
            # Update cache with new commit
            if [ -n "$latest_commit" ]; then
                update_cache "$repo" "$latest_commit"
            fi
        else
            echo -e "  ${YELLOW}⚠ Unable to clone repository (may be empty or inaccessible)${NC}"
            continue
        fi
    fi
    
    # Now scan the repo (whether cached or freshly cloned)
    if [ -d "$repo_dir" ]; then
        # Search for OCP documentation links in all files
        files_with_docs=$(cd "$repo_dir" && grep -r -l "$DOC_PATTERN" . 2>/dev/null | grep -v ".git" | sed 's|^\./||' || true)
        
        if [ -n "$files_with_docs" ]; then
            file_count=$(echo "$files_with_docs" | wc -l | tr -d ' ')
            echo -e "  ${GREEN}✓ Found ${file_count} file(s) with OCP documentation links${NC}"
            repos_with_docs+=("$repo")
            total_files_found=$((total_files_found + file_count))
            
            # Check if ocp-doc-checker is being used in workflows
            using_checker=false
            checker_workflows=""
            if [ -d "$repo_dir/.github/workflows" ]; then
                checker_workflows=$(cd "$repo_dir" && grep -l "ocp-doc-checker" .github/workflows/* 2>/dev/null || true)
                if [ -n "$checker_workflows" ]; then
                    using_checker=true
                    repos_using_checker+=("$repo")
                    echo -e "    ${BLUE}ℹ Using ocp-doc-checker${NC}"
                fi
            fi
            
            # Check for open PRs mentioning ocp-doc-checker
            open_prs=$(gh pr list --repo "$repo" --search "ocp-doc-checker" --state open --json number,title,url --limit 5 2>/dev/null || echo "[]")
            pr_count=$(echo "$open_prs" | jq '. | length' 2>/dev/null || echo "0")
            
            if [ "$pr_count" -gt 0 ]; then
                repos_with_checker_prs+=("$repo")
                echo -e "    ${YELLOW}⚠ $pr_count open PR(s) mentioning ocp-doc-checker${NC}"
            fi
            
            if [ "$using_checker" = false ] && [ "$pr_count" -eq 0 ]; then
                repos_not_using_checker+=("$repo")
                echo -e "    ${RED}✗ NOT using ocp-doc-checker${NC}"
            fi
            
            # Write to output file
            echo "Repository: $repo" >> "$OUTPUT_FILE"
            echo "  GitHub URL: https://github.com/$repo" >> "$OUTPUT_FILE"
            echo "  Default branch: $default_branch" >> "$OUTPUT_FILE"
            echo "  Using ocp-doc-checker: $([ "$using_checker" = true ] && echo "YES" || echo "NO")" >> "$OUTPUT_FILE"
            
            if [ -n "$checker_workflows" ]; then
                echo "  Workflows using ocp-doc-checker:" >> "$OUTPUT_FILE"
                while IFS= read -r workflow; do
                    if [ -n "$workflow" ]; then
                        workflow_name=$(basename "$workflow")
                        echo "    - $workflow_name" >> "$OUTPUT_FILE"
                    fi
                done <<< "$checker_workflows"
            fi
            
            if [ "$pr_count" -gt 0 ]; then
                echo "  Open PRs mentioning ocp-doc-checker:" >> "$OUTPUT_FILE"
                echo "$open_prs" | jq -r '.[] | "    - #\(.number): \(.title)\n      \(.url)"' >> "$OUTPUT_FILE"
            fi
            
            echo "  Files with OCP documentation links:" >> "$OUTPUT_FILE"
            
            # List all files found with their URLs
            while IFS= read -r file; do
                if [ -n "$file" ]; then
                    echo "    - $file" >> "$OUTPUT_FILE"
                    
                    # Count occurrences in this file
                    occurrence_count=$(cd "$repo_dir" && grep -c "$DOC_PATTERN" "$file" 2>/dev/null || echo "0")
                    echo "      Occurrences: $occurrence_count" >> "$OUTPUT_FILE"
                    
                    # Get a few sample URLs from the file
                    sample_urls=$(cd "$repo_dir" && grep -o "https://[^[:space:]\"']*docs\.redhat\.com/en/documentation/openshift_container_platform[^[:space:]\"']*" "$file" 2>/dev/null | head -3 || true)
                    if [ -n "$sample_urls" ]; then
                        echo "      Sample URLs:" >> "$OUTPUT_FILE"
                        while IFS= read -r url; do
                            if [ -n "$url" ]; then
                                echo "        - $url" >> "$OUTPUT_FILE"
                            fi
                        done <<< "$sample_urls"
                    fi
                    
                    # GitHub URL to the file
                    github_url="https://github.com/${repo}/blob/${default_branch}/${file}"
                    echo "      GitHub: $github_url" >> "$OUTPUT_FILE"
                    echo "" >> "$OUTPUT_FILE"
                fi
            done <<< "$files_with_docs"
            
            echo "" >> "$OUTPUT_FILE"
            
            # Run fix mode if enabled
            if [ "$FIX_MODE" = true ]; then
                echo -e "  ${BLUE}🔧 Fix mode enabled${NC}"
                
                # Count PRs before running fix
                prs_before=0
                if [ -f "${OUTPUT_FILE}.prs" ]; then
                    prs_before=$(wc -l < "${OUTPUT_FILE}.prs" 2>/dev/null | tr -d ' ')
                fi
                
                if fix_and_create_pr "$repo" "$repo_dir" "$default_branch"; then
                    # Check if a new PR was actually added
                    prs_after=0
                    if [ -f "${OUTPUT_FILE}.prs" ]; then
                        prs_after=$(wc -l < "${OUTPUT_FILE}.prs" 2>/dev/null | tr -d ' ')
                    fi
                    
                    if [ $prs_after -gt $prs_before ]; then
                        repos_with_prs_created+=("$repo")
                        prs_created=$((prs_created + 1))
                    fi
                else
                    prs_failed=$((prs_failed + 1))
                fi
            fi
        fi
    fi
    
    # Calculate time for this repo
    repo_end_time=$(date +%s)
    repo_duration=$((repo_end_time - repo_start_time))
    total_repo_time=$((total_repo_time + repo_duration))
    
    # Show timing if it took more than 1 second
    if [ $repo_duration -gt 1 ]; then
        echo -e "    ${BLUE}⏱ Processed in ${repo_duration}s${NC}"
    fi
done <<< "$ALL_REPOS"

# Calculate final timing
SCRIPT_END_TIME=$(date +%s)
TOTAL_DURATION=$((SCRIPT_END_TIME - SCRIPT_START_TIME))
MINUTES=$((TOTAL_DURATION / 60))
SECONDS=$((TOTAL_DURATION % 60))

# Calculate average time per repo
if [ $TOTAL_REPO_COUNT -gt 0 ]; then
    AVG_TIME_PER_REPO=$(awk "BEGIN {printf \"%.2f\", $TOTAL_DURATION / $TOTAL_REPO_COUNT}")
else
    AVG_TIME_PER_REPO="0.00"
fi

# Update total counts
total_repos_with_docs=${#repos_with_docs[@]}
total_using_checker=${#repos_using_checker[@]}
total_with_prs=${#repos_with_checker_prs[@]}
total_not_using=${#repos_not_using_checker[@]}

# Write summary to output file
cat >> "$OUTPUT_FILE" << EOF

================================================================================
SUMMARY
================================================================================
Total repositories scanned: ${TOTAL_REPO_COUNT}
Total repositories with OCP documentation links: ${total_repos_with_docs}
Total files with OCP documentation links: ${total_files_found}

Performance Metrics:
  - Total duration: ${MINUTES}m ${SECONDS}s (${TOTAL_DURATION} seconds)
  - Average time per repository: ${AVG_TIME_PER_REPO}s
  - Completed: $(date '+%Y-%m-%d %H:%M:%S')

Cache Performance:
  - Cache hits (repos already cached): ${cache_hits}
  - Cache misses (cloned/updated): ${cache_misses}
  - Cache hit rate: $(awk "BEGIN {printf \"%.1f\", ($cache_hits / $TOTAL_REPO_COUNT) * 100}")%

ocp-doc-checker Status:
  - Already using ocp-doc-checker: ${total_using_checker}
  - Open PRs for ocp-doc-checker: ${total_with_prs}
  - NOT using ocp-doc-checker: ${total_not_using}

EOF

if [ "$FIX_MODE" = true ]; then
    cat >> "$OUTPUT_FILE" << EOF
Fix Mode Results:
  - Pull requests created: ${prs_created}
  - Failed to create PR: ${prs_failed}
  - No changes needed: $((total_repos_with_docs - prs_created - prs_failed))

EOF
    
    if [ $prs_created -gt 0 ] && [ -f "${OUTPUT_FILE}.prs" ]; then
        echo "Pull Requests Created:" >> "$OUTPUT_FILE"
        while IFS= read -r pr_url; do
            if [ -n "$pr_url" ]; then
                echo "  - $pr_url" >> "$OUTPUT_FILE"
            fi
        done < "${OUTPUT_FILE}.prs"
        echo "" >> "$OUTPUT_FILE"
    fi
fi

if [ $total_repos_with_docs -gt 0 ]; then
    echo "All Repositories with OCP documentation links:" >> "$OUTPUT_FILE"
    for repo in "${repos_with_docs[@]}"; do
        echo "  - $repo" >> "$OUTPUT_FILE"
        echo "    https://github.com/$repo" >> "$OUTPUT_FILE"
    done
    echo "" >> "$OUTPUT_FILE"
fi

if [ $total_using_checker -gt 0 ]; then
    echo "Repositories already using ocp-doc-checker:" >> "$OUTPUT_FILE"
    for repo in "${repos_using_checker[@]}"; do
        echo "  - $repo" >> "$OUTPUT_FILE"
        echo "    https://github.com/$repo" >> "$OUTPUT_FILE"
    done
    echo "" >> "$OUTPUT_FILE"
fi

if [ $total_with_prs -gt 0 ]; then
    echo "Repositories with open PRs for ocp-doc-checker:" >> "$OUTPUT_FILE"
    for repo in "${repos_with_checker_prs[@]}"; do
        echo "  - $repo" >> "$OUTPUT_FILE"
        echo "    https://github.com/$repo" >> "$OUTPUT_FILE"
    done
    echo "" >> "$OUTPUT_FILE"
fi

if [ $total_not_using -gt 0 ]; then
    echo "⚠ Repositories NOT using ocp-doc-checker (action needed):" >> "$OUTPUT_FILE"
    for repo in "${repos_not_using_checker[@]}"; do
        echo "  - $repo" >> "$OUTPUT_FILE"
        echo "    https://github.com/$repo" >> "$OUTPUT_FILE"
    done
    echo "" >> "$OUTPUT_FILE"
fi

# Calculate cache hit rate
cache_hit_rate=$(awk "BEGIN {printf \"%.1f\", ($cache_hits / $TOTAL_REPO_COUNT) * 100}")

# Print summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Scan Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Total repositories scanned: ${GREEN}${TOTAL_REPO_COUNT}${NC}"
echo -e "Repositories with OCP docs: ${GREEN}${total_repos_with_docs}${NC}"
echo -e "Total files found: ${GREEN}${total_files_found}${NC}"
echo ""
echo -e "${YELLOW}⏱ Performance Metrics:${NC}"
echo -e "  Total duration: ${GREEN}${MINUTES}m ${SECONDS}s${NC} (${TOTAL_DURATION}s)"
echo -e "  Average per repository: ${GREEN}${AVG_TIME_PER_REPO}s${NC}"
echo -e "  Completed: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""
echo -e "${YELLOW}Cache Performance:${NC}"
echo -e "  Cache hits: ${GREEN}${cache_hits}${NC}"
echo -e "  Cache misses: ${YELLOW}${cache_misses}${NC}"
echo -e "  Hit rate: ${GREEN}${cache_hit_rate}%${NC}"
echo ""
echo -e "${YELLOW}ocp-doc-checker Status:${NC}"
echo -e "  Already using: ${GREEN}${total_using_checker}${NC}"
echo -e "  Open PRs: ${YELLOW}${total_with_prs}${NC}"
echo -e "  NOT using: ${RED}${total_not_using}${NC}"
echo ""

if [ "$FIX_MODE" = true ]; then
    echo -e "${YELLOW}🔧 Fix Mode Results:${NC}"
    echo -e "  PRs created: ${GREEN}${prs_created}${NC}"
    echo -e "  Failed: ${RED}${prs_failed}${NC}"
    echo -e "  No changes needed: ${GREEN}$((total_repos_with_docs - prs_created - prs_failed))${NC}"
    echo ""
    
    if [ $prs_created -gt 0 ]; then
        echo -e "${GREEN}✓ Created $prs_created pull request(s)${NC}"
        if [ -f "${OUTPUT_FILE}.prs" ]; then
            echo -e "${YELLOW}Pull requests:${NC}"
            while IFS= read -r pr_url; do
                if [ -n "$pr_url" ]; then
                    echo -e "  ${BLUE}$pr_url${NC}"
                fi
            done < "${OUTPUT_FILE}.prs"
        fi
        echo ""
    fi
fi

echo -e "Results saved to: ${GREEN}${OUTPUT_FILE}${NC}"
echo ""

if [ $total_repos_with_docs -gt 0 ]; then
    echo -e "${YELLOW}All Repositories with OCP documentation links:${NC}"
    for repo in "${repos_with_docs[@]}"; do
        echo -e "  ${GREEN}•${NC} $repo"
        echo -e "    ${BLUE}https://github.com/$repo${NC}"
    done
    echo ""
fi

if [ $total_not_using -gt 0 ]; then
    echo -e "${RED}⚠ Repositories NOT using ocp-doc-checker (action needed):${NC}"
    for repo in "${repos_not_using_checker[@]}"; do
        echo -e "  ${RED}•${NC} $repo"
        echo -e "    ${BLUE}https://github.com/$repo${NC}"
    done
else
    echo -e "${GREEN}✓ All repositories with OCP docs are using ocp-doc-checker!${NC}"
fi

exit 0
