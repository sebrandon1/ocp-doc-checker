#!/bin/bash
set +e  # Don't exit on error - we want to see what happens

echo "=========================================="
echo "Test 3: Verifying suggested URLs"
echo "=========================================="

echo ""
echo "Step 1: Running ocp-doc-checker..."
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full" -json 2>&1 | tee test3_output.json
EXIT_CODE=${PIPESTATUS[0]}
echo "Exit code: $EXIT_CODE"

echo ""
echo "Step 2: Displaying JSON output..."
if [ -f test3_output.json ]; then
  echo "File exists, size: $(wc -c < test3_output.json) bytes"
  cat test3_output.json
else
  echo "ERROR: test3_output.json does not exist!"
  ls -la test3*.json || echo "No test3 files found"
fi
echo ""

echo "Step 3: Parsing JSON for newer versions..."
if [ -f test3_output.json ]; then
  jq -r '.newer_versions' test3_output.json || echo "jq parse failed"
fi

# Extract suggested URLs
SUGGESTED_URLS=$(jq -r '.newer_versions[].url' test3_output.json 2>&1)
JQ_EXIT=$?

echo ""
echo "Step 4: Extracted URLs (jq exit code: $JQ_EXIT):"
echo "$SUGGESTED_URLS"

if [ -z "$SUGGESTED_URLS" ] || [ $JQ_EXIT -ne 0 ]; then
  echo "❌ FAIL: No suggested URLs found or jq parse error"
  echo "JSON content was:"
  cat test3_output.json 2>&1 || echo "Could not read file"
  exit 1
fi

echo ""
echo "Step 5: Testing each URL with curl..."
# Test each suggested URL
FAIL_COUNT=0
URL_COUNT=0
while IFS= read -r url; do
  if [ -n "$url" ]; then
    URL_COUNT=$((URL_COUNT + 1))
    echo ""
    echo "[$URL_COUNT] Testing: $url"
    
    # Get full response with timing
    TIME_START=$(date +%s)
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -L --max-time 30 "$url" 2>&1)
    CURL_EXIT=$?
    TIME_END=$(date +%s)
    TIME_TAKEN=$((TIME_END - TIME_START))
    
    echo "    Curl exit code: $CURL_EXIT"
    echo "    HTTP status: $HTTP_CODE"
    echo "    Time taken: ${TIME_TAKEN}s"
    
    if [ $CURL_EXIT -ne 0 ]; then
      echo "    ✗ CURL FAILED (exit code: $CURL_EXIT)"
      FAIL_COUNT=$((FAIL_COUNT + 1))
    elif [ $HTTP_CODE -ge 200 ] && [ $HTTP_CODE -lt 400 ]; then
      echo "    ✓ SUCCESS"
    else
      echo "    ✗ BAD HTTP STATUS"
      FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
  fi
done <<< "$SUGGESTED_URLS"

echo ""
echo "=========================================="
echo "Test 3 Results Summary"
echo "=========================================="
echo "Total URLs tested: $URL_COUNT"
echo "Failed: $FAIL_COUNT"
echo "Passed: $((URL_COUNT - FAIL_COUNT))"

if [ $FAIL_COUNT -gt 0 ]; then
  echo "❌ FAIL: $FAIL_COUNT suggested URL(s) are not accessible"
  exit 1
fi

echo "✅ PASS: Test 3 passed - all suggested URLs are accessible"

