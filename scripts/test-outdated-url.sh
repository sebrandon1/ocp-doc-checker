#!/bin/bash
set +e  # Don't exit on error - we want to see what happens

echo "=========================================="
echo "Test 1: Outdated URL Detection"
echo "=========================================="

echo ""
echo "Step 1: Running ocp-doc-checker with 4.17 URL..."
START_TIME=$(date +%s)
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full" -json 2>&1 | tee test1_output.json
EXIT_CODE=${PIPESTATUS[0]}
END_TIME=$(date +%s)
echo ""
echo "Command completed in $((END_TIME - START_TIME)) seconds"
echo "Exit code: $EXIT_CODE"

echo ""
echo "Step 2: Verifying output file..."
if [ -f test1_output.json ]; then
  echo "File exists, size: $(wc -c < test1_output.json) bytes"
  echo "Content:"
  cat test1_output.json
else
  echo "ERROR: test1_output.json does not exist!"
  ls -la test1*.json || echo "No test1 files found"
fi
echo ""

echo "Step 3: Validating exit code..."
if [ $EXIT_CODE -ne 1 ]; then
  echo "❌ FAIL: Expected exit code 1 (outdated), got $EXIT_CODE"
  exit 1
fi
echo "✓ Exit code is 1 as expected"

echo ""
echo "Step 4: Checking is_outdated flag..."
IS_OUTDATED=$(jq -r '.is_outdated' test1_output.json)
echo "is_outdated: $IS_OUTDATED"
if [ "$IS_OUTDATED" != "true" ]; then
  echo "❌ FAIL: Expected is_outdated=true, got $IS_OUTDATED"
  exit 1
fi
echo "✓ is_outdated is true"

echo ""
echo "Step 5: Checking newer versions count..."
NEWER_COUNT=$(jq '.newer_versions | length' test1_output.json)
echo "Number of newer versions found: $NEWER_COUNT"
if [ $NEWER_COUNT -lt 1 ]; then
  echo "❌ FAIL: Expected at least 1 newer version, got $NEWER_COUNT"
  exit 1
fi
echo "✓ Found $NEWER_COUNT newer version(s)"

echo ""
echo "Step 6: Verifying 4.19 and 4.20 are in newer versions..."
echo "All newer versions:"
jq -r '.newer_versions[] | "\(.version): \(.url)"' test1_output.json

HAS_419=$(jq -r '.newer_versions[] | select(.version=="4.19") | .version' test1_output.json)
echo ""
echo "4.19 version found: $HAS_419"
if [ "$HAS_419" != "4.19" ]; then
  echo "❌ FAIL: Expected version 4.19 in newer versions"
  exit 1
fi
echo "✓ Version 4.19 found"

HAS_420=$(jq -r '.newer_versions[] | select(.version=="4.20") | .version' test1_output.json)
echo ""
echo "4.20 version found: $HAS_420"
if [ "$HAS_420" != "4.20" ]; then
  echo "❌ FAIL: Expected version 4.20 in newer versions"
  exit 1
fi
echo "✓ Version 4.20 found"

echo ""
echo "=========================================="
echo "✅ PASS: Test 1 passed - outdated URL correctly detected"
echo "=========================================="

