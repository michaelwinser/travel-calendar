# Claude Code Configuration

This directory contains configuration for Claude Code agents working on this project.

## Files

| File | Purpose |
|------|---------|
| `settings.json` | Claude Code settings and hooks |
| `agents.md` | Component-specific agent definitions |
| `reviewer.md` | Pushback/review criteria for evaluating requests |
| `hooks/` | Scripts that run on Claude Code events |

## Agents

See `agents.md` for full definitions. Summary:

| Agent | Scope | Purpose |
|-------|-------|---------|
| Task Router | All | Analyzes tasks, routes to component agents |
| Backend | `packages/backend/` | REST API, services, entities |
| Frontend | `packages/frontend/` | Svelte components, stores |
| MCP Server | `packages/mcp-server/` | MCP tools, LLM formatting |
| Shared | `packages/shared/` | TypeScript types only |
| Cross-Component | Multiple packages | Plans for multi-component changes |
| E2E Test | `tests/e2e/` | Shell script tests |
| Reviewer | All | Evaluates requests against principles |

## Hooks

### `postConversation`

Saves conversation metadata to `.claude/conversations/` for later analysis.

Captured data:
- Timestamp
- Git context (recent commits, modified files)
- Conversation ID
- Working directory

## Conversation History

Conversation logs are saved to `.claude/conversations/`. These can be used to:
- Analyze patterns in development
- Track what changes were made and why
- Debug issues by reviewing past sessions
- Learn from effective prompts

**Note**: These files may contain sensitive information. Consider adding `.claude/conversations/` to `.gitignore` if committing this repo publicly.

## Settings

`settings.json` configures:
- **hooks**: Scripts to run on events
- **contextFiles**: Files automatically loaded for context

## Validation

Run `pnpm validate:map` to check `PROJECT_MAP.md` against the codebase.
