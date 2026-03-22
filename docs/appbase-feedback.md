# appbase Feedback & Issues

Running log of friction, bugs, and feature requests encountered while building travel-calendar v2 on appbase. To be filed as GitHub issues at the right time.

---

## Issues

### 1. `./ab lint-api` fails for consumer apps — uses stdout redirection instead of temp config

**Context:** The `lint-api` command in `./ab` checks if generated code is up to date by running `oapi-codegen --config ... > tmpfile`. But `--config` writes to the `output:` path in the config file, not stdout. The `> tmpfile` redirection captures nothing (empty file), so the diff always fails.

**Fix applied locally in `./tc`:** Create temp copies of the config files with redirected output paths via `sed`, then diff against those. This should be fixed in `./ab lint-api` upstream.

## Feature Requests

### ~~1. OpenAPI codegen / CLI-through-API pattern~~ RESOLVED

Resolved in appbase commit `1b72cb3` (2026-03-21). The `todo-api` example provides the full pattern: OpenAPI spec → oapi-codegen → generated server interface + client → CLI uses client. travel-calendar v2 now follows this pattern.

### 2. Noisy startup logs for CLI usage

**Context:** When running CLI commands (e.g. `travel add "Trip" --from ...`), appbase prints log lines like "Using SQLite store" and "Running preflight check". These make sense for server startup but are noisy for quick CLI commands.

**Suggestion:** Suppress or reduce log verbosity for non-serve commands.

## Notes

(none yet)
