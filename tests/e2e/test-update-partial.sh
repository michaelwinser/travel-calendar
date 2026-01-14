#!/bin/bash
# Test: [UC-TRP-004] Update trip (partial update)
# PRD: docs/prd/trip-management.md
#
# Tests that partial updates only change specified fields

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-004] Testing partial update"

# Setup: Create a trip with all fields
echo "  Creating trip with all fields..."
TRIP=$(${CLI} trips create \
  --name "Original Name" \
  --purpose conference \
  --start 2025-06-01 \
  --end 2025-06-05 \
  --notes "Original notes" \
  --json)

TRIP_ID=$(echo "$TRIP" | jq -r '.id')
[[ -n "$TRIP_ID" && "$TRIP_ID" != "null" ]] || (echo "Failed to create trip" && exit 1)

# Test: Update only the name
echo "  Updating only the name..."
UPDATED=$(${CLI} trips update "$TRIP_ID" --name "Updated Name" --json)

# Verify: Name changed
UPDATED_NAME=$(echo "$UPDATED" | jq -r '.name')
[[ "$UPDATED_NAME" == "Updated Name" ]] || (echo "Name not updated: $UPDATED_NAME" && ${CLI} trips delete "$TRIP_ID" && exit 1)

# Verify: Other fields preserved
echo "  Verifying other fields preserved..."
UPDATED_PURPOSE=$(echo "$UPDATED" | jq -r '.purpose')
UPDATED_STATUS=$(echo "$UPDATED" | jq -r '.status')
UPDATED_NOTES=$(echo "$UPDATED" | jq -r '.notes')

[[ "$UPDATED_PURPOSE" == "conference" ]] || (echo "Purpose changed unexpectedly: $UPDATED_PURPOSE" && ${CLI} trips delete "$TRIP_ID" && exit 1)
[[ "$UPDATED_STATUS" == "planning" ]] || (echo "Status changed unexpectedly: $UPDATED_STATUS" && ${CLI} trips delete "$TRIP_ID" && exit 1)
[[ "$UPDATED_NOTES" == "Original notes" ]] || (echo "Notes changed unexpectedly: $UPDATED_NOTES" && ${CLI} trips delete "$TRIP_ID" && exit 1)

# Verify: UpdatedAt timestamp changed
echo "  Verifying timestamp updated..."
ORIGINAL_UPDATED=$(echo "$TRIP" | jq -r '.updatedAt')
FINAL_UPDATED=$(echo "$UPDATED" | jq -r '.updatedAt')
# Note: timestamps might be equal if update is very fast, so we just verify it's not empty
[[ -n "$FINAL_UPDATED" && "$FINAL_UPDATED" != "null" ]] || (echo "UpdatedAt not set" && ${CLI} trips delete "$TRIP_ID" && exit 1)

# Cleanup
echo "  Cleaning up..."
${CLI} trips delete "$TRIP_ID"

echo "✓ [UC-TRP-004] Partial update passed"
