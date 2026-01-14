#!/bin/bash
# Test: [UC-TRP-002] List upcoming trips
# PRD: docs/prd/trip-management.md
#
# Tests filtering trips by upcoming (start date >= today)

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-002] Testing list upcoming trips"

# Setup: Create a future trip
echo "  Creating future trip..."
FUTURE_DATE=$(date -v+30d +%Y-%m-%d 2>/dev/null || date -d "+30 days" +%Y-%m-%d)
FUTURE_END=$(date -v+35d +%Y-%m-%d 2>/dev/null || date -d "+35 days" +%Y-%m-%d)
FUTURE=$(${CLI} trips create \
  --name "Future E2E Trip" \
  --purpose vacation \
  --start "$FUTURE_DATE" \
  --end "$FUTURE_END" \
  --json)
FUTURE_ID=$(echo "$FUTURE" | jq -r '.id')

# Setup: Create a past trip
echo "  Creating past trip..."
PAST_DATE=$(date -v-30d +%Y-%m-%d 2>/dev/null || date -d "-30 days" +%Y-%m-%d)
PAST_END=$(date -v-25d +%Y-%m-%d 2>/dev/null || date -d "-25 days" +%Y-%m-%d)
PAST=$(${CLI} trips create \
  --name "Past E2E Trip" \
  --purpose business \
  --start "$PAST_DATE" \
  --end "$PAST_END" \
  --json)
PAST_ID=$(echo "$PAST" | jq -r '.id')

# Test: List upcoming trips
echo "  Listing upcoming trips..."
UPCOMING=$(${CLI} trips list --upcoming --json)

# Verify: Future trip is in list
echo "  Verifying future trip is included..."
if ! echo "$UPCOMING" | jq -e ".[] | select(.id == \"$FUTURE_ID\")" > /dev/null 2>&1; then
  echo "Future trip should be in upcoming list"
  ${CLI} trips delete "$FUTURE_ID" 2>/dev/null || true
  ${CLI} trips delete "$PAST_ID" 2>/dev/null || true
  exit 1
fi

# Verify: Past trip is NOT in list
echo "  Verifying past trip is excluded..."
if echo "$UPCOMING" | jq -e ".[] | select(.id == \"$PAST_ID\")" > /dev/null 2>&1; then
  echo "Past trip should NOT be in upcoming list"
  ${CLI} trips delete "$FUTURE_ID" 2>/dev/null || true
  ${CLI} trips delete "$PAST_ID" 2>/dev/null || true
  exit 1
fi

# Cleanup
echo "  Cleaning up..."
${CLI} trips delete "$FUTURE_ID"
${CLI} trips delete "$PAST_ID"

echo "✓ [UC-TRP-002] List upcoming trips passed"
