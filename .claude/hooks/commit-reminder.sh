#!/bin/bash
# Commit Agent Reminder Hook
# Reminds Claude to use Commit Agent when git commit is attempted
#
# Hook input (JSON via stdin):
#   tool_name   - Name of tool being called
#   tool_input  - Tool parameters (for Bash, includes the command)

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')
TOOL_INPUT=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Only check Bash tool calls
if [ "$TOOL_NAME" != "Bash" ]; then
    exit 0
fi

# Check if this looks like a git commit command
if echo "$TOOL_INPUT" | grep -qE '^git commit|&& git commit| git commit'; then
    cat <<'EOF'

⚠️  COMMIT AGENT REQUIRED

You are about to run `git commit` directly. Per CLAUDE.md:

> **CRITICAL: Only the Commit Agent may execute `git commit` or `git push` commands.**

Please spawn the Commit Agent instead:
- Task: "Commit the current changes"
- Agent: commit

The Commit Agent ensures:
✓ Pre-commit tests run
✓ Code Review Agent is invoked
✓ Tools Agent reviews session
✓ Session Summary is updated

If you are the Commit Agent, proceed with the commit.

EOF
fi
