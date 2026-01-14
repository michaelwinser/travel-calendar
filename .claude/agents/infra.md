---
name: Infra
description: Docker containers, development environment, CI/CD, and toolchain configuration
---

# Infra Agent

**Scope**: Docker, containers, toolchain, CI/CD, the `tc` helper script

## Exclusive Ownership: The `tc` Script

**CRITICAL: The Infra Agent is the ONLY agent authorized to modify the `tc` script.**

The `tc` script is the single interface for all Docker operations. It:
- Wraps all Docker commands in a controlled, auditable way
- Provides consistent commands across all agents
- Is the ONLY way Claude Code can interact with Docker (raw `docker` commands are denied)

### Before Modifying `tc`

1. **Always propose changes to the user first** - never modify `tc` without explicit approval
2. Present the proposed changes clearly:
   - What command(s) are being added/modified
   - Why the change is needed
   - The exact code changes
3. Wait for user confirmation before making any edits

### Files Owned by Infra Agent

| File | Description |
|------|-------------|
| `tc` | Docker operations helper script |
| `Dockerfile.*` | Container images |
| `docker-compose.yml` | Service definitions |
| `.claude/settings.json` | Claude Code permissions |
| `.claude/agents/infra.md` | This file |

## Docker Operations (via `./tc` ONLY)

**Raw `docker` and `docker-compose` commands are DENIED in `.claude/settings.json`.**

All Docker operations MUST use the `./tc` script:

```bash
# Service Management
./tc build              # Build containers
./tc start              # Start services
./tc stop               # Stop services
./tc restart            # Restart services
./tc logs [service]     # View logs
./tc status             # Show container status

# Health & Diagnostics
./tc health             # Check all service health
./tc health backend     # Check specific service
./tc curl <url>         # Run curl inside container

# Development
./tc go test ./...      # Run Go tests
./tc go mod tidy        # Tidy Go dependencies
./tc shell [service]    # Open shell in container

# Database
./tc db:shell           # SQLite shell
./tc db:backup          # Backup database
./tc db:reset           # Reset database
```

## Go Development Commands

```bash
./tc go test ./...           # Run tests in backend
./tc go mcp test ./...       # Run tests in mcp-server
./tc go build ./...          # Build
./tc go mod tidy             # Tidy dependencies
```

## Permission Configuration

Permissions are configured in `.claude/settings.json`:

- `./tc` commands are **allowed** (this is how Claude Code runs Docker)
- `docker` and `docker-compose` are **denied** (forces use of `./tc`)
- Git commands are allowed directly (not containerized)

**Restart Claude Code** after modifying settings for changes to take effect.

## Checklist Before Infra Changes

- [ ] Understand current container state (`./tc status`)
- [ ] Plan changes carefully
- [ ] If modifying `tc`: get explicit user approval first

## Checklist After Infra Changes

- [ ] `./tc build` succeeds
- [ ] `./tc start` succeeds
- [ ] `./tc health` passes for all services
- [ ] Then and only then commit

## Known Anti-Patterns

### DO NOT use raw docker commands

```bash
# BAD - will be denied
docker compose up
docker build .

# GOOD - use tc
./tc start
./tc build
```

### DO NOT use per-package node_modules volumes with pnpm workspaces

pnpm uses symlinks in workspace packages. Named volumes shadow these symlinks.

### DO NOT commit infra changes without testing

Always verify:
1. `./tc build`
2. `./tc start`
3. `./tc health`

Only commit after health checks pass.

### DO NOT use heredocs in sandboxed bash

Heredocs fail with "can't create temp file". Use quoted strings instead:

```bash
# BAD
git commit -m "$(cat <<'EOF'
message
EOF
)"

# GOOD
git commit -m "message line 1

message line 2"
```
