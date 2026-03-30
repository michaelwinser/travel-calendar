# Sandboxing Claude Code Projects with nono

A guide for applying capability-based sandboxing to any project using [nono](https://github.com/always-further/nono). Based on the approach developed for travel-calendar.

## Concept

nono enforces OS-level filesystem and network restrictions on processes. When Claude Code runs inside a nono sandbox, it can only access explicitly allowed paths — even if it tries to read secrets, modify other projects, or access cloud credentials.

The sandbox has two roles:
1. **During development**: Protect the system from the AI agent. The agent can edit project files but can't read `~/.ssh`, `~/.aws`, or `~/.config/gcloud`.
2. **For agent isolation**: Each independent agent session gets minimum permissions for its task. A review agent gets read-only access. A build agent gets write access to build outputs only.

## The ./sandbox Script

Every project gets a top-level `./sandbox` script (parallel to `./dev`). It wraps any command in a nono sandbox with project-specific capabilities.

```bash
#!/bin/sh
# Run a command inside a nono sandbox with project-specific capabilities.
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Graceful degradation: run unsandboxed if nono isn't installed
if ! command -v nono >/dev/null 2>&1; then
    echo "nono not found — running unsandboxed"
    exec "$@"
fi

# --- Pre-sandbox environment setup ---
# Tools that resolve paths at startup need help before entering the sandbox.

# Go: resolve GOROOT before sandbox (Go's trimmed binary can't follow symlinks under sandbox)
if command -v go >/dev/null 2>&1 && [ -z "$GOROOT" ]; then
    export GOROOT="$(go env GOROOT)"
fi

# PATH: ensure toolchain directories are on PATH (nono doesn't source shell profiles)
for p in /opt/homebrew/bin /usr/local/bin /usr/local/go/bin "$HOME/go/bin"; do
    case ":$PATH:" in
        *":$p:"*) ;;
        *) [ -d "$p" ] && export PATH="$p:$PATH" ;;
    esac
done

# Go binary: resolve its directory for the sandbox allowlist
GO_BIN_DIR=""
if command -v go >/dev/null 2>&1; then
    GO_BIN_DIR="$(dirname "$(command -v go)")"
fi

# Docker: nono blocks ~/.docker (contains registry credentials) — correct behavior.
# Create a local empty config to suppress Docker CLI warnings.
DOCKER_DIR="$PROJECT_DIR/.docker"
mkdir -p "$DOCKER_DIR" 2>/dev/null
[ -f "$DOCKER_DIR/config.json" ] || echo '{}' > "$DOCKER_DIR/config.json"
export DOCKER_CONFIG="$DOCKER_DIR"

# --- Interactive detection ---
EXEC_FLAG=""
case "$1" in
    claude|bash|zsh|sh|vim|nvim)
        EXEC_FLAG="--exec"  # Preserve TTY for interactive apps
        ;;
esac

# --- Build sandbox flags ---
GO_FLAG=""
if [ -n "$GO_BIN_DIR" ] && [ "$GO_BIN_DIR" != "/usr/bin" ]; then
    GO_FLAG="--read $GO_BIN_DIR"
fi

# --- Launch sandbox ---
exec nono run \
    --profile claude-code \
    --allow "$PROJECT_DIR" \
    $GO_FLAG \
    $EXEC_FLAG \
    -- "$@"
```

## What to Customize Per Project

The script above is the skeleton. Each project adds its own capabilities:

### Data directories
```bash
--allow "$HOME/.config/myapp"     # App data (SQLite, config)
--allow "$HOME/go"                # Go module cache (if Go project)
--read "$HOME/sdk"                # Go SDK installations
```

### Port binding
```bash
--allow-bind 3001                 # Dev server port
--allow-bind 5173                 # Vite dev server
```

### Sibling dependencies (temporary)
```bash
# If your project sources files from a sibling directory, add read access.
# File an issue to eliminate the dependency.
if [ -d "$PROJECT_DIR/../other-project" ]; then
    OTHER_DIR="$(cd "$PROJECT_DIR/../other-project" && pwd)"
    EXTRA_FLAGS="$EXTRA_FLAGS --read $OTHER_DIR"
fi
```

## What the Sandbox Blocks (and Should Block)

nono's `claude-code` profile + sensitive path policy blocks:

| Path | Why | Workaround |
|------|-----|-----------|
| `~/.ssh` | SSH keys | None needed — agents shouldn't have SSH access |
| `~/.aws` | AWS credentials | Use `--env-credential` for specific keys if needed |
| `~/.config/gcloud` | Google Cloud auth | Deploy outside sandbox (see below) |
| `~/.docker` | Registry credentials | Local empty config via `DOCKER_CONFIG` |
| `~/.gnupg` | GPG keys | None needed |

**Deploy must run outside the sandbox.** Cloud deployments need credentials that the sandbox correctly blocks. Add a guard:

```bash
# In your deploy command:
if [ -n "$NONO_CAP_FILE" ]; then
    echo "Error: deploy cannot run inside a nono sandbox."
    echo "Run ./dev deploy directly, outside the sandbox."
    return 1
fi
```

## Gotchas Discovered

### 1. Go binary resolution
Go installed via homebrew is a "trimmed" binary that resolves GOROOT by following symlinks. The sandbox blocks this resolution. **Fix**: set `GOROOT` before entering the sandbox.

### 2. PATH isn't inherited from shell profile
nono starts a clean process — `~/.zshrc` and `~/.bash_profile` aren't sourced. **Fix**: explicitly add toolchain directories to PATH in the script.

### 3. Docker config warning
Docker CLI always tries `~/.docker/config.json` first. nono blocks it (sensitive path). Docker works fine without it but prints a warning. **Fix**: create a local empty `config.json` and set `DOCKER_CONFIG`.

### 4. `--read` doesn't override sensitive paths
`nono run --read ~/.docker` is still denied — security policy overrides explicit flags. This is correct behavior. Don't fight it.

### 5. Wails/CGO on Linux needs specific packages
Wails v2 requires `webkit2gtk-4.0` which is only on Ubuntu 22.04. Ubuntu 24.04 has `4.1`. Use `ubuntu-22.04` runners for Linux desktop builds.

## .gitignore Additions

```gitignore
# Sandbox artifacts
.docker/
```

## CLAUDE.md Documentation

Add to your project's CLAUDE.md:

```markdown
## Sandboxing

All development sessions should run inside a nono sandbox:

\```bash
./sandbox claude          # Interactive Claude Code session
./sandbox ./dev ci        # CI pipeline
./sandbox ./travel list   # CLI command
\```

Deploy runs outside the sandbox — it needs cloud credentials:
\```bash
./dev deploy              # Run directly, not via ./sandbox
\```
```

## Agent-Specific Profiles (Future)

For forge-based coordination where separate Claude Code sessions handle different tasks, each agent type gets a constrained sandbox:

```bash
# Review agent: read-only
nono run --profile claude-code --read "$PROJECT_DIR" -- claude -p "Review PR #42"

# Build agent: write to build outputs only
nono run --profile claude-code --read "$PROJECT_DIR" --allow "$PROJECT_DIR/dist" -- claude -p "Build release"

# Full development agent: read-write
./sandbox claude
```

## Should This Be in appbase?

### What belongs in appbase
- **`appbase sandbox-template`**: Print the `./sandbox` script to stdout (like `appbase dev-template`). Consumer apps customize it. Keeps the pattern consistent across projects.
- **Deploy guard**: `appbase deploy` could check `NONO_CAP_FILE` internally and refuse to deploy from a sandbox.

### What doesn't belong in appbase
- **The sandbox script itself**: It's project-specific (different ports, data dirs, dependencies). appbase provides the template; the project customizes it.
- **nono installation or profiles**: nono is a system tool, not an app dependency. The sandbox script gracefully degrades if nono isn't installed.

### Where this fits in an appbase split
If appbase splits into parts:
- **Dev tooling** (dev-template, sandbox-template, codegen, lint-api): The sandbox template belongs here.
- **Runtime framework** (auth, store, server, sessions): No sandbox concern.
- **Deploy tooling** (provision, deploy, secrets): Deploy guard belongs here.

The sandbox is purely a dev-time concern — it protects the developer's machine during AI-assisted development. It has no runtime footprint.
