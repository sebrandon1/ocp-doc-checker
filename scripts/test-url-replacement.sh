#!/bin/bash
set -e

echo "Testing specific URL replacement expectation..."

INPUT_URL="https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
EXPECTED_URL="https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-full"

# Run checker (will exit 1 for outdated, but that's expected)
./ocp-doc-checker -url "$INPUT_URL" -json > test5_output.json || true

# Find the 4.19 version URL
ACTUAL_URL=$(jq -r '.newer_versions[] | select(.version=="4.19") | .url' test5_output.json)

if [ "$ACTUAL_URL" != "$EXPECTED_URL" ]; then
  echo "❌ FAIL: URL replacement mismatch"
  echo "  Expected: $EXPECTED_URL"
  echo "  Actual:   $ACTUAL_URL"
  exit 1
fi

echo "✅ PASS: Test 5 passed - correct URL replacement"

