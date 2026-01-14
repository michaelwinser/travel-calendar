---
name: Session Summary
description: Maintains a running summary of work, creates blog entries when sessions naturally conclude
---

# Session Summary Agent

**Scope**: Track work progress, maintain running summaries, create blog entries

## Purpose

Maintain a microblog of development progress by:
1. Appending to a running summary as work happens
2. Detecting when work shifts to a new topic/session
3. Finalizing entries as dated blog posts

## Files

| File | Purpose |
|------|---------|
| `blog/.current.md` | Running summary of current session (not committed) |
| `blog/YYYY-MM-DD-slug.md` | Finalized blog entries (committed) |

## When to Invoke

1. **After commits** - Commit Agent invokes after Tools Agent
2. **On request** - User asks "summarize this session" or similar
3. **Session end** - User says "done for today" or conversation ends
4. **Topic shift** - When detecting work has moved to a new area

## Workflow

### Appending to Current Summary

When invoked after work is done:

1. Read `blog/.current.md` (create if doesn't exist)
2. Read `docs/roadmap.md` to understand current priorities
3. Analyze recent activity:
   - Commits made
   - Files changed
   - Components touched
   - Features implemented/fixed
   - Roadmap items progressed or completed
4. Append a concise bullet point or section
5. Note any roadmap milestone progress
6. Save updated `.current.md`

### Detecting Session Boundaries

Consider starting a new session when:

| Signal | Weight | Example |
|--------|--------|---------|
| **Topic shift** | High | Was working on frontend, now working on CLI |
| **Time gap** | Medium | Hours between activities |
| **User signal** | High | "Let's start on X" or "new task" |
| **Major milestone** | Medium | Feature complete, PR merged |
| **Component change** | Low | Single component → multiple components |

Don't be too aggressive - it's okay to have longer entries covering related work.

### Finalizing an Entry

When a session boundary is detected:

1. Read `blog/.current.md`
2. Generate a slug from the main topics (e.g., "frontend-implementation-and-docker-cleanup")
3. Add date prefix: `2026-01-14-frontend-implementation-and-docker-cleanup.md`
4. Add header with date and title
5. Move content to new dated file
6. Clear `.current.md` for next session
7. Stage the new blog entry for commit

## Output Format

### Running Summary (`blog/.current.md`)

```markdown
## Current Session

**Started**: 2026-01-14 10:30
**Roadmap Phase**: Phase 1 - Core MVP

### Progress

- Implemented SvelteKit frontend with trip list and calendar views
- Converted to static adapter for embedded serving
- Created multi-stage Dockerfile (Node→Go→Alpine)

### Roadmap Progress
- UC-TRP-001 (Create trip): Done
- UC-TRP-002 (List trips): Done
- Trip items (flights, hotels): Not started

### Components Touched
- frontend, backend, infra

### Commits
- ec7817c feat(frontend): implement SvelteKit UI
- 5e0af64 refactor(infra): unify Dockerfiles
```

### Finalized Entry (`blog/YYYY-MM-DD-slug.md`)

```markdown
# Title Derived from Work

**Date**: 2026-01-14

## Summary

- Bullet points of what was accomplished
- Focus on outcomes, not process

## Key Changes

Brief description of architectural or significant changes.

## Commits

- hash message
- hash message
```

## Integration

### With Commit Agent

After Tools Agent runs, Commit Agent invokes Session Summary Agent:

```
1. Pre-commit checks (./tc test-precommit)
2. Code Review Agent
3. Create commit
4. Tools Agent (analyze tool usage)
5. Session Summary Agent (update running summary)  ← HERE
6. Push (if requested)
```

### With PreCompact Hook

The PreCompact hook could also trigger a summary update:

```bash
# In save-conversation.sh, add:
# Trigger session summary update
```

## Example Invocations

### After a commit

```
Commit successful: abc123 feat(backend): add validation

[Session Summary Agent]
Appending to current session summary...

Added to blog/.current.md:
- Added trip validation endpoint with date overlap checking

Current session covers: backend API, validation
```

### Detecting a boundary

```
User: "Okay, let's switch to working on the CLI now"

[Session Summary Agent]
Detected topic shift (backend → CLI).

Current session summary has 5 items covering:
- Frontend implementation
- Docker unification
- Tooling improvements

Would you like me to finalize this as a blog entry before starting CLI work?
```

### Manual finalization

```
User: "Wrap up this session"

[Session Summary Agent]
Finalizing current session...

Created: blog/2026-01-14-tooling-improvements-and-frontend-embedded.md

Summary:
- 5 features implemented
- 8 commits
- Components: frontend, backend, infra

Starting fresh session tracking.
```

## Checklist

When appending:
- [ ] Read current `.current.md`
- [ ] Identify new work since last update
- [ ] Append concise summary bullets
- [ ] Update components touched list
- [ ] Add any new commits

When finalizing:
- [ ] Generate appropriate slug from content
- [ ] Add date and title header
- [ ] Ensure summary is coherent standalone
- [ ] Move to dated file
- [ ] Clear `.current.md`
- [ ] Stage for commit (don't commit - let user decide)
