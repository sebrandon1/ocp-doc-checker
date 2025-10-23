#!/bin/bash

# This script verifies that the GitHub Action correctly detected current documentation
# It expects the is-outdated and latest-version outputs as arguments
# Usage: ./verify-action-current.sh IS_OUTDATED_VALUE LATEST_VERSION_VALUE
# If no arguments are provided, defaults to "false" and "4.20" for local testing

IS_OUTDATED=${1:-"false"}
LATEST_VERSION=${2:-"4.20"}

echo "✅ Action correctly passed for current documentation"
echo "Is Outdated: $IS_OUTDATED"
echo "Latest Version: $LATEST_VERSION"

# Verify outputs
if [ "$IS_OUTDATED" != "false" ]; then
  echo "❌ FAIL: Expected is-outdated=false"
  exit 1
fi

if [ "$LATEST_VERSION" != "4.20" ]; then
  echo "❌ FAIL: Expected latest-version=4.20"
  exit 1
fi

echo "✅ PASS: Current detection verified"

