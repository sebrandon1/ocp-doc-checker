#!/bin/bash

# This script generates a test summary for GitHub Action tests
# It expects test outcomes and outputs to be passed as arguments
# Usage: ./generate-action-test-summary.sh OUTDATED_OUTCOME OUTDATED_IS_OUTDATED CURRENT_OUTCOME CURRENT_IS_OUTDATED
# If no arguments are provided, defaults to passing values for local testing

OUTDATED_OUTCOME=${1:-"failure"}
OUTDATED_IS_OUTDATED=${2:-"true"}
CURRENT_OUTCOME=${3:-"success"}
CURRENT_IS_OUTDATED=${4:-"false"}

# If running locally (no GitHub Actions), write to stdout instead
OUTPUT_FILE=${GITHUB_STEP_SUMMARY:-/dev/stdout}

echo "## GitHub Action Test Results" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

if [ "$OUTDATED_OUTCOME" == "failure" ] && [ "$OUTDATED_IS_OUTDATED" == "true" ]; then
  echo "✅ Outdated URL detection - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Outdated URL detection - FAILED" >> $OUTPUT_FILE
fi

if [ "$CURRENT_OUTCOME" == "success" ] && [ "$CURRENT_IS_OUTDATED" == "false" ]; then
  echo "✅ Current URL validation - PASSED" >> $OUTPUT_FILE
else
  echo "❌ Current URL validation - FAILED" >> $OUTPUT_FILE
fi

