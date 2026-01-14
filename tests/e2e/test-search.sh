#!/bin/bash
# Test: [UC-TRP-006] Search trips
# PRD: docs/prd/trip-management.md
#
# Tests search functionality: case-insensitive, partial matching

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-006] Testing search trips"

# Use unique identifiers to avoid collisions with leftover test data
UNIQUE_ID="$$$(date +%s)"

# Setup: Create trips with different names
echo "  Creating test trips..."
TRIP1=$(${CLI} trips create \
  --name "SearchTest${UNIQUE_ID} Alpha Conference" \
  --purpose conference \
  --json)
TRIP1_ID=$(echo "$TRIP1" | jq -r '.id')

TRIP2=$(${CLI} trips create \
  --name "SearchTest${UNIQUE_ID} Beta Vacation" \
  --purpose vacation \
  --json)
TRIP2_ID=$(echo "$TRIP2" | jq -r '.id')

TRIP3=$(${CLI} trips create \
  --name "SearchTest${UNIQUE_ID} Gamma Business" \
  --purpose business \
  --json)
TRIP3_ID=$(echo "$TRIP3" | jq -r '.id')

# Test 1: Exact match (using unique prefix + "Alpha")
echo "  Testing exact match..."
RESULTS=$(${CLI} trips search "SearchTest${UNIQUE_ID} Alpha" --json)
COUNT=$(echo "$RESULTS" | jq 'length')
[[ "$COUNT" -eq 1 ]] || (echo "Expected 1 result for 'SearchTest${UNIQUE_ID} Alpha', got $COUNT" && ${CLI} trips delete "$TRIP1_ID" && ${CLI} trips delete "$TRIP2_ID" && ${CLI} trips delete "$TRIP3_ID" && exit 1)

# Test 2: Case-insensitive
echo "  Testing case-insensitive search..."
RESULTS=$(${CLI} trips search "searchtest${UNIQUE_ID} alpha" --json)
COUNT=$(echo "$RESULTS" | jq 'length')
[[ "$COUNT" -eq 1 ]] || (echo "Expected 1 result for lowercase search, got $COUNT" && ${CLI} trips delete "$TRIP1_ID" && ${CLI} trips delete "$TRIP2_ID" && ${CLI} trips delete "$TRIP3_ID" && exit 1)

# Test 3: Partial match (matches all 3 with unique prefix)
echo "  Testing partial match..."
RESULTS=$(${CLI} trips search "SearchTest${UNIQUE_ID}" --json)
COUNT=$(echo "$RESULTS" | jq 'length')
[[ "$COUNT" -eq 3 ]] || (echo "Expected 3 results for partial match 'SearchTest${UNIQUE_ID}', got $COUNT" && ${CLI} trips delete "$TRIP1_ID" && ${CLI} trips delete "$TRIP2_ID" && ${CLI} trips delete "$TRIP3_ID" && exit 1)

# Test 4: No match
echo "  Testing no match..."
RESULTS=$(${CLI} trips search "xyznotfound${UNIQUE_ID}zzz" --json)
COUNT=$(echo "$RESULTS" | jq 'length')
[[ "$COUNT" -eq 0 ]] || (echo "Expected 0 results for 'xyznotfound', got $COUNT" && ${CLI} trips delete "$TRIP1_ID" && ${CLI} trips delete "$TRIP2_ID" && ${CLI} trips delete "$TRIP3_ID" && exit 1)

# Cleanup
echo "  Cleaning up..."
${CLI} trips delete "$TRIP1_ID"
${CLI} trips delete "$TRIP2_ID"
${CLI} trips delete "$TRIP3_ID"

echo "✓ [UC-TRP-006] Search trips passed"
