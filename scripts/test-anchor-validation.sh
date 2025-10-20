#!/bin/bash
# Integration test for anchor validation with real Red Hat documentation URLs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Anchor Validation Integration Test"
echo "=========================================="
echo ""
echo "Testing with real Red Hat documentation URLs..."
echo ""

# Build if needed
if [ ! -f "./ocp-doc-checker" ]; then
    echo -e "${YELLOW}Building ocp-doc-checker...${NC}"
    go build -o ocp-doc-checker .
fi

# Test 1: SR-IOV URL - anchor exists in 4.17 but NOT in 4.18/4.19
echo "=========================================="
echo "Test 1: Missing Anchor Detection (SR-IOV)"
echo "=========================================="
echo ""
echo "URL: https://docs.redhat.com/.../4.17/html-single/networking/index#installing-sr-iov-operator_installing-sriov-operator"
echo "Expected: Tool should detect that anchor is missing in 4.18 and 4.19"
echo ""

URL="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/networking/index#installing-sr-iov-operator_installing-sriov-operator"

set +e
OUTPUT=$(./ocp-doc-checker -url "$URL" -json 2>&1)
EXIT_CODE=$?
set -e

echo "Output:"
echo "$OUTPUT" | jq . || echo "$OUTPUT"
echo ""

# Parse JSON output
IS_OUTDATED=$(echo "$OUTPUT" | jq -r '.is_outdated')
NEWER_COUNT=$(echo "$OUTPUT" | jq '.newer_versions | length')

echo "Results:"
echo "  - is_outdated: $IS_OUTDATED"
echo "  - newer_versions count: $NEWER_COUNT"
echo "  - exit_code: $EXIT_CODE"
echo ""

# Expected: is_outdated should be FALSE because no newer versions have the anchor
if [ "$IS_OUTDATED" != "false" ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected is_outdated=false (no valid newer versions)"
    echo "The anchor doesn't exist in 4.18/4.19, so tool should NOT suggest upgrade"
    exit 1
fi

# Expected: newer_versions should be 0 because anchor doesn't exist in newer versions
if [ "$NEWER_COUNT" != "0" ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected 0 newer versions (anchor missing in 4.18/4.19)"
    echo "Got: $NEWER_COUNT newer versions"
    exit 1
fi

# Expected: Exit code 0 (not outdated)
if [ $EXIT_CODE -ne 0 ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected exit code 0"
    exit 1
fi

echo -e "${GREEN}✅ PASS${NC}: Tool correctly detected missing anchors in newer versions"
echo ""

# Test 2: Verify with verbose output to see anchor status
echo "=========================================="
echo "Test 2: Verbose Anchor Status"
echo "=========================================="
echo ""
echo "Running with --verbose to see detailed anchor status..."
echo ""

set +e
VERBOSE_OUTPUT=$(./ocp-doc-checker -url "$URL" --verbose 2>&1)
set -e

echo "$VERBOSE_OUTPUT"
echo ""

# Check for anchor missing message
if echo "$VERBOSE_OUTPUT" | grep -q "anchor missing"; then
    echo -e "${GREEN}✅ PASS${NC}: Verbose output shows anchor missing status"
else
    echo -e "${RED}❌ FAIL${NC}: Expected 'anchor missing' in verbose output"
    exit 1
fi

echo ""

# Test 3: URL with anchor that exists in all versions (disconnected_environments)
echo "=========================================="
echo "Test 3: Valid Anchor in All Versions"
echo "=========================================="
echo ""
echo "URL: https://docs.redhat.com/.../4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
echo "Expected: Tool should find newer versions (anchor exists in 4.18/4.19)"
echo ""

URL2="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"

set +e
OUTPUT2=$(./ocp-doc-checker -url "$URL2" -json 2>&1)
EXIT_CODE2=$?
set -e

echo "Output:"
echo "$OUTPUT2" | jq . || echo "$OUTPUT2"
echo ""

IS_OUTDATED2=$(echo "$OUTPUT2" | jq -r '.is_outdated')
NEWER_COUNT2=$(echo "$OUTPUT2" | jq '.newer_versions | length')

echo "Results:"
echo "  - is_outdated: $IS_OUTDATED2"
echo "  - newer_versions count: $NEWER_COUNT2"
echo "  - exit_code: $EXIT_CODE2"
echo ""

# Expected: is_outdated should be TRUE because anchor exists in newer versions
if [ "$IS_OUTDATED2" != "true" ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected is_outdated=true (anchor exists in newer versions)"
    exit 1
fi

# Expected: newer_versions should be > 0
if [ "$NEWER_COUNT2" -lt 1 ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected at least 1 newer version"
    exit 1
fi

# Expected: Exit code 1 (outdated)
if [ $EXIT_CODE2 -ne 1 ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected exit code 1"
    exit 1
fi

echo -e "${GREEN}✅ PASS${NC}: Tool correctly found newer versions with valid anchors"
echo ""

# Test 4: URL without anchor (should still work)
echo "=========================================="
echo "Test 4: URL Without Anchor"
echo "=========================================="
echo ""
echo "URL: https://docs.redhat.com/.../4.17/html-single/disconnected_environments/index"
echo "Expected: Tool should find newer versions (no anchor to validate)"
echo ""

URL3="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index"

set +e
OUTPUT3=$(./ocp-doc-checker -url "$URL3" -json 2>&1)
EXIT_CODE3=$?
set -e

echo "Output:"
echo "$OUTPUT3" | jq . || echo "$OUTPUT3"
echo ""

IS_OUTDATED3=$(echo "$OUTPUT3" | jq -r '.is_outdated')
NEWER_COUNT3=$(echo "$OUTPUT3" | jq '.newer_versions | length')

echo "Results:"
echo "  - is_outdated: $IS_OUTDATED3"
echo "  - newer_versions count: $NEWER_COUNT3"
echo ""

# Expected: Should work normally (no anchor to validate)
if [ "$IS_OUTDATED3" != "true" ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected is_outdated=true"
    exit 1
fi

if [ "$NEWER_COUNT3" -lt 1 ]; then
    echo -e "${RED}❌ FAIL${NC}: Expected at least 1 newer version"
    exit 1
fi

echo -e "${GREEN}✅ PASS${NC}: Tool works correctly without anchor"
echo ""

# Summary
echo "=========================================="
echo "Anchor Validation Test Summary"
echo "=========================================="
echo ""
echo -e "${GREEN}✅ All anchor validation tests passed!${NC}"
echo ""
echo "Key findings:"
echo "  ✓ Detects when anchors are missing in newer versions (SR-IOV case)"
echo "  ✓ Still suggests upgrades when anchors exist in newer versions"
echo "  ✓ Works correctly for URLs without anchors"
echo "  ✓ Prevents false positives from reorganized content"
echo ""

