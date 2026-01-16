#!/bin/bash
# Test: [UC-ORG-001] Merge two existing trips
# PRD: docs/prd/trip-organization.md
#
# Tests that merging two trips:
# - Moves all items from source to target
# - Extends target dates if source has wider range
# - Concatenates notes (optional)
# - Deletes source trip

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"
API_URL="${API_URL:-http://localhost:3000}"

echo "[UC-ORG-001] Testing merge trips"

# CREATE SOURCE TRIP
echo "  Creating source trip..."
SOURCE=$(${CLI} trips create \
  --name "Source Trip" \
  --purpose vacation \
  --start 2025-07-01 \
  --end 2025-07-05 \
  --notes "Source notes" \
  --json)

SOURCE_ID=$(echo "$SOURCE" | jq -r '.id')
[[ -n "$SOURCE_ID" && "$SOURCE_ID" != "null" ]] || (echo "Failed to create source trip" && exit 1)

# CREATE TARGET TRIP
echo "  Creating target trip..."
TARGET=$(${CLI} trips create \
  --name "Target Trip" \
  --purpose vacation \
  --start 2025-07-03 \
  --end 2025-07-07 \
  --notes "Target notes" \
  --json)

TARGET_ID=$(echo "$TARGET" | jq -r '.id')
[[ -n "$TARGET_ID" && "$TARGET_ID" != "null" ]] || (echo "Failed to create target trip" && exit 1)

# Cleanup function
cleanup() {
  echo "  Cleaning up..."
  ${CLI} trips delete "$TARGET_ID" 2>/dev/null || true
  ${CLI} trips delete "$SOURCE_ID" 2>/dev/null || true
}
trap cleanup EXIT

# ADD ITEMS TO SOURCE TRIP
echo "  Adding items to source trip..."
${CLI} items add "$SOURCE_ID" flight \
  --from JFK \
  --to LAX \
  --date 2025-07-01 \
  --time 09:00 \
  --carrier "Delta" \
  --flight "DL100"

${CLI} items add "$SOURCE_ID" hotel \
  --name "LA Hotel" \
  --location "Los Angeles" \
  --check-in 2025-07-01 \
  --check-out 2025-07-05

# ADD ITEMS TO TARGET TRIP
echo "  Adding items to target trip..."
${CLI} items add "$TARGET_ID" event \
  --name "Conference" \
  --location "Convention Center" \
  --date 2025-07-04

# VERIFY ITEMS BEFORE MERGE
SOURCE_ITEMS_BEFORE=$(${CLI} trips get "$SOURCE_ID" --json | jq '.items | length')
TARGET_ITEMS_BEFORE=$(${CLI} trips get "$TARGET_ID" --json | jq '.items | length')
echo "  Source has $SOURCE_ITEMS_BEFORE items, target has $TARGET_ITEMS_BEFORE items"

# MERGE SOURCE INTO TARGET
echo "  Merging source into target..."
MERGED=$(curl -s -X POST "${API_URL}/api/trips/${SOURCE_ID}/merge/${TARGET_ID}" \
  -H "Content-Type: application/json" \
  -d '{"mergeNotes": true}')

# Check if merge was successful
MERGED_ID=$(echo "$MERGED" | jq -r '.id')
[[ "$MERGED_ID" == "$TARGET_ID" ]] || (echo "Merge failed: $MERGED" && exit 1)
echo "  ✓ Merge completed successfully"

# VERIFY ALL ITEMS MOVED TO TARGET
TARGET_ITEMS_AFTER=$(echo "$MERGED" | jq '.items | length')
EXPECTED_ITEMS=$((SOURCE_ITEMS_BEFORE + TARGET_ITEMS_BEFORE))
[[ "$TARGET_ITEMS_AFTER" -eq "$EXPECTED_ITEMS" ]] || (echo "Expected $EXPECTED_ITEMS items, got $TARGET_ITEMS_AFTER" && exit 1)
echo "  ✓ All items moved to target ($TARGET_ITEMS_AFTER items)"

# VERIFY DATES EXTENDED (source starts earlier)
MERGED_START=$(echo "$MERGED" | jq -r '.startDate')
[[ "$MERGED_START" == "2025-07-01" ]] || (echo "Start date should be 2025-07-01, got $MERGED_START" && exit 1)
echo "  ✓ Start date extended to source trip start"

MERGED_END=$(echo "$MERGED" | jq -r '.endDate')
[[ "$MERGED_END" == "2025-07-07" ]] || (echo "End date should be 2025-07-07, got $MERGED_END" && exit 1)
echo "  ✓ End date preserved from target trip"

# VERIFY NOTES CONCATENATED
MERGED_NOTES=$(echo "$MERGED" | jq -r '.notes')
echo "$MERGED_NOTES" | grep -q "Target notes" || (echo "Target notes missing" && exit 1)
echo "$MERGED_NOTES" | grep -q "Source notes" || (echo "Source notes missing" && exit 1)
echo "  ✓ Notes concatenated"

# VERIFY SOURCE TRIP DELETED
echo "  Verifying source trip deleted..."
SOURCE_GET=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/api/trips/${SOURCE_ID}")
[[ "$SOURCE_GET" == "404" ]] || (echo "Source trip should be deleted, got HTTP $SOURCE_GET" && exit 1)
echo "  ✓ Source trip deleted"

echo "✓ [UC-ORG-001] Merge trips tests passed"
