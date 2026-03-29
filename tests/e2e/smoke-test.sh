#!/bin/sh
# Smoke test: Travel Calendar CLI walkthrough
#
# Demonstrates typical usage while verifying output. Serves as both
# a test and a usage example for new users.
#
# Uses the in-process transport — no server, no login, no setup needed.
#
# Usage: ./tests/e2e/smoke-test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Config ---
BINARY="$PROJECT_DIR/travel"
FAILURES=0

# Isolated temp database — each run starts clean
TEST_DATA=$(mktemp -d)
export DEV_USER_EMAIL="smoke-test@localhost"

# Colors
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    DIM='\033[2m'
    NC='\033[0m'
else
    GREEN='' RED='' DIM='' NC=''
fi

cleanup() {
    rm -rf "$TEST_DATA"
}
trap cleanup EXIT

# Run a command, show it, capture output, check against expected patterns.
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

# ============================================
# Setup
# ============================================

echo "Building travel binary..."
(cd "$PROJECT_DIR" && go build -o travel .)
echo ""

# ============================================
# Walkthrough
# ============================================

echo "=== Travel Calendar CLI Walkthrough ==="
echo ""

# --- Plan a conference trip ---

echo "--- Planning a conference trip to Brussels ---"
echo ""
run_check "Create conference" "European Summit" \
    "$BINARY" --data "$TEST_DATA" add "European Summit" --from 2026-10-04 --to 2026-10-07 --loc Brussels --type conference

# --- Quick add a vacation ---

echo "--- Quick-adding a Hawaiian vacation ---"
echo ""
run_check "Quick add vacation" "Hawaii Vacation" \
    "$BINARY" --data "$TEST_DATA" quick "Hawaii Vacation Oct 15 - Oct 22 in Maui" --yes

# --- Create a conflict ---

echo "--- Oops, dentist appointment during the conference ---"
echo ""
run_check "Create conflicting appointment" "Overlapping activities" \
    "$BINARY" --data "$TEST_DATA" add "Dentist Appointment" --from 2026-10-05 --loc Home --type commitment

# --- List everything ---

echo "--- What does October look like? ---"
echo ""
run_check "List October" "European Summit" \
    "$BINARY" --data "$TEST_DATA" list --month 2026-10

# --- Check a conflict date ---

echo "--- Am I free on October 5? ---"
echo ""
run_check "Check conflict date" "Location conflict" \
    "$BINARY" --data "$TEST_DATA" check 2026-10-05

# --- Check a non-conflict date ---

echo "--- What about October 6? ---"
echo ""
run_check "Check clean date" "Brussels" \
    "$BINARY" --data "$TEST_DATA" check 2026-10-06

# --- Check a day with nothing ---

echo "--- How about November 1? ---"
echo ""
run_check "Check empty date" "Home" \
    "$BINARY" --data "$TEST_DATA" check 2026-11-01

# --- Delete the conflicting appointment ---

echo "--- Cancel the dentist, resolve the conflict ---"
echo ""
DENTIST_PREFIX=$("$BINARY" --data "$TEST_DATA" list 2>/dev/null | grep "Dentist" | awk '{print $1}')
if [ -n "$DENTIST_PREFIX" ]; then
    run_check "Delete dentist" "Deleted" \
        "$BINARY" --data "$TEST_DATA" delete "$DENTIST_PREFIX"
else
    printf "${RED}  Could not find Dentist Appointment to delete${NC}\n\n"
    FAILURES=$((FAILURES + 1))
fi

# --- Verify conflict is resolved ---

echo "--- October 5 after cancelling the dentist ---"
echo ""
run_check "Conflict resolved" "Brussels" \
    "$BINARY" --data "$TEST_DATA" check 2026-10-05

# --- Final list ---

echo "--- Final schedule ---"
echo ""
run_check "Final list" "Hawaii Vacation" \
    "$BINARY" --data "$TEST_DATA" list

# ============================================
# Results
# ============================================

if [ "$FAILURES" -gt 0 ]; then
    printf "${RED}SMOKE TEST FAILED: %d check(s) did not match expected output${NC}\n" "$FAILURES"
    exit 1
else
    printf "${GREEN}SMOKE TEST PASSED${NC}\n"
fi
