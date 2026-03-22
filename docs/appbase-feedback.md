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

### 3. Consumer apps can't `go run ../appbase/cmd/secret` — cross-module restriction

**Context:** The scaffold-app skill's `./tc` template uses `go run "$APPBASE_DIR/cmd/secret"` for secret management and `_load_secrets`. This fails with "directory outside main module or its selected dependencies" because Go won't build source from a different module.

**Workaround in travel-calendar:** Build and install the binary once (`go build -o ~/go/bin/appbase-secret ./cmd/secret` from within appbase), then call the installed binary from `./tc`. The `_ensure_secret_bin` helper handles this automatically.

**Suggestion:** Either the scaffold-app template should use this install-on-first-use pattern, or appbase should provide an `./ab install-tools` command that installs CLI utilities to GOPATH/bin.

## Notes

(none yet)
