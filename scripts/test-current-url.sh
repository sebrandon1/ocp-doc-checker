#!/bin/bash
set -e

echo "Testing current 4.19 URL (expect success/exit code 0)..."
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-partial" -json > test2_output.json
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

# Check latest version is 4.19
LATEST=$(jq -r '.latest_version' test2_output.json)
if [ "$LATEST" != "4.19" ]; then
  echo "❌ FAIL: Expected latest_version=4.19, got $LATEST"
  exit 1
fi

echo "✅ PASS: Test 2 passed - current URL correctly identified"

