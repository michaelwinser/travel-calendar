#!/bin/sh
# Run a command inside a nono sandbox with project-specific capabilities.
#
# Usage:
#   scripts/sandbox.sh claude                    # Interactive Claude Code session
#   scripts/sandbox.sh ./dev ci                  # CI pipeline
#   scripts/sandbox.sh ./travel list             # CLI command
#   TRAVEL_TEST_MODE=true scripts/sandbox.sh ... # With test auth
#
# The sandbox provides:
#   - Read+write: project directory, app data, Go module cache
#   - Read-only:  Go toolchain, homebrew, system paths
#   - Network:    allowed (for go modules, docker, etc.)
#   - Port 3001:  bound (for dev server)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Check nono is available
if ! command -v nono >/dev/null 2>&1; then
    echo "nono not found — running unsandboxed"
    exec "$@"
fi

# Pre-resolve GOROOT so Go works inside the sandbox
if command -v go >/dev/null 2>&1 && [ -z "$GOROOT" ]; then
    export GOROOT="$(go env GOROOT)"
fi

# Determine if interactive (Claude Code, shell)
EXEC_FLAG=""
case "$1" in
    claude|bash|zsh|sh|vim|nvim)
        EXEC_FLAG="--exec"
        ;;
esac

exec nono run \
    --profile claude-code \
    --allow "$PROJECT_DIR" \
    --allow "$HOME/.config/travel" \
    --allow "$HOME/go" \
    --read "$HOME/sdk" \
    --allow-bind 3001 \
    $EXEC_FLAG \
    -- "$@"
