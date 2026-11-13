#!/bin/bash
set -e

echo "Testing current 5.0 URL (expect success/exit code 0)..."
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/5.0/html/release_notes/index" -json > test2_output.json
EXIT_CODE=$?

cat test2_output.json

# Should exit with code 0 (up to date)
if [ $EXIT_CODE -ne 0 ]; then
  echo "❌ FAIL: Expected exit code 0, got $EXIT_CODE"
  exit 1
fi

# Check if it's marked as NOT outdated
IS_OUTDATED=$(jq -r '.is_outdated' test2_output.json)
if [ "$IS_OUTDATED" != "false" ]; then
  echo "❌ FAIL: Expected is_outdated=false, got $IS_OUTDATED"
  exit 1
fi

# Check latest version is 5.0
LATEST=$(jq -r '.latest_version' test2_output.json)
if [ "$LATEST" != "5.0" ]; then
  echo "❌ FAIL: Expected latest_version=5.0, got $LATEST"
  exit 1
fi

echo "✅ PASS: Test 2 passed - current URL correctly identified"

