# PRD Template

PRDs in this project are structured to enable automatic test generation. Every use case becomes a test.

---

# Feature: {Feature Name}

## Overview

{2-3 sentence description of the feature}

## User Value

{Why does this feature matter? What problem does it solve?}

---

## Use Cases

### [UC-001] {Use Case Title}

**Actor**: {User | LLM | System}

**Preconditions**:
- {State that must exist before this use case}

**Steps**:
1. {Action 1}
2. {Action 2}
3. {Action 3}

**Expected Result**:
- {Observable outcome 1}
- {Observable outcome 2}

**CLI Test** (for automated testing):
```bash
# Setup
TRIP_ID=$(travel trips create --name "Test" --purpose vacation --start 2025-01-01 --end 2025-01-05 --json | jq -r '.id')

# Action
travel items add $TRIP_ID flight --from EWR --to LAX --date 2025-01-01

# Verify
travel trips get $TRIP_ID --json | jq -e '.items | length == 1'

# Cleanup
travel trips delete $TRIP_ID
```

---

### [UC-002] {Another Use Case}

...

---

## API Endpoints

| Method | Endpoint | Description | Use Cases |
|--------|----------|-------------|-----------|
| POST | `/api/trips` | Create trip | UC-001 |
| GET | `/api/trips/:id` | Get trip with items | UC-001, UC-002 |

---

## MCP Tools

| Tool | Description | Use Cases |
|------|-------------|-----------|
| `create_trip` | Create a new trip | UC-001 |
| `get_trip` | Get trip details | UC-002 |

---

## UI Components

| Component | Views | Use Cases |
|-----------|-------|-----------|
| TripCard | List view | UC-003 |
| TripDetail | Full page | UC-001, UC-002 |

---

## Acceptance Criteria

- [ ] All use cases pass as CLI tests
- [ ] API endpoints documented in OpenAPI spec
- [ ] MCP tools tested with inspector
- [ ] UI components have visual tests

---

## Out of Scope

- {Explicitly excluded functionality}
