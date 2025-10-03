#!/bin/bash

# This script verifies that the GitHub Action correctly detected outdated documentation
# It expects the is-outdated output as an argument
# Usage: ./verify-action-outdated.sh IS_OUTDATED_VALUE
# If no argument is provided, defaults to "true" for local testing

IS_OUTDATED=${1:-"true"}

echo "✅ Action correctly failed for outdated documentation"
echo "Is Outdated: $IS_OUTDATED"

# Verify outputs
if [ "$IS_OUTDATED" != "true" ]; then
  echo "❌ FAIL: Expected is-outdated=true"
  exit 1
fi

echo "✅ PASS: Outdated detection verified"

