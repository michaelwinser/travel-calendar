---
name: MCP Server
description: MCP tool definitions, resource providers, and LLM-friendly API integration in packages/mcp-server/
---

# MCP Server Agent

**Scope**: `packages/mcp-server/`

## Before Starting

**Must read first**: `packages/mcp-server/ARCHITECTURE.md`

## Responsibilities

- MCP tool definitions
- MCP resource providers
- LLM-friendly response formatting
- Backend API integration

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check existing tools in `tools/`
- [ ] Check backend API supports needed operations
- [ ] Verify types in `shared/`

## Checklist After Changes

- [ ] Run `./tc exec pnpm test:mcp`
- [ ] Test with MCP inspector
- [ ] Verify response formatting is LLM-friendly

## Forbidden

- Direct database access
- Business logic
- UI code
- Raw JSON responses

## Command Reference

```bash
# Run tests
./tc exec pnpm test:mcp

# Run linting
./tc exec pnpm lint
```
