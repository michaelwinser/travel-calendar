#!/bin/sh
# Smoke test: Travel Calendar CLI walkthrough
#
# Demonstrates typical usage while verifying output. Serves as both
# a test and a usage example for new users.
#
# Prerequisites:
#   - appbase binary installed (go install github.com/michaelwinser/appbase/cmd/appbase@latest)
#
# Usage: ./tests/e2e/smoke-test.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Config ---
PORT=3198
DB_PATH="$PROJECT_DIR/data/smoke-test.db"
BINARY="$PROJECT_DIR/travel"
APPBASE="${APPBASE:-$(go env GOPATH)/bin/appbase}"
SERVER_PID=""
FAILURES=0

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
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    "$APPBASE" test-logout --app travel-calendar 2>/dev/null || true
    rm -f "$DB_PATH" "$BINARY"
}
trap cleanup EXIT

# Run a command, show it, capture output, check against expected patterns.
# Usage: run_check "description" "expected_pattern" command args...
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
        return 0  # don't exit on failure
    fi
}

# ============================================
# Setup
# ============================================

echo "Building travel binary..."
(cd "$PROJECT_DIR" && go build -o travel .)
echo ""

echo "Starting server on port $PORT..."
rm -f "$DB_PATH"
mkdir -p "$(dirname "$DB_PATH")"
STORE_TYPE=sqlite SQLITE_DB_PATH="$DB_PATH" PORT="$PORT" \
    "$BINARY" serve > /dev/null 2>&1 &
SERVER_PID=$!

for i in 1 2 3 4 5 6 7 8 9 10; do
    if curl -s "http://localhost:$PORT/health" > /dev/null 2>&1; then break; fi
    if [ "$i" = "10" ]; then echo "Server failed to start"; exit 1; fi
    sleep 0.5
done

echo "Logging in for CLI testing..."
STORE_TYPE=sqlite SQLITE_DB_PATH="$DB_PATH" \
    "$APPBASE" test-login --db "$DB_PATH" --server "http://localhost:$PORT" --app travel-calendar
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
    "$BINARY" add "European Summit" --from 2026-10-04 --to 2026-10-07 --loc Brussels --type conference

# --- Quick add a vacation ---

echo "--- Quick-adding a Hawaiian vacation ---"
echo ""
run_check "Quick add vacation" "Hawaii Vacation" \
    "$BINARY" quick "Hawaii Vacation Oct 15 - Oct 22 in Maui" --yes

# --- Create a conflict ---

echo "--- Oops, dentist appointment during the conference ---"
echo ""
run_check "Create conflicting appointment" "Overlapping activities" \
    "$BINARY" add "Dentist Appointment" --from 2026-10-05 --loc Home --type commitment

# --- List everything ---

echo "--- What does October look like? ---"
echo ""
run_check "List October" "European Summit" \
    "$BINARY" list --month 2026-10

# --- Check a conflict date ---

echo "--- Am I free on October 5? ---"
echo ""
run_check "Check conflict date" "Location conflict" \
    "$BINARY" check 2026-10-05

# --- Check a non-conflict date ---

echo "--- What about October 6? ---"
echo ""
run_check "Check clean date" "Brussels" \
    "$BINARY" check 2026-10-06

# --- Check a day with nothing ---

echo "--- How about November 1? ---"
echo ""
run_check "Check empty date" "Home" \
    "$BINARY" check 2026-11-01

# --- Delete the conflicting appointment ---

echo "--- Cancel the dentist, resolve the conflict ---"
echo ""
# Find the dentist ID prefix from the list
DENTIST_PREFIX=$("$BINARY" list 2>/dev/null | grep "Dentist" | awk '{print $1}')
if [ -n "$DENTIST_PREFIX" ]; then
    run_check "Delete dentist" "Deleted" \
        "$BINARY" delete "$DENTIST_PREFIX"
else
    printf "${RED}  Could not find Dentist Appointment to delete${NC}\n\n"
    FAILURES=$((FAILURES + 1))
fi

# --- Verify conflict is resolved ---

echo "--- October 5 after cancelling the dentist ---"
echo ""
run_check "Conflict resolved" "Brussels" \
    "$BINARY" check 2026-10-05

# --- Final list ---

echo "--- Final schedule ---"
echo ""
run_check "Final list" "Hawaii Vacation" \
    "$BINARY" list

# ============================================
# Results
# ============================================

if [ "$FAILURES" -gt 0 ]; then
    printf "${RED}SMOKE TEST FAILED: %d check(s) did not match expected output${NC}\n" "$FAILURES"
    exit 1
else
    printf "${GREEN}SMOKE TEST PASSED${NC}\n"
fi
