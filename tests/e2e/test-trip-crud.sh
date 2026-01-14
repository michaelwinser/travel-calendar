#!/bin/bash
# Test: [UC-TRP-001] Trip CRUD operations
# PRD: docs/prd/trip-management.md
#
# Tests basic create, read, update, delete operations for trips.

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-001] Testing trip CRUD operations"

# CREATE
echo "  Creating trip..."
TRIP=$(${CLI} trips create \
  --name "E2E Test Trip" \
  --purpose conference \
  --start 2025-06-01 \
  --end 2025-06-05 \
  --json)

TRIP_ID=$(echo "$TRIP" | jq -r '.id')
[[ -n "$TRIP_ID" && "$TRIP_ID" != "null" ]] || (echo "Failed to create trip" && exit 1)

# READ
echo "  Reading trip..."
GOT=$(${CLI} trips get "$TRIP_ID" --json)
GOT_NAME=$(echo "$GOT" | jq -r '.name')
[[ "$GOT_NAME" == "E2E Test Trip" ]] || (echo "Name mismatch: $GOT_NAME" && exit 1)

# UPDATE
echo "  Updating trip..."
UPDATED=$(${CLI} trips update "$TRIP_ID" --name "Updated Test Trip" --status confirmed --json)
UPDATED_NAME=$(echo "$UPDATED" | jq -r '.name')
UPDATED_STATUS=$(echo "$UPDATED" | jq -r '.status')
[[ "$UPDATED_NAME" == "Updated Test Trip" ]] || (echo "Update name failed" && exit 1)
[[ "$UPDATED_STATUS" == "confirmed" ]] || (echo "Update status failed" && exit 1)

# DELETE
echo "  Deleting trip..."
${CLI} trips delete "$TRIP_ID"

# VERIFY DELETED
echo "  Verifying deletion..."
if ${CLI} trips get "$TRIP_ID" --json 2>/dev/null | jq -e '.id' > /dev/null 2>&1; then
  echo "Trip should have been deleted but still exists"
  exit 1
fi

echo "✓ [UC-TRP-001] Trip CRUD operations passed"
