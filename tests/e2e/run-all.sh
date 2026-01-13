#!/bin/bash
# E2E Test Runner
# Runs all shell script tests in tests/e2e/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# Counters
passed=0
failed=0
skipped=0

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Travel Calendar E2E Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if API is running
if ! curl -s "${TRAVEL_API_URL:-http://localhost:3000}/health" > /dev/null 2>&1; then
  echo -e "${YELLOW}Warning: API not running at ${TRAVEL_API_URL:-http://localhost:3000}${NC}"
  echo "Start the API with: pnpm dev"
  echo ""
  exit 1
fi

# Find and run all test scripts
for test_file in "$SCRIPT_DIR"/test-*.sh; do
  if [[ ! -f "$test_file" ]]; then
    continue
  fi

  test_name=$(basename "$test_file" .sh)

  # Check for skip marker
  if head -5 "$test_file" | grep -q "# SKIP"; then
    echo -e "${YELLOW}○${NC} $test_name (skipped)"
    ((skipped++))
    continue
  fi

  # Run test
  echo -n "  Running $test_name... "

  if bash "$test_file" > /tmp/test-output.txt 2>&1; then
    echo -e "${GREEN}✓${NC}"
    ((passed++))
  else
    echo -e "${RED}✗${NC}"
    ((failed++))
    echo ""
    echo "  Output:"
    sed 's/^/    /' /tmp/test-output.txt
    echo ""
  fi
done

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  Results: ${GREEN}$passed passed${NC}, ${RED}$failed failed${NC}, ${YELLOW}$skipped skipped${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Exit with failure if any tests failed
[[ $failed -eq 0 ]]
