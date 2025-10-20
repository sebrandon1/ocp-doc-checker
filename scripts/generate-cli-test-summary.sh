#!/bin/bash

# This script generates a test summary for GitHub Actions
# It expects test outcomes to be passed as arguments
# Usage: ./generate-cli-test-summary.sh TEST1_OUTCOME TEST2_OUTCOME TEST3_OUTCOME TEST4_OUTCOME TEST5_OUTCOME TEST6_OUTCOME
# If no arguments are provided, defaults to all "success" for local testing

TEST1_OUTCOME=${1:-"success"}
TEST2_OUTCOME=${2:-"success"}
TEST3_OUTCOME=${3:-"success"}
TEST4_OUTCOME=${4:-"success"}
TEST5_OUTCOME=${5:-"success"}
TEST6_OUTCOME=${6:-"success"}

# If running locally (no GitHub Actions), write to stdout instead
OUTPUT_FILE=${GITHUB_STEP_SUMMARY:-/dev/stdout}

echo "## Test Results Summary" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

if [ "$TEST1_OUTCOME" == "success" ]; then
  echo "✅ Test 1: Outdated URL detection - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 1: Outdated URL detection - FAILED" >> $OUTPUT_FILE
fi

if [ "$TEST2_OUTCOME" == "success" ]; then
  echo "✅ Test 2: Current URL validation - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 2: Current URL validation - FAILED" >> $OUTPUT_FILE
fi

if [ "$TEST3_OUTCOME" == "success" ]; then
  echo "✅ Test 3: Suggested URLs accessibility - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 3: Suggested URLs accessibility - FAILED" >> $OUTPUT_FILE
fi

if [ "$TEST4_OUTCOME" == "success" ]; then
  echo "✅ Test 4: Invalid URL handling - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 4: Invalid URL handling - FAILED" >> $OUTPUT_FILE
fi

if [ "$TEST5_OUTCOME" == "success" ]; then
  echo "✅ Test 5: Specific URL replacement - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 5: Specific URL replacement - FAILED" >> $OUTPUT_FILE
fi

if [ "$TEST6_OUTCOME" == "success" ]; then
  echo "✅ Test 6: Anchor validation with real docs - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Test 6: Anchor validation with real docs - FAILED" >> $OUTPUT_FILE
fi

