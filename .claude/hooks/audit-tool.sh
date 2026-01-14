#!/bin/bash
# Project-level tool audit hook
# Logs all tool calls to .claude/audit.jsonl for Tools Agent analysis
#
# Hook input (JSON via stdin):
#   session_id  - Session identifier
#   tool_name   - Name of tool being called
#   tool_input  - Tool parameters

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AUDIT_LOG="$SCRIPT_DIR/../audit.jsonl"

# Read hook input from stdin and append to audit log with timestamp
jq -c '{
  timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
  session_id: .session_id,
  tool: .tool_name,
  input: .tool_input
}' >> "$AUDIT_LOG"
