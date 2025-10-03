#!/bin/bash
set +e  # Don't exit on error - we're testing error handling

echo "Testing invalid URL (expect graceful error)..."
./ocp-doc-checker -url "https://example.com/invalid" 2>&1 | tee test4_output.txt
EXIT_CODE=${PIPESTATUS[0]}

echo "Exit code was: $EXIT_CODE"

# Should exit with error
if [ $EXIT_CODE -eq 0 ]; then
  echo "❌ FAIL: Expected non-zero exit code for invalid URL"
  exit 1
fi

# Should contain error message
if ! grep -q -i "error" test4_output.txt; then
  echo "❌ FAIL: Expected error message in output"
  exit 1
fi

echo "✅ PASS: Test 4 passed - invalid URL handled gracefully"

