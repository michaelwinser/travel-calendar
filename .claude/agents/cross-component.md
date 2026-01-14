---
name: Cross-Component
description: Multi-package changes requiring coordination across backend, frontend, mcp-server, cli, or shared
model: opus
---

# Cross-Component Agent

**Scope**: Multiple packages

## Triggered When

Task affects more than one component.

## Behavior

1. **Check `docs/roadmap.md`** to understand current phase and priorities
2. Create plan document at `docs/plans/{issue-number}.md`
3. Break down changes by component
4. Define integration points
5. Note how this work aligns with roadmap priorities
6. Request explicit approval
7. Execute component-by-component with tests between each

## Plan Document Template

```markdown
# Plan: {Task Title}

Issue: #{issue-number}
Components: api, backend, cli, mcp-server, frontend
Roadmap Phase: {Current phase from docs/roadmap.md}
Roadmap Item: {Which feature/use case this addresses, if any}

## Summary
{Brief description of the change}

## OpenAPI Changes (if any)
- [ ] {Endpoint/schema modification}

## Backend Changes (Go)
- [ ] Regenerate types from OpenAPI
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## CLI Changes (Go)
- [ ] Regenerate client from OpenAPI
- [ ] {Specific change 1}

## MCP Server Changes (Go)
- [ ] {Specific change 1}

## Frontend Changes
- [ ] {Specific change 1}
- [ ] {Specific change 2}

## Shared Types Changes
- [ ] Regenerate from OpenAPI

## Integration Points
- API endpoint: `POST /api/trips/{tripId}/items`
- CLI command: `travel items add`
- MCP tool: `add_item`

## Testing Strategy
1. Backend Go tests
2. CLI build and manual test
3. MCP tool curl test
4. Frontend build
5. E2E test: `tests/e2e/uc-XXX-description.sh`

## Approval
- [ ] Plan reviewed and approved
```

## Execution Order

1. **api** - OpenAPI spec first (source of truth)
2. **shared** - TypeScript types (generated from OpenAPI)
3. **backend** - Go API and data layer (regenerate types)
4. **cli** - Go CLI (regenerate client)
5. **mcp-server** - Go MCP tools (update if needed)
6. **frontend** - UI consuming the API
7. **e2e tests** - Verify the integration

## OpenAPI-First Workflow

When adding or modifying API endpoints:

1. **Edit OpenAPI spec** at `packages/api/openapi.yaml`
2. **Validate** with `npx @redocly/cli lint packages/api/openapi.yaml`
3. **Regenerate backend types**
   ```bash
   ./tc exec sh -c "cd packages/backend && oapi-codegen -generate types,chi-server -package api ../api/openapi.yaml > internal/api/openapi.gen.go"
   ```
4. **Regenerate CLI client**
   ```bash
   cd packages/cli && oapi-codegen -generate types,client -package client ../api/openapi.yaml > internal/client/client.gen.go
   ```
5. **Regenerate TypeScript types**
   ```bash
   ./tc exec pnpm --filter @travel-calendar/shared generate
   ```
6. **Implement changes** in each component
7. **Test** at each step
