# OCP Doc Checker Scripts

This directory contains utility scripts for testing, validating, and automating the ocp-doc-checker tool.

## Table of Contents

- [Organization Scanning](#organization-scanning)
- [CLI Testing Scripts](#cli-testing-scripts)
- [GitHub Action Testing Scripts](#github-action-testing-scripts)
- [Requirements](#requirements)

---

## Organization Scanning

### `scan-org-for-ocp-docs.sh`

**Purpose:** Scan entire GitHub organizations for OpenShift documentation links, optionally fixing outdated links and creating pull requests automatically.

**Features:**
- Scans all repositories in a GitHub organization for OCP documentation links
- Caches cloned repositories for faster subsequent scans
- Optional `--fix` mode to automatically update outdated docs and create PRs
- Interactive prompts or `--force` mode for full automation
- Smart duplicate detection - skips repos with existing open PRs
- Links all PRs to a tracking issue with `--link-to`
- Generates detailed reports with statistics

**Usage:**

```bash
# Basic scan (read-only)
./scripts/scan-org-for-ocp-docs.sh openshift-kni

# Scan with custom output file
./scripts/scan-org-for-ocp-docs.sh openshift-kni my-results.txt

# Interactive fix mode - prompts before each action
./scripts/scan-org-for-ocp-docs.sh --fix openshift-kni

# Fix mode with tracking issue link
./scripts/scan-org-for-ocp-docs.sh --fix --link-to https://github.com/owner/repo/issues/18 openshift-kni

# Non-interactive automation (no prompts)
./scripts/scan-org-for-ocp-docs.sh --fix --force openshift-kni

# Full automation with tracking
./scripts/scan-org-for-ocp-docs.sh --fix --force --link-to https://github.com/owner/repo/issues/18 openshift

# View cache information
./scripts/scan-org-for-ocp-docs.sh --cache-info

# Clear the cache
./scripts/scan-org-for-ocp-docs.sh --clear-cache

# Show help
./scripts/scan-org-for-ocp-docs.sh --help
```

**Options:**
- `--fix` - Run ocp-doc-checker in fix mode and create pull requests
- `--force` - Non-interactive mode, automatically accepts all prompts (use with `--fix`)
- `--link-to URL` - Add a tracking issue link to all created PRs
- `--cache-info` - Display cache information
- `--clear-cache` - Clear the repository cache

**Interactive Prompts (when using `--fix` without `--force`):**
1. Before committing and pushing changes to each repository
2. Before creating a fork (if needed)
3. Before creating a pull request

**Requirements:**
- GitHub CLI (`gh`) installed and authenticated
- `ocp-doc-checker` binary built and available
- `jq` for JSON parsing
- Git configured with appropriate credentials

**Cache Location:**
- `~/.cache/ocp-doc-scanner/`

**Output Files:**
- `<output-file>` - Detailed scan results and statistics
- `<output-file>.prs` - List of created PR URLs (in fix mode)

---

## CLI Testing Scripts

These scripts test the core functionality of the ocp-doc-checker CLI tool.

### `test-outdated-url.sh`

**Purpose:** Test detection of outdated OCP documentation URLs.

**What it tests:**
- Checks that an outdated 4.14 URL is properly detected
- Validates the tool exits with code 1 (error) for outdated URLs
- Verifies `is_outdated` is set to `true`
- Confirms the latest version is correctly identified

**Usage:**
```bash
./scripts/test-outdated-url.sh
```

**Expected behavior:**
- Exit code: 1 (indicates outdated URL detected)
- Output: JSON with `is_outdated: true` and current latest version
- Test result: ✅ PASS

---

### `test-current-url.sh`

**Purpose:** Test validation of current OCP documentation URLs.

**What it tests:**
- Checks that a current 4.19 URL is properly validated
- Validates the tool exits with code 0 (success) for current URLs
- Verifies `is_outdated` is set to `false`
- Confirms the version matches the latest

**Usage:**
```bash
./scripts/test-current-url.sh
```

**Expected behavior:**
- Exit code: 0 (indicates current URL)
- Output: JSON with `is_outdated: false`
- Test result: ✅ PASS

---

### `test-invalid-url.sh`

**Purpose:** Test handling of invalid or malformed URLs.

**What it tests:**
- Validates error handling for non-OCP documentation URLs
- Checks appropriate error messages are returned
- Verifies the tool fails gracefully

**Usage:**
```bash
./scripts/test-invalid-url.sh
```

---

### `test-url-accessibility.sh`

**Purpose:** Test URL accessibility and network error handling.

**What it tests:**
- Validates handling of unreachable URLs
- Tests timeout and network error scenarios
- Verifies appropriate error messages

**Usage:**
```bash
./scripts/test-url-accessibility.sh
```

---

### `test-url-replacement.sh`

**Purpose:** Test URL replacement functionality in fix mode.

**What it tests:**
- Validates that outdated URLs are correctly replaced in files
- Tests the `-fix` flag functionality
- Verifies backup and rollback mechanisms

**Usage:**
```bash
./scripts/test-url-replacement.sh
```

---

### `test-anchor-validation.sh`

**Purpose:** Integration test for anchor/fragment validation with real Red Hat documentation URLs.

**What it tests:**
- Validates that the tool detects missing anchors in newer versions
- Tests the real-world SR-IOV case where content moved between documentation sections
- Verifies that anchors are properly validated when they exist
- Confirms URLs without anchors still work correctly

**Test Cases:**
1. **Missing Anchor Detection (SR-IOV case):**
   - URL: `https://docs.redhat.com/.../4.17/html-single/networking/index#installing-sr-iov-operator_installing-sriov-operator`
   - Expected: Tool detects anchor missing in 4.18/4.19, doesn't suggest upgrade
   - This is the real bug that was found in PR reviews!

2. **Valid Anchor in All Versions:**
   - URL: `https://docs.redhat.com/.../4.17/html-single/disconnected_environments/index#mirroring-image-set-full`
   - Expected: Tool finds newer versions because anchor exists in 4.18/4.19

3. **URL Without Anchor:**
   - URL: `https://docs.redhat.com/.../4.17/html-single/disconnected_environments/index`
   - Expected: Normal behavior, finds newer versions

**Usage:**
```bash
./scripts/test-anchor-validation.sh
```

**Expected behavior:**
- All 4 test cases should pass
- Demonstrates anchor validation prevents false positives
- Shows verbose output with anchor status messages

**Performance Note:**
This test is slower than other tests because it fetches and parses full HTML pages from Red Hat docs to validate anchors.

---

### `generate-cli-test-summary.sh`

**Purpose:** Generate a formatted summary of CLI test results.

**Usage:**
```bash
./scripts/generate-cli-test-summary.sh
```

**Output:**
- Formatted test summary with pass/fail indicators
- Statistics on test coverage
- Detailed results for each test case

---

## GitHub Action Testing Scripts

These scripts test and validate the GitHub Action version of ocp-doc-checker.

### `verify-action-outdated.sh`

**Purpose:** Verify that the GitHub Action correctly detects outdated documentation.

**Usage:**
```bash
# With Action outputs
./scripts/verify-action-outdated.sh "$IS_OUTDATED_OUTPUT" "$LATEST_VERSION_OUTPUT"

# Local testing with defaults
./scripts/verify-action-outdated.sh
```

**Parameters:**
1. `IS_OUTDATED_VALUE` - Expected to be "true"
2. `LATEST_VERSION_VALUE` - Expected latest version

**Expected behavior:**
- Validates `is-outdated` output is "true"
- Confirms latest version is detected correctly
- Test result: ✅ PASS

---

### `verify-action-current.sh`

**Purpose:** Verify that the GitHub Action correctly validates current documentation.

**Usage:**
```bash
# With Action outputs
./scripts/verify-action-current.sh "$IS_OUTDATED_OUTPUT" "$LATEST_VERSION_OUTPUT"

# Local testing with defaults
./scripts/verify-action-current.sh
```

**Parameters:**
1. `IS_OUTDATED_VALUE` - Expected to be "false"
2. `LATEST_VERSION_VALUE` - Expected to match current version

**Expected behavior:**
- Validates `is-outdated` output is "false"
- Confirms version matches expected
- Test result: ✅ PASS

---

### `generate-action-test-summary.sh`

**Purpose:** Generate a formatted summary of GitHub Action test results.

**Usage:**
```bash
# In GitHub Actions workflow
./scripts/generate-action-test-summary.sh "$OUTDATED_OUTCOME" "$OUTDATED_IS_OUTDATED" "$CURRENT_OUTCOME" "$CURRENT_IS_OUTDATED"

# Local testing
./scripts/generate-action-test-summary.sh
```

**Parameters:**
1. `OUTDATED_OUTCOME` - Test outcome for outdated URL test
2. `OUTDATED_IS_OUTDATED` - Boolean result from outdated test
3. `CURRENT_OUTCOME` - Test outcome for current URL test
4. `CURRENT_IS_OUTDATED` - Boolean result from current test

**Output:**
- Formatted test summary in GitHub Actions format
- Pass/fail indicators for each test
- Written to `$GITHUB_STEP_SUMMARY` or stdout

---

## Requirements

### General Requirements

All scripts require:
- Bash 4.0 or later
- Standard Unix utilities (grep, sed, awk)

### Tool-Specific Requirements

#### For `scan-org-for-ocp-docs.sh`:
```bash
# GitHub CLI
brew install gh
# or
sudo apt install gh

# Authenticate
gh auth login

# jq for JSON parsing
brew install jq
# or
sudo apt install jq

# Build ocp-doc-checker
cd /path/to/ocp-doc-checker
make build
```

#### For CLI Testing Scripts:
```bash
# ocp-doc-checker binary must be built
make build

# jq for JSON parsing
brew install jq
```

#### For GitHub Action Scripts:
- These scripts are designed to run within GitHub Actions workflows
- Can be tested locally by providing expected values as arguments

---

## Running All Tests

To run all CLI tests:

```bash
#!/bin/bash
echo "Running ocp-doc-checker test suite..."

# Build the tool first
make build

# Run all tests
./scripts/test-outdated-url.sh
./scripts/test-current-url.sh
./scripts/test-invalid-url.sh
./scripts/test-url-accessibility.sh
./scripts/test-url-replacement.sh

# Generate summary
./scripts/generate-cli-test-summary.sh

echo "All tests complete!"
```

---

## Examples

### Example 1: Scan and Fix OpenShift Repos

```bash
# Interactive mode - you'll be prompted before each action
./scripts/scan-org-for-ocp-docs.sh --fix openshift

# Review the results
cat ocp-docs-scan-results.txt
```

### Example 2: Automated Scanning with Tracking

```bash
# Create a tracking issue first (e.g., https://github.com/myorg/myrepo/issues/42)

# Run automated scan with all fixes and PRs linked to the issue
./scripts/scan-org-for-ocp-docs.sh \
  --fix \
  --force \
  --link-to https://github.com/myorg/myrepo/issues/42 \
  openshift-kni \
  scan-results-$(date +%Y%m%d).txt

# Check created PRs
cat scan-results-*.txt.prs
```

### Example 3: CI/CD Integration

```bash
# In your CI/CD pipeline
export GH_TOKEN="${GITHUB_TOKEN}"

# Run non-interactive scan
./scripts/scan-org-for-ocp-docs.sh \
  --fix \
  --force \
  --link-to "${TRACKING_ISSUE_URL}" \
  "${ORG_NAME}" \
  "results-${CI_BUILD_ID}.txt"

# Upload results as artifact
# ... artifact upload commands ...
```

---

## Troubleshooting

### Common Issues

**Issue:** "Error: GitHub CLI (gh) is not installed"
```bash
# Install GitHub CLI
brew install gh  # macOS
sudo apt install gh  # Debian/Ubuntu
```

**Issue:** "Error: Not authenticated with GitHub CLI"
```bash
gh auth login
# Follow the prompts to authenticate
```

**Issue:** "Error: ocp-doc-checker binary not found"
```bash
# Build the binary
make build

# Verify it exists
ls -l ocp-doc-checker
```

**Issue:** "Fork creation failed"
- Ensure you have permission to fork repositories
- Check your GitHub token has the necessary scopes
- Verify the repository isn't already forked

**Issue:** "PR creation failed"
- Ensure you have a fork of the repository
- Check that your branch was successfully pushed
- Verify there isn't already an open PR with the same branch name

### Cache Issues

If you encounter issues with cached repositories:

```bash
# Clear the cache
./scripts/scan-org-for-ocp-docs.sh --clear-cache

# View cache info
./scripts/scan-org-for-ocp-docs.sh --cache-info
```

### Debug Mode

For detailed debugging, modify scripts to add:

```bash
set -x  # Enable debug output
# ... rest of script ...
```

---

## Contributing

When adding new scripts to this directory:

1. Include a shebang (`#!/bin/bash`)
2. Add a comment block describing the script's purpose
3. Include usage examples in the script comments
4. Update this README.md with documentation
5. Make the script executable: `chmod +x scripts/your-script.sh`
6. Follow the existing naming conventions

---

## License

See the main [LICENSE](../LICENSE) file in the repository root.

