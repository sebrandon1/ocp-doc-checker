#!/bin/bash
# Test script for ocp-doc-checker with expected/actual assertions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_func="$2"
    
    echo ""
    echo "=========================================="
    echo "Running: $test_name"
    echo "=========================================="
    
    if $test_func; then
        echo -e "${GREEN}✅ PASS${NC}: $test_name"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    else
        echo -e "${RED}❌ FAIL${NC}: $test_name"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
}

# Test 1: Outdated URL should be detected
test_outdated_url() {
    echo "Testing outdated 4.17 URL..."
    
    local url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" -json 2>&1)
    EXIT_CODE=$?
    set -e
    
    echo "$OUTPUT" | jq . || echo "$OUTPUT"
    
    # Expected: Exit code 1 (outdated)
    if [ $EXIT_CODE -ne 1 ]; then
        echo "Expected exit code 1, got $EXIT_CODE"
        return 1
    fi
    
    # Expected: is_outdated = true
    IS_OUTDATED=$(echo "$OUTPUT" | jq -r '.is_outdated')
    if [ "$IS_OUTDATED" != "true" ]; then
        echo "Expected is_outdated=true, got $IS_OUTDATED"
        return 1
    fi
    
    # Expected: At least one newer version
    NEWER_COUNT=$(echo "$OUTPUT" | jq '.newer_versions | length')
    if [ $NEWER_COUNT -lt 1 ]; then
        echo "Expected at least 1 newer version, got $NEWER_COUNT"
        return 1
    fi
    
    echo "Newer versions found: $NEWER_COUNT"
    return 0
}

# Test 2: Current URL should pass
test_current_url() {
    echo "Testing current 4.19 URL..."
    
    local url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-partial"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" -json 2>&1)
    EXIT_CODE=$?
    set -e
    
    echo "$OUTPUT" | jq . || echo "$OUTPUT"
    
    # Expected: Exit code 0 (up to date)
    if [ $EXIT_CODE -ne 0 ]; then
        echo "Expected exit code 0, got $EXIT_CODE"
        return 1
    fi
    
    # Expected: is_outdated = false
    IS_OUTDATED=$(echo "$OUTPUT" | jq -r '.is_outdated')
    if [ "$IS_OUTDATED" != "false" ]; then
        echo "Expected is_outdated=false, got $IS_OUTDATED"
        return 1
    fi
    
    # Expected: latest_version = 4.19
    LATEST=$(echo "$OUTPUT" | jq -r '.latest_version')
    if [ "$LATEST" != "4.19" ]; then
        echo "Expected latest_version=4.19, got $LATEST"
        return 1
    fi
    
    return 0
}

# Test 3: Verify specific URL replacement
test_url_replacement() {
    echo "Testing specific URL replacement..."
    
    local input_url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
    local expected_url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-full"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$input_url" -json 2>&1)
    set -e
    
    # Extract 4.19 URL
    ACTUAL_URL=$(echo "$OUTPUT" | jq -r '.newer_versions[] | select(.version=="4.19") | .url')
    
    if [ "$ACTUAL_URL" != "$expected_url" ]; then
        echo "URL replacement mismatch:"
        echo "  Expected: $expected_url"
        echo "  Actual:   $ACTUAL_URL"
        return 1
    fi
    
    echo "Correct URL replacement: $ACTUAL_URL"
    return 0
}

# Test 4: Invalid URL should fail gracefully
test_invalid_url() {
    echo "Testing invalid URL..."
    
    local url="https://example.com/invalid"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" 2>&1)
    EXIT_CODE=$?
    set -e
    
    echo "$OUTPUT"
    
    # Expected: Non-zero exit code
    if [ $EXIT_CODE -eq 0 ]; then
        echo "Expected non-zero exit code for invalid URL"
        return 1
    fi
    
    # Expected: Error message
    if ! echo "$OUTPUT" | grep -qi "error"; then
        echo "Expected error message in output"
        return 1
    fi
    
    return 0
}

# Test 5: Anchor preservation
test_anchor_preservation() {
    echo "Testing anchor preservation..."
    
    local url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
    local expected_anchor="mirroring-image-set-full"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" -json 2>&1)
    set -e
    
    # Check that all newer version URLs contain the anchor
    URLS=$(echo "$OUTPUT" | jq -r '.newer_versions[].url')
    
    while IFS= read -r check_url; do
        if [ -n "$check_url" ] && ! echo "$check_url" | grep -q "#$expected_anchor"; then
            echo "Anchor not preserved in: $check_url"
            return 1
        fi
    done <<< "$URLS"
    
    echo "Anchor preserved in all newer version URLs"
    return 0
}

# Test 6: Version comparison logic
test_version_comparison() {
    echo "Testing version comparison..."
    
    # 4.15 should find multiple newer versions
    local url="https://docs.redhat.com/en/documentation/openshift_container_platform/4.15/html-single/disconnected_environments/index"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" -json 2>&1)
    set -e
    
    # Count how many newer versions are found
    NEWER_COUNT=$(echo "$OUTPUT" | jq '.newer_versions | length')
    
    # Should find at least 2 newer versions
    if [ $NEWER_COUNT -lt 2 ]; then
        echo "Expected at least 2 newer versions from 4.15, got $NEWER_COUNT"
        return 1
    fi
    
    # Verify versions are sorted (latest should be higher than first)
    FIRST_VERSION=$(echo "$OUTPUT" | jq -r '.newer_versions[0].version')
    LAST_VERSION=$(echo "$OUTPUT" | jq -r '.newer_versions[-1].version')
    
    echo "Found $NEWER_COUNT newer versions from 4.15 (${FIRST_VERSION} to ${LAST_VERSION})"
    return 0
}

# Main execution
main() {
    echo "======================================"
    echo "OCP Documentation Checker Test Suite"
    echo "======================================"
    
    # Check if binary exists
    if [ ! -f "./ocp-doc-checker" ]; then
        echo -e "${YELLOW}Building ocp-doc-checker...${NC}"
        go build -o ocp-doc-checker .
    fi
    
    # Check for jq
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}Error: jq is required but not installed${NC}"
        exit 1
    fi
    
    # Run all tests
    run_test "Test 1: Outdated URL Detection" test_outdated_url || true
    run_test "Test 2: Current URL Validation" test_current_url || true
    run_test "Test 3: URL Replacement" test_url_replacement || true
    run_test "Test 4: Invalid URL Handling" test_invalid_url || true
    run_test "Test 5: Anchor Preservation" test_anchor_preservation || true
    run_test "Test 6: Version Comparison" test_version_comparison || true
    
    # Summary
    echo ""
    echo "=========================================="
    echo "Test Summary"
    echo "=========================================="
    TOTAL=$((PASS_COUNT + FAIL_COUNT))
    echo -e "${GREEN}Passed: $PASS_COUNT${NC}"
    echo -e "${RED}Failed: $FAIL_COUNT${NC}"
    echo "Total:  $TOTAL"
    
    if [ $FAIL_COUNT -eq 0 ]; then
        echo ""
        echo -e "${GREEN}🎉 All tests passed!${NC}"
        exit 0
    else
        echo ""
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

main

