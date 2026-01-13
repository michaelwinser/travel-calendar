# Contributing Guidelines

This document defines engineering principles and workflows for this project.

## Development Environment

**All development is container-based.** You only need Docker installed - no Node.js, pnpm, or other tools on your host machine.

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (macOS/Windows) or Docker Engine (Linux)
- Git

### Quick Start

```bash
# Clone the repository
git clone git@github.com:michaelwinser/travel-calendar.git
cd travel-calendar

# Build and start all services
./tc build
./tc start

# Services available at:
# - Backend API: http://localhost:3000
# - Frontend UI: http://localhost:5173
# - MCP Server: http://localhost:3001 (with --mcp flag)
```

### Helper Script (`./tc`)

The `tc` script provides convenient commands for all common operations:

```bash
./tc start              # Start services (background)
./tc start --mcp        # Start with MCP server
./tc start --fg         # Start in foreground
./tc stop               # Stop all services
./tc restart            # Restart services
./tc logs               # Follow all logs
./tc logs backend       # Follow specific service
./tc status             # Show container status
./tc shell backend      # Shell into container
./tc test               # Run all tests
./tc test e2e           # Run E2E tests only
./tc lint               # Run linters
./tc db:reset           # Reset database
./tc db:backup          # Backup database
./tc db:restore <file>  # Restore from backup
./tc db:shell           # SQLite shell
./tc clean              # Remove containers
./tc clean volumes      # Remove containers + volumes
./tc help               # Show all commands
```

### Docker Commands

```bash
# Start services (foreground)
pnpm dev:docker
# Or directly: docker compose up

# Start services (background)
docker compose up -d

# Start with MCP server
pnpm dev:docker:mcp
# Or: docker compose --profile mcp up

# View logs
pnpm dev:docker:logs
# Or: docker compose logs -f backend

# Stop services
pnpm dev:docker:down
# Or: docker compose down

# Rebuild after dependency changes
pnpm dev:docker:build
# Or: docker compose build --no-cache

# Run tests in container
docker compose --profile test up

# Shell into a container
docker compose exec backend sh

# Run one-off command
docker compose exec backend pnpm test
```

### Hot Reload

Source code is mounted as volumes. Changes to files in `packages/*/src/` will trigger hot reload automatically.

**Note for macOS users**: If file watching seems slow, the `CHOKIDAR_USEPOLLING=true` environment variable is set to enable polling mode.

### Data Persistence

SQLite database is stored in `./data/travel.db`. This directory is mounted into the container and persists across restarts.

### Using the CLI

The CLI tool works against the containerized backend:

```bash
# Set the API URL (containers expose on localhost)
export TRAVEL_API_URL=http://localhost:3000

# Use the CLI
./cli/travel trips list
./cli/travel trips create --name "Test" --purpose work --start 2025-01-01 --end 2025-01-05
```

### When NOT to Use Docker

In rare cases, you may need to run services on the host:
- Debugging native dependencies
- Performance profiling
- IDE integration that requires local Node.js

To run without Docker:
```bash
# Install pnpm (requires Node.js 20+)
corepack enable

# Install dependencies
pnpm install

# Run services
pnpm dev
```

---

## Git Workflow

### Branch Strategy

```
main              # Protected, requires PR
  └── feat/42-add-expense-tracking    # Feature branches
  └── fix/43-trip-date-validation     # Bug fix branches
```

### Branch Naming

```
{type}/{issue-number}-{short-description}

Examples:
feat/42-add-expense-tracking
fix/43-trip-date-validation
refactor/44-simplify-item-types
docs/45-api-documentation
```

### Commit Messages

**Every commit must reference an issue.**

Format:
```
{type}({component}): {description} (#{issue})

[optional body]

[optional footer]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `refactor` - Code change that neither fixes a bug nor adds a feature
- `test` - Adding or updating tests
- `docs` - Documentation only
- `chore` - Maintenance tasks

Components:
- `backend` - packages/backend
- `frontend` - packages/frontend
- `mcp-server` - packages/mcp-server
- `shared` - packages/shared
- `infra` - CI/CD, configuration
- `cli` - CLI tool

Examples:
```
feat(backend): add expense entity and endpoints (#42)

fix(frontend): prevent ID-based lookup in TripCard (#43)

refactor(shared): split item types into discriminated union (#44)

test(e2e): add trip creation journey test (#45)
```

### Pull Requests

1. **Title** matches commit format: `feat(backend): add expense tracking (#42)`

2. **Description** includes:
   - Summary of changes
   - Link to issue
   - Testing done
   - Screenshots (for UI changes)

3. **Checklist**:
   - [ ] Tests pass
   - [ ] Linting passes
   - [ ] ARCHITECTURE.md followed
   - [ ] Types exported to shared (if new entities)
   - [ ] Documentation updated (if API changed)

4. **Review requirements**:
   - Single component: 1 approval
   - Cross-component: 2 approvals + plan document

## Issue Management

### Issue Templates

**Feature Request**:
```markdown
## Description
{What should be built}

## User Value
{Why this matters}

## Acceptance Criteria
- [ ] {Criterion 1}
- [ ] {Criterion 2}

## Components Affected
- [ ] backend
- [ ] frontend
- [ ] mcp-server
- [ ] shared
```

**Bug Report**:
```markdown
## Description
{What's broken}

## Steps to Reproduce
1. {Step 1}
2. {Step 2}

## Expected Behavior
{What should happen}

## Actual Behavior
{What actually happens}

## Component
{backend | frontend | mcp-server}
```

### Issue Labels

- `component:backend` - Backend changes needed
- `component:frontend` - Frontend changes needed
- `component:mcp-server` - MCP server changes needed
- `cross-component` - Affects multiple components (requires plan)
- `good-first-issue` - Good for new contributors
- `blocked` - Waiting on something

## Development Workflow

### Starting Work

```bash
# 1. Create issue (if doesn't exist)
gh issue create --title "Add expense tracking" --body "..."

# 2. Create branch
git checkout -b feat/42-add-expense-tracking

# 3. Read relevant ARCHITECTURE.md
cat packages/backend/ARCHITECTURE.md

# 4. Start development
pnpm dev
```

### During Development

```bash
# Run component tests frequently
pnpm test:backend
pnpm test:frontend

# Check linting
pnpm lint

# Check component boundaries
pnpm lint:boundaries
```

### Before Committing

```bash
# Run all checks
pnpm test
pnpm lint

# Stage changes
git add .

# Commit with issue reference
git commit -m "feat(backend): add expense entity (#42)"
```

### Creating PR

```bash
# Push branch
git push -u origin feat/42-add-expense-tracking

# Create PR
gh pr create --title "feat(backend): add expense entity (#42)" \
  --body "## Summary
Adds expense tracking to trips.

## Changes
- New Expense entity
- CRUD endpoints at /api/expenses
- Types exported to shared

## Testing
- Unit tests added
- E2E test: tests/e2e/expense-tracking.sh

Closes #42"
```

## Code Review Guidelines

### For Reviewers

1. **Check ARCHITECTURE.md compliance**
   - Does the change follow component patterns?
   - Are forbidden patterns avoided?

2. **Check component boundaries**
   - No cross-component imports (except shared)
   - Business logic in correct layer

3. **Check test coverage**
   - New functionality has tests
   - Bug fixes have regression tests

4. **Check types**
   - New entities have types in shared
   - Types are used consistently

### For Authors

1. **Self-review first**
   - Read your own diff before requesting review
   - Check ARCHITECTURE.md compliance

2. **Small PRs**
   - One logical change per PR
   - If it touches multiple components, use plan document

3. **Respond promptly**
   - Address feedback within 24 hours
   - Explain if you disagree with feedback

## Testing Requirements

### Unit Tests

- All services have unit tests
- All components have rendering tests
- Test file adjacent to source: `foo.ts` → `foo.test.ts`

### E2E Tests

- Every PRD use case has a CLI test
- Tests in `tests/e2e/`
- Tests are shell scripts using CLI

### Test Commands

```bash
pnpm test              # All tests
pnpm test:backend      # Backend unit tests
pnpm test:frontend     # Frontend component tests
pnpm test:mcp          # MCP server tests
pnpm test:e2e          # E2E shell script tests
```

## Documentation Requirements

### When to Update Docs

- **API changes**: Update OpenAPI spec
- **New MCP tools**: Update tool descriptions
- **New components**: Add to component directory README
- **Architecture changes**: Update relevant ARCHITECTURE.md

### Where Docs Live

- `CLAUDE.md` - Agent instructions (constitution)
- `CONTRIBUTING.md` - This file (human guidelines)
- `packages/*/ARCHITECTURE.md` - Component-specific rules
- `docs/prd/` - Feature requirements and use cases
- `docs/plans/` - Cross-component change plans
