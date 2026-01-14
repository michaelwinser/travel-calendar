---
name: MCP Server
description: MCP tool definitions, resource providers, and LLM-friendly API integration in packages/mcp-server/
---

# MCP Server Agent (Go)

**Scope**: `packages/mcp-server/`

## Before Starting

1. **Check `docs/roadmap.md`** - understand current phase and priorities
2. **Read `packages/mcp-server/ARCHITECTURE.md`** - component patterns

## Responsibilities

- MCP tool definitions
- JSON-RPC 2.0 handler
- LLM-friendly markdown response formatting
- Backend API integration (HTTP client)

## Checklist Before Changes

- [ ] Read ARCHITECTURE.md
- [ ] Check existing tools in `internal/tools/`
- [ ] Check backend API supports needed operations
- [ ] Check formatters in `internal/formatter/`

## Checklist After Changes

- [ ] Run Go tests: `./tc exec sh -c "cd packages/mcp-server && go test ./..."`
- [ ] Test tools with curl
- [ ] Verify response formatting is LLM-friendly (markdown, not JSON)

## Forbidden

- Direct database access (use backend API)
- Business logic (belongs in backend)
- UI code
- Raw JSON responses (format as markdown)

## Command Reference

```bash
# Run tests
./tc exec sh -c "cd packages/mcp-server && go test ./..."

# List MCP tools
./tc curl mcp:3001/mcp -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Call a tool
./tc curl mcp:3001/mcp -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_trips","arguments":{}}}'

# Health check
./tc curl mcp:3001/health
```
