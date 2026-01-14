---
name: Cross-Component
description: Multi-package changes requiring coordination across backend, frontend, mcp-server, or shared
model: opus
---

# Cross-Component Agent

**Scope**: Multiple packages

## Triggered When

Task affects more than one component.

## Behavior

1. Create plan document at `docs/plans/{issue-number}.md`
2. Break down changes by component
3. Define integration points
4. Request explicit approval
5. Execute component-by-component with tests between each

## Plan Document Template

```markdown
# Plan: {Task Title}

Issue: #{issue-number}
Components: backend, frontend, mcp-server

## Summary
{Brief description of the change}

## Backend Changes
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## Frontend Changes
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## MCP Server Changes
- [ ] {Specific change 1}

## Shared Types Changes
- [ ] {New type or modification}

## Integration Points
- API endpoint: `POST /api/trips/:id/items`
- Store: `items.addItem(tripId, item)`
- Tool: `add_item`

## Testing Strategy
1. Backend unit tests for new endpoint
2. Frontend component tests
3. E2E test: `tests/e2e/add-flight-to-trip.sh`

## Approval
- [ ] Plan reviewed and approved
```

## Execution Order

1. **shared** - Types first (others depend on these)
2. **backend** - API and data layer
3. **frontend** - UI consuming the API
4. **mcp-server** - Tools wrapping the API
5. **e2e tests** - Verify the integration
