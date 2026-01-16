#!/bin/bash
# Check Todos Complete Hook (PostToolUse)
# Detects when all todos are marked complete and reminds Claude to invoke Session Summary Agent
#
# Hook input (JSON via stdin):
#   tool_name   - Name of tool being called (TodoWrite)
#   tool_input  - Tool parameters (includes the todos array)
#   tool_output - Result of the tool call

INPUT=$(cat)
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')

# Only check TodoWrite tool calls
if [ "$TOOL_NAME" != "TodoWrite" ]; then
    exit 0
fi

# Extract the todos array from tool_input
TODOS=$(echo "$INPUT" | jq -r '.tool_input.todos // empty')

# Check if todos array exists and is not empty
if [ -z "$TODOS" ] || [ "$TODOS" = "null" ] || [ "$TODOS" = "[]" ]; then
    # Empty or missing todos - no reminder needed
    exit 0
fi

# Count total todos and completed todos
TOTAL_COUNT=$(echo "$TODOS" | jq 'length')
COMPLETED_COUNT=$(echo "$TODOS" | jq '[.[] | select(.status == "completed")] | length')

# Check if there's at least one todo and all are completed
if [ "$TOTAL_COUNT" -gt 0 ] && [ "$TOTAL_COUNT" -eq "$COMPLETED_COUNT" ]; then
    cat <<'EOF'

ALL TODOS COMPLETE

All tasks in your todo list have been marked as completed.

Consider invoking the Session Summary Agent to:
- Update the running summary in blog/.current.md
- Capture this session's progress before it's lost
- Potentially finalize a blog entry if this session is wrapping up

To invoke, use: Task(subagent_type="Session Summary") or read .claude/agents/session-summary.md

EOF
fi
