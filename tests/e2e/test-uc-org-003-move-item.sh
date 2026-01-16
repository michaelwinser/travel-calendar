#!/bin/bash
# Test: [UC-ORG-003] Move item to another trip
# PRD: docs/prd/trip-organization.md
#
# Tests that moving an item:
# - Removes item from source trip
# - Adds item to target trip
# - Can move to a new trip (create new trip)

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"
API_URL="${API_URL:-http://localhost:3000}"

echo "[UC-ORG-003] Testing move item"

# CREATE SOURCE TRIP
echo "  Creating source trip..."
SOURCE=$(${CLI} trips create \
  --name "Source Trip" \
  --purpose vacation \
  --start 2025-08-01 \
  --end 2025-08-05 \
  --json)

SOURCE_ID=$(echo "$SOURCE" | jq -r '.id')
[[ -n "$SOURCE_ID" && "$SOURCE_ID" != "null" ]] || (echo "Failed to create source trip" && exit 1)

# CREATE TARGET TRIP
echo "  Creating target trip..."
TARGET=$(${CLI} trips create \
  --name "Target Trip" \
  --purpose business \
  --start 2025-08-10 \
  --end 2025-08-15 \
  --json)

TARGET_ID=$(echo "$TARGET" | jq -r '.id')
[[ -n "$TARGET_ID" && "$TARGET_ID" != "null" ]] || (echo "Failed to create target trip" && exit 1)

# Variable to track new trip ID if created
NEW_TRIP_ID=""

# Cleanup function
cleanup() {
  echo "  Cleaning up..."
  ${CLI} trips delete "$SOURCE_ID" 2>/dev/null || true
  ${CLI} trips delete "$TARGET_ID" 2>/dev/null || true
  [[ -n "$NEW_TRIP_ID" ]] && ${CLI} trips delete "$NEW_TRIP_ID" 2>/dev/null || true
}
trap cleanup EXIT

# ADD ITEMS TO SOURCE TRIP
echo "  Adding items to source trip..."
${CLI} items add "$SOURCE_ID" flight \
  --from JFK \
  --to LAX \
  --date 2025-08-01 \
  --time 09:00 \
  --carrier "United" \
  --flight "UA200"

${CLI} items add "$SOURCE_ID" hotel \
  --name "Beach Resort" \
  --location "Los Angeles" \
  --check-in 2025-08-01 \
  --check-out 2025-08-05 \
  --confirmation "HR123"

# GET ITEM IDS
SOURCE_TRIP=$(${CLI} trips get "$SOURCE_ID" --json)
FLIGHT_ID=$(echo "$SOURCE_TRIP" | jq -r '.items[] | select(.type == "flight") | .id')
HOTEL_ID=$(echo "$SOURCE_TRIP" | jq -r '.items[] | select(.type == "hotel") | .id')
echo "  Flight ID: $FLIGHT_ID"
echo "  Hotel ID: $HOTEL_ID"

# TEST 1: MOVE ITEM TO EXISTING TRIP
echo "  Moving flight to target trip..."
MOVE_RESULT=$(curl -s -X POST "${API_URL}/api/items/${FLIGHT_ID}/move" \
  -H "Content-Type: application/json" \
  -d "{\"targetTripId\": \"${TARGET_ID}\"}")

# Verify move result
MOVED_ITEM_TRIP=$(echo "$MOVE_RESULT" | jq -r '.item.tripId')
[[ "$MOVED_ITEM_TRIP" == "$TARGET_ID" ]] || (echo "Item not moved to target: $MOVE_RESULT" && exit 1)
echo "  ✓ Flight moved to target trip"

# Verify item removed from source
SOURCE_AFTER=$(${CLI} trips get "$SOURCE_ID" --json)
SOURCE_ITEM_COUNT=$(echo "$SOURCE_AFTER" | jq '.items | length')
[[ "$SOURCE_ITEM_COUNT" -eq 1 ]] || (echo "Source should have 1 item, got $SOURCE_ITEM_COUNT" && exit 1)
echo "  ✓ Source trip now has 1 item"

# Verify item added to target
TARGET_AFTER=$(${CLI} trips get "$TARGET_ID" --json)
TARGET_ITEM_COUNT=$(echo "$TARGET_AFTER" | jq '.items | length')
[[ "$TARGET_ITEM_COUNT" -eq 1 ]] || (echo "Target should have 1 item, got $TARGET_ITEM_COUNT" && exit 1)
echo "  ✓ Target trip now has 1 item"

# Verify item type in target
TARGET_ITEM_TYPE=$(echo "$TARGET_AFTER" | jq -r '.items[0].type')
[[ "$TARGET_ITEM_TYPE" == "flight" ]] || (echo "Expected flight, got $TARGET_ITEM_TYPE" && exit 1)
echo "  ✓ Moved item is a flight"

# TEST 2: MOVE ITEM TO NEW TRIP
echo "  Moving hotel to a new trip..."
MOVE_NEW_RESULT=$(curl -s -X POST "${API_URL}/api/items/${HOTEL_ID}/move" \
  -H "Content-Type: application/json" \
  -d '{"newTrip": {"name": "New Trip From Move", "purpose": "other"}}')

# Verify new trip was created
NEW_TRIP_ID=$(echo "$MOVE_NEW_RESULT" | jq -r '.trip.id')
[[ -n "$NEW_TRIP_ID" && "$NEW_TRIP_ID" != "null" ]] || (echo "New trip not created: $MOVE_NEW_RESULT" && exit 1)
echo "  ✓ New trip created: $NEW_TRIP_ID"

# Verify item moved to new trip
NEW_TRIP_ITEM_TRIP=$(echo "$MOVE_NEW_RESULT" | jq -r '.item.tripId')
[[ "$NEW_TRIP_ITEM_TRIP" == "$NEW_TRIP_ID" ]] || (echo "Item not moved to new trip" && exit 1)
echo "  ✓ Hotel moved to new trip"

# Verify source trip now empty
SOURCE_FINAL=$(${CLI} trips get "$SOURCE_ID" --json)
SOURCE_FINAL_COUNT=$(echo "$SOURCE_FINAL" | jq '.items | length')
[[ "$SOURCE_FINAL_COUNT" -eq 0 ]] || (echo "Source should have 0 items, got $SOURCE_FINAL_COUNT" && exit 1)
echo "  ✓ Source trip now has 0 items"

# Verify new trip has the item
NEW_TRIP=$(${CLI} trips get "$NEW_TRIP_ID" --json)
NEW_TRIP_ITEM_COUNT=$(echo "$NEW_TRIP" | jq '.items | length')
[[ "$NEW_TRIP_ITEM_COUNT" -eq 1 ]] || (echo "New trip should have 1 item, got $NEW_TRIP_ITEM_COUNT" && exit 1)
echo "  ✓ New trip has 1 item"

NEW_TRIP_ITEM_TYPE=$(echo "$NEW_TRIP" | jq -r '.items[0].type')
[[ "$NEW_TRIP_ITEM_TYPE" == "hotel" ]] || (echo "Expected hotel, got $NEW_TRIP_ITEM_TYPE" && exit 1)
echo "  ✓ New trip item is a hotel"

# TEST 3: ERROR CASE - MOVE TO SAME TRIP
echo "  Testing error: move to same trip..."
SAME_TRIP_RESULT=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/items/${FLIGHT_ID}/move" \
  -H "Content-Type: application/json" \
  -d "{\"targetTripId\": \"${TARGET_ID}\"}")

[[ "$SAME_TRIP_RESULT" == "400" ]] || (echo "Expected 400 for same trip move, got $SAME_TRIP_RESULT" && exit 1)
echo "  ✓ Same trip move rejected with 400"

echo "✓ [UC-ORG-003] Move item tests passed"
