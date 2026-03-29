#!/bin/sh
# Scenario test: Trip lifecycle
#
# Tests: create trip with dates, add activities to trip, update trip,
# verify trip list, delete trip (activities become standalone).
#
# Uses isolated temp database — no side effects.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

BINARY="$PROJECT_DIR/travel"
TEST_DATA=$(mktemp -d)
export DEV_USER_EMAIL="trip-test@localhost"
FAILURES=0

if [ -t 1 ]; then
    GREEN='\033[0;32m'; RED='\033[0;31m'; DIM='\033[2m'; NC='\033[0m'
else
    GREEN=''; RED=''; DIM=''; NC=''
fi

cleanup() { rm -rf "$TEST_DATA"; }
trap cleanup EXIT

T="$BINARY --data $TEST_DATA"

run_check() {
    desc="$1"; shift
    expected="$1"; shift
    printf "${DIM}\$ %s${NC}\n" "$*"
    actual=$("$@" 2>/dev/null) || true
    echo "$actual"
    echo ""
    if echo "$actual" | grep -q "$expected"; then
        return 0
    else
        printf "${RED}  ^^^ expected to see: %s${NC}\n\n" "$expected"
        FAILURES=$((FAILURES + 1))
        return 0
    fi
}

# Build
echo "Building travel binary..."
(cd "$PROJECT_DIR" && go build -o travel .)
echo ""

echo "=== Trip Lifecycle Scenario ==="
echo ""

# Create a trip with dates
echo "--- Create a trip with explicit dates ---"
run_check "Create trip" "FOSDEM" \
    $T trip create "FOSDEM 2027" --from 2027-01-30 --to 2027-02-02

# Verify trip shows in list
echo "--- Trip list ---"
run_check "Trip in list" "FOSDEM 2027" \
    $T trip list

# Add activities to the trip
echo "--- Add activities to the trip ---"
run_check "Add flight" "Created" \
    $T add "Flight to Brussels" --from 2027-01-30 --to 2027-01-30 --type travel --trip "FOSDEM 2027"

run_check "Add conference" "Created" \
    $T add "FOSDEM Day 1" --from 2027-02-01 --to 2027-02-01 --loc Brussels --type conference --trip "FOSDEM 2027"

# Verify activities are listed
echo "--- List activities ---"
run_check "Flight listed" "Flight to Brussels" \
    $T list

run_check "Conference listed" "FOSDEM Day 1" \
    $T list

# Show trip details
echo "--- Trip details ---"
run_check "Trip show" "Flight to Brussels" \
    $T trip show "FOSDEM 2027"

# Update an activity
echo "--- Update activity ---"
FLIGHT_ID=$($T list 2>/dev/null | grep "Flight to Brussels" | awk '{print $1}')
if [ -n "$FLIGHT_ID" ]; then
    run_check "Update flight" "Updated" \
        $T update "$FLIGHT_ID" --loc "EWR"
else
    printf "${RED}  Could not find Flight to Brussels${NC}\n\n"
    FAILURES=$((FAILURES + 1))
fi

# Quick add with trip
echo "--- Quick add into trip ---"
run_check "Quick add" "Created" \
    $T quick "Hotel in Brussels Feb 1-2" --trip "FOSDEM 2027" --yes

# Delete trip (activities become standalone)
echo "--- Delete trip ---"
TRIP_ID=$($T trip list 2>/dev/null | grep "FOSDEM" | awk '{print $1}')
if [ -n "$TRIP_ID" ]; then
    run_check "Delete trip" "Deleted" \
        $T trip delete "$TRIP_ID"
else
    printf "${RED}  Could not find FOSDEM trip to delete${NC}\n\n"
    FAILURES=$((FAILURES + 1))
fi

# Verify activities still exist (standalone)
echo "--- Activities survive trip deletion ---"
run_check "Activities still exist" "Flight to Brussels" \
    $T list

# Results
if [ "$FAILURES" -gt 0 ]; then
    printf "${RED}TRIP SCENARIO FAILED: %d check(s) did not match${NC}\n" "$FAILURES"
    exit 1
else
    printf "${GREEN}TRIP SCENARIO PASSED${NC}\n"
fi
