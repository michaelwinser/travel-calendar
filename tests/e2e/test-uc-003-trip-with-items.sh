#!/bin/bash
# Test: [UC-TRP-003] Get trip with all items
# PRD: docs/prd/trip-management.md
#
# Tests that a trip includes all its items when fetched,
# and that items are sorted by date.

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-003] Testing trip with items"

# CREATE TRIP
echo "  Creating trip..."
TRIP=$(${CLI} trips create \
  --name "E2E Trip With Items" \
  --purpose vacation \
  --start 2025-06-01 \
  --end 2025-06-05 \
  --json)

TRIP_ID=$(echo "$TRIP" | jq -r '.id')
[[ -n "$TRIP_ID" && "$TRIP_ID" != "null" ]] || (echo "Failed to create trip" && exit 1)

# Cleanup function
cleanup() {
  echo "  Cleaning up..."
  ${CLI} trips delete "$TRIP_ID" 2>/dev/null || true
}
trap cleanup EXIT

# ADD FLIGHT ITEM (later date - should be second)
echo "  Adding flight item..."
${CLI} items add "$TRIP_ID" flight \
  --from JFK \
  --to LAX \
  --date 2025-06-02 \
  --time 10:30 \
  --carrier "Delta" \
  --flight "DL456"

# ADD HOTEL ITEM (earliest date - should be first)
echo "  Adding hotel item..."
${CLI} items add "$TRIP_ID" hotel \
  --name "Beach Resort" \
  --location "Los Angeles" \
  --check-in 2025-06-01 \
  --check-out 2025-06-05 \
  --confirmation "HR789"

# ADD EVENT ITEM (middle date)
echo "  Adding event item..."
${CLI} items add "$TRIP_ID" event \
  --name "Welcome Dinner" \
  --location "Hotel Restaurant" \
  --date 2025-06-01 \
  --time 19:00

# GET TRIP WITH ITEMS
echo "  Getting trip with items..."
TRIP_WITH_ITEMS=$(${CLI} trips get "$TRIP_ID" --json)

# VERIFY ITEM COUNT
ITEM_COUNT=$(echo "$TRIP_WITH_ITEMS" | jq '.items | length')
[[ "$ITEM_COUNT" -eq 3 ]] || (echo "Expected 3 items, got $ITEM_COUNT" && exit 1)
echo "  ✓ Trip has 3 items"

# VERIFY ITEMS ARE SORTED BY DATE
# Hotel check-in (2025-06-01) and Event (2025-06-01) should come before Flight (2025-06-02)
FIRST_DATE=$(echo "$TRIP_WITH_ITEMS" | jq -r '.items[0].checkIn // .items[0].date')
LAST_DATE=$(echo "$TRIP_WITH_ITEMS" | jq -r '.items[2].checkIn // .items[2].date')

# First item should be on or before last item
[[ "$FIRST_DATE" < "$LAST_DATE" || "$FIRST_DATE" == "$LAST_DATE" ]] || \
  (echo "Items not sorted by date: first=$FIRST_DATE, last=$LAST_DATE" && exit 1)
echo "  ✓ Items are sorted by date"

# VERIFY ITEM TYPES
TYPES=$(echo "$TRIP_WITH_ITEMS" | jq -r '[.items[].type] | sort | join(",")')
[[ "$TYPES" == "event,flight,hotel" ]] || (echo "Unexpected item types: $TYPES" && exit 1)
echo "  ✓ All item types present (event, flight, hotel)"

# VERIFY ITEM DATA
HOTEL_NAME=$(echo "$TRIP_WITH_ITEMS" | jq -r '.items[] | select(.type == "hotel") | .name')
[[ "$HOTEL_NAME" == "Beach Resort" ]] || (echo "Hotel name mismatch: $HOTEL_NAME" && exit 1)
echo "  ✓ Item data preserved correctly"

# DELETE ITEM
echo "  Testing item deletion..."
FLIGHT_ID=$(echo "$TRIP_WITH_ITEMS" | jq -r '.items[] | select(.type == "flight") | .id')
${CLI} items delete "$FLIGHT_ID"

# VERIFY ITEM DELETED
TRIP_AFTER_DELETE=$(${CLI} trips get "$TRIP_ID" --json)
ITEM_COUNT_AFTER=$(echo "$TRIP_AFTER_DELETE" | jq '.items | length')
[[ "$ITEM_COUNT_AFTER" -eq 2 ]] || (echo "Expected 2 items after delete, got $ITEM_COUNT_AFTER" && exit 1)
echo "  ✓ Item deleted successfully"

echo "✓ [UC-TRP-003] Trip with items tests passed"
