# Review: Container-Based Development Environment

**Plan**: `docs/plans/1-container-development.md`
**Reviewer**: Applying `.claude/reviewer.md` criteria

---

## Alignment with Project Principles

### CLAUDE.md Compliance

| Principle | Status | Notes |
|-----------|--------|-------|
| Component boundaries | ✓ | Infra is separate, each service isolated |
| Plan document required | ✓ | Multi-component → plan created |
| Commit format | N/A | Will follow `chore(infra): ...` |

### PROJECT_MAP.md Compliance

| Check | Status | Notes |
|-------|--------|-------|
| Uses correct terminology | ✓ | "backend", "frontend", "mcp-server" |
| Maintains data flow | ✓ | Same architecture, just containerized |
| Updates map if needed | ⚠️ | Should add infra section to PROJECT_MAP.md |

### ARCHITECTURE.md Compliance

| Component | Status | Notes |
|-----------|--------|-------|
| backend | ✓ | No architecture changes, just env vars |
| frontend | ✓ | No architecture changes |
| mcp-server | ✓ | No architecture changes |
| shared | ✓ | No changes needed |

---

## Engineering Best Practices

### Simplicity ✓

- Uses standard Docker Compose (not Kubernetes)
- Single Dockerfile.dev for all services
- Minimal configuration needed

### Maintainability ✓

- Follows common Docker patterns
- Named volumes for node_modules (standard practice)
- Clear separation of dev vs production concerns

### Performance Considerations ⚠️

**Concern**: Bind mounts can be slow on macOS (Docker Desktop)

**Mitigation**: Plan already addresses this with named volumes for node_modules. May need to add polling for file watchers.

### Security ✓

- No credentials in docker-compose.yml
- SQLite in bind-mounted volume (data persists)

---

## Scope Creep Check

| Check | Status |
|-------|--------|
| Only requested features | ✓ |
| No "while we're at it" | ✓ |
| No production deployment | ✓ (explicitly excluded) |

---

## Concerns

### 1. Missing devcontainer support

**Impact**: VS Code users could benefit from devcontainer.json for full IDE integration.

**Suggestion**: Consider adding `.devcontainer/devcontainer.json` as a follow-up enhancement, not required for MVP.

**Recommendation**: Note as future enhancement, don't add to scope now.

### 2. MCP Server stdio complexity

**Impact**: MCP uses stdio, which doesn't work well with `docker compose`. Claude Desktop expects to launch the server directly.

**Suggestion**: The plan correctly puts mcp-server in a profile (`--profile mcp`) and notes it may need to run on host. This is acceptable.

**Recommendation**: Document clearly that MCP server may run on host for Claude Desktop integration.

### 3. Initial pnpm-lock.yaml

**Impact**: `pnpm install --frozen-lockfile` will fail if no lock file exists.

**Suggestion**: Update Dockerfile.dev to handle missing lock file gracefully, or ensure lock file is committed.

**Recommendation**: Add step to create initial lock file before first container build.

---

## Questions for User

1. **macOS vs Linux**: Are you primarily on macOS? This affects volume performance tuning.

2. **IDE integration**: Do you want VS Code devcontainer support now or later?

3. **MCP Server location**: Acceptable that MCP server may run on host (not in container) for Claude Desktop?

---

## Recommendation

**PROCEED WITH MINOR CHANGES**

The plan is well-structured and follows project principles. Minor adjustments:

1. Add note about creating `pnpm-lock.yaml` before first build
2. Clarify MCP server host-vs-container tradeoff in docs
3. Add infra section to PROJECT_MAP.md after implementation

---

## Checklist for Execution

- [ ] Verify no existing Docker files conflict
- [ ] Create lock file if needed (`pnpm install` on host once, or in CI)
- [ ] Test on target OS (macOS/Linux)
- [ ] Update PROJECT_MAP.md with infra component after completion
