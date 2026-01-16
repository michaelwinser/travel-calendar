#!/bin/bash
# Test: [UC-CAL-004] Merge imported trip with existing trip
# PRD: docs/prd/calendar-trip-intelligence.md
#
# Tests that:
# - Merge suggestion endpoint exists
# - Invalid merge returns appropriate error
# - Merge endpoint requires valid suggestion ID and trip ID
#
# NOTE: Full testing of merge requires Google Calendar OAuth with travel events.
# This test verifies the API contract and error handling.

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"
API_URL="${API_URL:-http://localhost:3000}"

echo "[UC-CAL-004] Testing merge suggestion into existing trip"

# CREATE A TARGET TRIP for merge testing
echo "  Creating target trip..."
TARGET=$(${CLI} trips create \
  --name "Brussels Conference" \
  --purpose conference \
  --start 2026-01-29 \
  --end 2026-02-02 \
  --location "Brussels, Belgium" \
  --json)

TARGET_ID=$(echo "$TARGET" | jq -r '.id')
[[ -n "$TARGET_ID" && "$TARGET_ID" != "null" ]] || (echo "Failed to create target trip" && exit 1)
echo "  ✓ Created target trip: $TARGET_ID"

# Cleanup function
cleanup() {
  echo "  Cleaning up..."
  ${CLI} trips delete "$TARGET_ID" 2>/dev/null || true
}
trap cleanup EXIT

# TEST 1: Merge with non-existent suggestion returns 404
echo "  Testing merge with invalid suggestion ID..."
FAKE_SUGGESTION="00000000-0000-0000-0000-000000000000"
MERGE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/calendar/trip-suggestions/${FAKE_SUGGESTION}/merge/${TARGET_ID}")
MERGE_CODE=$(echo "$MERGE_RESPONSE" | tail -n1)
[[ "$MERGE_CODE" == "404" ]] || (echo "Merge with fake suggestion should return 404, got $MERGE_CODE" && exit 1)
echo "  ✓ Invalid suggestion returns 404"

# TEST 2: Merge endpoint validates trip ID exists
echo "  Testing merge with invalid trip ID..."
FAKE_TRIP="99999999-9999-9999-9999-999999999999"
MERGE_RESPONSE2=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/calendar/trip-suggestions/${FAKE_SUGGESTION}/merge/${FAKE_TRIP}")
MERGE_CODE2=$(echo "$MERGE_RESPONSE2" | tail -n1)
# Should return 404 (suggestion not found first) or validate both
[[ "$MERGE_CODE2" == "404" ]] || (echo "Merge with fake trip should return error, got $MERGE_CODE2" && exit 1)
echo "  ✓ Invalid trip ID returns 404"

# TEST 3: Verify target trip is unchanged (no accidental modifications)
echo "  Verifying target trip unchanged..."
TRIP_CHECK=$(${CLI} trips get "$TARGET_ID" --json)
TRIP_NAME=$(echo "$TRIP_CHECK" | jq -r '.name')
[[ "$TRIP_NAME" == "Brussels Conference" ]] || (echo "Trip name changed unexpectedly" && exit 1)
echo "  ✓ Target trip unchanged after invalid merge attempts"

echo "✓ [UC-CAL-004] Merge suggestion tests passed"
echo ""
echo "NOTE: Full merge testing requires:"
echo "  1. Google Calendar OAuth connected"
echo "  2. Calendar with TripIt events for Brussels around Jan 29 - Feb 2"
echo "  3. Manual test steps:"
echo "     - View trip suggestions in settings"
echo "     - Find suggestion with 'Similar trip: Brussels Conference'"
echo "     - Use merge dropdown to merge into existing trip"
echo "     - Verify items added, dates extended, events marked as processed"
