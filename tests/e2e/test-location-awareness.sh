#!/bin/bash
# Test: [UC-TRP-008 through UC-TRP-012] Location Awareness
# PRD: docs/prd/trip-management.md
#
# Tests location awareness features:
# - UC-TRP-008: Configure base locations (home/work)
# - UC-TRP-009: Set trip locations
# - UC-TRP-010: Query location on specific date
# - UC-TRP-011: Query location for date range
# - UC-TRP-012: Trip without explicit location defaults to "Away"

set -e

CLI="${PROJECT_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}/packages/cli/travel"

echo "[UC-TRP-008] Testing base location configuration"

# Set home location
echo "  Setting home location..."
${CLI} config set home "New York, USA"
HOME_LOC=$(${CLI} config get home --json | jq -r '.home')
[[ "$HOME_LOC" == "New York, USA" ]] || (echo "Home location not set correctly: $HOME_LOC" && exit 1)

# Set work location
echo "  Setting work location..."
${CLI} config set work "San Francisco, USA"
WORK_LOC=$(${CLI} config get work --json | jq -r '.work')
[[ "$WORK_LOC" == "San Francisco, USA" ]] || (echo "Work location not set correctly: $WORK_LOC" && exit 1)

echo "✓ [UC-TRP-008] Base location configuration passed"

echo ""
echo "[UC-TRP-009] Testing trip location creation"

# Create trip with location
echo "  Creating trip with location..."
TRIP=$(${CLI} trips create \
  --name "FOSDEM Brussels" \
  --purpose conference \
  --start 2025-01-29 \
  --end 2025-02-02 \
  --location "Brussels, Belgium" \
  --json)

TRIP_ID=$(echo "$TRIP" | jq -r '.id')
[[ -n "$TRIP_ID" && "$TRIP_ID" != "null" ]] || (echo "Failed to create trip" && exit 1)

# Verify locations were created
echo "  Verifying trip locations..."
LOCATIONS=$(${CLI} trips get-locations "$TRIP_ID" --json)
LOC_COUNT=$(echo "$LOCATIONS" | jq 'length')
[[ "$LOC_COUNT" -ge 4 ]] || (echo "Expected at least 4 days of locations, got $LOC_COUNT" && exit 1)

echo "✓ [UC-TRP-009] Trip location creation passed"

echo ""
echo "[UC-TRP-010] Testing location query on date"

# Query location on a trip date - should return Brussels
echo "  Querying location on trip date..."
LOC_ON_DATE=$(${CLI} location on 2025-01-30 --json)
LOC_LOCATIONS=$(echo "$LOC_ON_DATE" | jq -r '.locations[0]')
LOC_SOURCE=$(echo "$LOC_ON_DATE" | jq -r '.source.type')
[[ "$LOC_LOCATIONS" == "Brussels, Belgium" ]] || (echo "Expected Brussels, got: $LOC_LOCATIONS" && exit 1)
[[ "$LOC_SOURCE" == "trip" ]] || (echo "Expected source type 'trip', got: $LOC_SOURCE" && exit 1)

# Query location on non-trip date - should return home
echo "  Querying location on non-trip date..."
LOC_HOME=$(${CLI} location on 2025-03-15 --json)
LOC_HOME_VAL=$(echo "$LOC_HOME" | jq -r '.locations[0]')
LOC_HOME_SRC=$(echo "$LOC_HOME" | jq -r '.source.type')
[[ "$LOC_HOME_VAL" == "New York, USA" ]] || (echo "Expected home location, got: $LOC_HOME_VAL" && exit 1)
[[ "$LOC_HOME_SRC" == "home" ]] || (echo "Expected source type 'home', got: $LOC_HOME_SRC" && exit 1)

echo "✓ [UC-TRP-010] Location query on date passed"

echo ""
echo "[UC-TRP-011] Testing location range query"

# Query location range spanning trip and non-trip dates
echo "  Querying location range..."
RANGE=$(${CLI} location from 2025-01-25 to 2025-02-05 --json)
SEGMENT_COUNT=$(echo "$RANGE" | jq 'length')
[[ "$SEGMENT_COUNT" -ge 2 ]] || (echo "Expected at least 2 segments, got $SEGMENT_COUNT" && exit 1)

echo "✓ [UC-TRP-011] Location range query passed"

echo ""
echo "[UC-TRP-012] Testing trip without location defaults to Away"

# Create trip without location
echo "  Creating trip without explicit location..."
TRIP2=$(${CLI} trips create \
  --name "Mystery Trip" \
  --purpose vacation \
  --start 2025-05-01 \
  --end 2025-05-03 \
  --json)

TRIP2_ID=$(echo "$TRIP2" | jq -r '.id')
[[ -n "$TRIP2_ID" && "$TRIP2_ID" != "null" ]] || (echo "Failed to create trip" && exit 1)

# Query location on trip date - should return "Away"
echo "  Querying location on trip date (expecting Away)..."
LOC_AWAY=$(${CLI} location on 2025-05-02 --json)
LOC_AWAY_VAL=$(echo "$LOC_AWAY" | jq -r '.locations[0]')
[[ "$LOC_AWAY_VAL" == "Away" ]] || (echo "Expected 'Away' for trip without location, got: $LOC_AWAY_VAL" && exit 1)

echo "✓ [UC-TRP-012] Trip without location defaults to Away passed"

echo ""
echo "Cleaning up..."

# Clean up test trips
${CLI} trips delete "$TRIP_ID" 2>/dev/null || true
${CLI} trips delete "$TRIP2_ID" 2>/dev/null || true

# Clean up config (set to empty to remove)
${CLI} config unset home 2>/dev/null || true
${CLI} config unset work 2>/dev/null || true

echo ""
echo "✓ All location awareness tests passed!"
