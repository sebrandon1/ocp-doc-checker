#!/bin/bash
# Preflight: can this runner reach docs.redhat.com?
#
# docs.redhat.com sits behind Akamai, which sometimes answers automated
# clients (including GitHub-hosted runners) with "403 Access Denied". When
# that happens every live-docs test fails for reasons unrelated to the change
# under test. This script probes a known-good page and writes
# `reachable=true|false` and `status=<code>` to $GITHUB_OUTPUT so workflows
# can skip live-docs jobs with a visible warning instead of failing.
#
# Usage: scripts/check-docs-reachable.sh [url]
# Always exits 0; read the outputs to decide.

set -u

URL="${1:-https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html-single/disconnected_environments/index}"
UA="ocp-doc-checker-ci (+https://github.com/sebrandon1/ocp-doc-checker)"

STATUS=$(curl -s -o /dev/null -w '%{http_code}' -L --max-time 30 --retry 2 --retry-delay 5 -A "$UA" "$URL" || true)
STATUS="${STATUS:-000}"

if [[ "$STATUS" =~ ^2[0-9][0-9]$ ]]; then
  REACHABLE=true
  echo "✅ docs.redhat.com reachable (HTTP $STATUS)"
else
  REACHABLE=false
  echo "::warning title=docs.redhat.com unreachable::HTTP $STATUS from $URL. Live-docs tests will be skipped on this run."
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "reachable=$REACHABLE"
    echo "status=$STATUS"
  } >> "$GITHUB_OUTPUT"
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ] && [ "$REACHABLE" = false ]; then
  {
    echo "### ⚠️ docs.redhat.com unreachable (HTTP $STATUS)"
    echo ""
    echo "Live-documentation tests were skipped. This is usually Akamai bot protection blocking the runner, not a problem with this change."
  } >> "$GITHUB_STEP_SUMMARY"
fi

exit 0
