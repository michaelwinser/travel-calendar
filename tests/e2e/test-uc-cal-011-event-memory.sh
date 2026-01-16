#!/bin/bash
# Test: [UC-CAL-011] Remember processed calendar events
# PRD: docs/prd/calendar-trip-intelligence.md
#
# Tests that:
# - Reset processed events endpoint works
# - After reset, the system allows suggestions to reappear
#
# NOTE: Full testing of event memory requires Google Calendar OAuth.
# This test covers the reset functionality which can be tested without OAuth.

set -e

API_URL="${API_URL:-http://localhost:3000}"

echo "[UC-CAL-011] Testing event memory - reset processed events"

# TEST 1: Reset processed events endpoint
echo "  Testing reset processed events endpoint..."
RESET_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_URL}/api/calendar/processed-events")
[[ "$RESET_RESPONSE" == "204" ]] || (echo "Reset should return 204, got $RESET_RESPONSE" && exit 1)
echo "  ✓ Reset endpoint returns 204 No Content"

# TEST 2: Reset is idempotent (can be called multiple times)
echo "  Testing reset idempotency..."
RESET_RESPONSE2=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_URL}/api/calendar/processed-events")
[[ "$RESET_RESPONSE2" == "204" ]] || (echo "Second reset should return 204, got $RESET_RESPONSE2" && exit 1)
echo "  ✓ Reset is idempotent"

# TEST 3: Verify suggestions endpoint works (requires no OAuth for empty response)
echo "  Testing suggestions endpoint exists..."
SUGGESTIONS_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/api/calendar/trip-suggestions")
# Should return 200 (empty list) or 401 (no auth) - both are valid
[[ "$SUGGESTIONS_RESPONSE" == "200" || "$SUGGESTIONS_RESPONSE" == "401" ]] || (echo "Suggestions should return 200 or 401, got $SUGGESTIONS_RESPONSE" && exit 1)
echo "  ✓ Suggestions endpoint exists (returned $SUGGESTIONS_RESPONSE)"

echo "✓ [UC-CAL-011] Event memory tests passed"
echo ""
echo "NOTE: Full event memory testing requires:"
echo "  1. Google Calendar OAuth connected"
echo "  2. Calendar with travel events"
echo "  3. Manual verification: import/dismiss → reset → suggestions reappear"
