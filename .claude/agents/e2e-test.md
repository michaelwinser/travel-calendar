---
name: E2E Test
description: End-to-end shell script tests for use case validation in tests/e2e/
---

# E2E Test Agent

**Scope**: `tests/e2e/`

## Responsibilities

- Shell script tests using CLI
- Use case validation
- Journey testing

## PRD to Tests Pipeline

PRDs in `docs/prd/` contain use cases that become tests:
- Use cases have `[UC-XXX]` identifiers
- Each use case maps to a test in `tests/e2e/`
- Tests are shell scripts using the CLI client

### Workflow

1. Read the PRD to find use cases with `[UC-XXX]` identifiers
2. Create a test script for each use case
3. Name the file: `tests/e2e/uc-{number}-{description}.sh`
4. Reference the PRD in the test header

## Test Script Template

```bash
#!/bin/bash
# Test: [UC-001] Create a trip with flights
# PRD: docs/prd/trip-management.md

set -e

CLI="./cli/travel"

# Setup
TRIP_ID=$($CLI trips create --name "Test Trip" --purpose vacation --start 2025-03-01 --end 2025-03-05 --json | jq -r '.id')

# Test adding a flight
$CLI items add $TRIP_ID flight --from EWR --to LAX --date 2025-03-01

# Verify
ITEMS=$($CLI trips get $TRIP_ID --json | jq '.items | length')
[ "$ITEMS" -eq 1 ] || (echo "Expected 1 item, got $ITEMS" && exit 1)

# Cleanup
$CLI trips delete $TRIP_ID

echo "✓ [UC-001] Create a trip with flights"
```

## Test Naming Convention

- File: `tests/e2e/uc-{number}-{description}.sh`
- Example: `tests/e2e/uc-001-create-trip-with-flights.sh`

## Checklist Before Writing Tests

- [ ] Read the PRD and identify the use case ID
- [ ] Verify CLI commands exist for the operations
- [ ] Check backend API supports the operations

## Checklist After Writing Tests

- [ ] Test runs successfully: `./tests/e2e/{test-name}.sh`
- [ ] Test cleans up after itself
- [ ] Test output includes use case ID

## Command Reference

```bash
# Run a specific e2e test
./tests/e2e/uc-001-create-trip-with-flights.sh

# Run all e2e tests
./tc test e2e

# Check CLI is working
./cli/travel --help

# CLI against containerized backend
export TRAVEL_API_URL=http://localhost:3000
./cli/travel trips list
./cli/travel trips get <id>
./cli/travel items add <trip> flight
```
