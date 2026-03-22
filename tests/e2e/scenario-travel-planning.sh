#!/bin/sh
# E2E test: Travel planning scenario
#
# Simulates a user planning trips, checking for conflicts, and managing activities.
# Runs against a real server with SQLite, using curl for API calls.
#
# Usage: ./tests/e2e/scenario-travel-planning.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Config ---
PORT=3199
DB_PATH="$PROJECT_DIR/data/e2e-test.db"
BASE_URL="http://localhost:$PORT"
SESSION_ID="e2e-test-session-$(date +%s)"
COOKIE="app_session=$SESSION_ID"
SERVER_PID=""

# Colors
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    GREEN='' RED='' BLUE='' NC=''
fi

pass() { printf "${GREEN}  PASS${NC} %s\n" "$1"; }
fail() { printf "${RED}  FAIL${NC} %s\n" "$1"; FAILURES=$((FAILURES + 1)); }
step() { printf "${BLUE}▶${NC} %s\n" "$1"; }

FAILURES=0
TESTS=0

assert_status() {
    expected="$1"
    actual="$2"
    label="$3"
    TESTS=$((TESTS + 1))
    if [ "$actual" = "$expected" ]; then
        pass "$label"
    else
        fail "$label (expected $expected, got $actual)"
    fi
}

assert_contains() {
    body="$1"
    needle="$2"
    label="$3"
    TESTS=$((TESTS + 1))
    if echo "$body" | grep -q "$needle"; then
        pass "$label"
    else
        fail "$label (expected body to contain '$needle')"
    fi
}

assert_not_contains() {
    body="$1"
    needle="$2"
    label="$3"
    TESTS=$((TESTS + 1))
    if echo "$body" | grep -q "$needle"; then
        fail "$label (expected body NOT to contain '$needle')"
    else
        pass "$label"
    fi
}

# --- Setup ---

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -f "$DB_PATH"
}
trap cleanup EXIT

step "Building travel binary..."
(cd "$PROJECT_DIR" && go build -o travel .)

step "Starting server on port $PORT..."
rm -f "$DB_PATH"
mkdir -p "$(dirname "$DB_PATH")"
STORE_TYPE=sqlite SQLITE_DB_PATH="$DB_PATH" PORT="$PORT" \
    "$PROJECT_DIR/travel" serve > /dev/null 2>&1 &
SERVER_PID=$!

# Wait for server to be ready
for i in 1 2 3 4 5 6 7 8 9 10; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        break
    fi
    if [ "$i" = "10" ]; then
        echo "Server failed to start"
        exit 1
    fi
    sleep 0.5
done

step "Seeding test session in SQLite..."
EXPIRES=$(date -u -v+1H "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+1 hour" "+%Y-%m-%dT%H:%M:%SZ")
NOW=$(date -u "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u "+%Y-%m-%dT%H:%M:%SZ")
sqlite3 "$DB_PATH" "INSERT INTO sessions (id, user_id, email, expires_at, created_at) VALUES ('$SESSION_ID', 'e2e-user', 'e2e@test.com', '$EXPIRES', '$NOW');"

# ============================================
# Scenario: Planning a month of travel
# ============================================

echo ""
step "Scenario: Planning a month of travel"
echo ""

# --- 1. Start with a clean slate ---

step "1. Verify empty activity list"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET /api/activities returns 200"
assert_contains "$BODY" '\[\]' "Activity list is empty"

# --- 2. Plan a conference trip ---

step "2. Create a conference trip to Brussels"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" \
    -H "Content-Type: application/json" \
    -d '{"title":"European Summit","type":"conference","startDate":"2026-10-04","endDate":"2026-10-07","location":"Brussels"}' \
    "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "201" "$STATUS" "POST /api/activities returns 201"
assert_contains "$BODY" '"European Summit"' "Response contains title"
assert_contains "$BODY" '"Brussels"' "Response contains location"
SUMMIT_ID=$(echo "$BODY" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')

# --- 3. Add a vacation ---

step "3. Add a vacation in Hawaii"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" \
    -H "Content-Type: application/json" \
    -d '{"title":"Hawaii Vacation","type":"vacation","startDate":"2026-10-15","endDate":"2026-10-22","location":"Maui"}' \
    "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
assert_status "201" "$STATUS" "Create Hawaii vacation"

# --- 4. Add a local commitment that conflicts with the conference ---

step "4. Add dentist appointment during the conference"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" \
    -H "Content-Type: application/json" \
    -d '{"title":"Dentist Appointment","type":"commitment","startDate":"2026-10-05","location":"Home"}' \
    "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
assert_status "201" "$STATUS" "Create dentist appointment"

# --- 5. Check for conflict on Oct 5 ---

step "5. Check Oct 5 for conflicts"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities/check/2026-10-05")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET check date returns 200"
assert_contains "$BODY" '"hasConflict":true' "Conflict detected on Oct 5"
assert_contains "$BODY" '"European Summit"' "Summit appears in check"
assert_contains "$BODY" '"Dentist Appointment"' "Dentist appears in check"

# --- 6. Check a non-conflict date ---

step "6. Check Oct 6 (no conflict)"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities/check/2026-10-06")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET check date returns 200"
assert_contains "$BODY" '"hasConflict":false' "No conflict on Oct 6"
assert_contains "$BODY" '"Brussels"' "Location is Brussels"

# --- 7. Check a date with no activities ---

step "7. Check Nov 1 (no activities)"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities/check/2026-11-01")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET check date returns 200"
assert_contains "$BODY" '"location":"Home"' "Default location is Home"
assert_contains "$BODY" '"hasConflict":false' "No conflict"

# --- 8. List all activities ---

step "8. List all activities"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET /api/activities returns 200"
assert_contains "$BODY" '"European Summit"' "Summit in list"
assert_contains "$BODY" '"Hawaii Vacation"' "Hawaii in list"
assert_contains "$BODY" '"Dentist Appointment"' "Dentist in list"

# --- 9. Filter by month ---

step "9. Filter activities to October"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities?month=2026-10")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "Month filter returns 200"
assert_contains "$BODY" '"European Summit"' "Summit in October"
assert_contains "$BODY" '"Hawaii Vacation"' "Hawaii in October"

# --- 10. Get a specific activity ---

step "10. Get the summit by ID"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities/$SUMMIT_ID")
STATUS=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
assert_status "200" "$STATUS" "GET by ID returns 200"
assert_contains "$BODY" '"European Summit"' "Correct activity returned"

# --- 11. Delete the dentist (resolve the conflict) ---

step "11. Delete dentist appointment to resolve conflict"
DENTIST_BODY=$(curl -s -b "$COOKIE" "$BASE_URL/api/activities")
DENTIST_ID=$(echo "$DENTIST_BODY" | sed -n 's/.*"id":"\([^"]*\)"[^}]*"title":"Dentist Appointment".*/\1/p')
if [ -z "$DENTIST_ID" ]; then
    # Try alternate JSON field ordering
    DENTIST_ID=$(echo "$DENTIST_BODY" | python3 -c "
import sys, json
for a in json.load(sys.stdin):
    if a['title'] == 'Dentist Appointment':
        print(a['id'])
        break
" 2>/dev/null || true)
fi

if [ -n "$DENTIST_ID" ]; then
    RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" -X DELETE "$BASE_URL/api/activities/$DENTIST_ID")
    STATUS=$(echo "$RESP" | tail -1)
    assert_status "200" "$STATUS" "DELETE dentist returns 200"

    # Verify conflict is resolved
    RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" "$BASE_URL/api/activities/check/2026-10-05")
    BODY=$(echo "$RESP" | sed '$d')
    assert_contains "$BODY" '"hasConflict":false' "Conflict resolved after delete"
else
    fail "Could not find Dentist Appointment ID for deletion"
fi

# --- 12. Auth enforcement ---

step "12. Verify unauthenticated request is rejected"
RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
assert_status "401" "$STATUS" "No cookie returns 401"

# --- 13. Validation ---

step "13. Verify validation errors"
RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" \
    -H "Content-Type: application/json" \
    -d '{"title":"","type":"stay","startDate":"2026-10-01"}' \
    "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
assert_status "400" "$STATUS" "Empty title returns 400"

RESP=$(curl -s -w "\n%{http_code}" -b "$COOKIE" \
    -H "Content-Type: application/json" \
    -d '{"title":"Bad","type":"stay","startDate":"2026-10-10","endDate":"2026-10-05"}' \
    "$BASE_URL/api/activities")
STATUS=$(echo "$RESP" | tail -1)
assert_status "400" "$STATUS" "End before start returns 400"

# ============================================
# Results
# ============================================

echo ""
if [ "$FAILURES" -gt 0 ]; then
    printf "${RED}FAILED${NC}: %d of %d tests failed\n" "$FAILURES" "$TESTS"
    exit 1
else
    printf "${GREEN}PASSED${NC}: All %d tests passed\n" "$TESTS"
fi
