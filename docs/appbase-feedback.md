# appbase Feedback & Issues

Running log of friction, bugs, and feature requests encountered while building travel-calendar v2 on appbase. To be filed as GitHub issues at the right time.

---

## Issues

### ~~1. `./ab lint-api` fails for consumer apps — uses stdout redirection instead of temp config~~ RESOLVED

Fixed upstream in appbase commit `22e42d2`. Same sed approach we used in `./tc`.

## Feature Requests

### ~~1. OpenAPI codegen / CLI-through-API pattern~~ RESOLVED

Resolved in appbase commit `1b72cb3` (2026-03-21). The `todo-api` example provides the full pattern: OpenAPI spec → oapi-codegen → generated server interface + client → CLI uses client. travel-calendar v2 now follows this pattern.

### ~~2. Noisy startup logs for CLI usage~~ RESOLVED

Fixed upstream in appbase commit `22e42d2`. New `Config.Quiet` flag suppresses startup logs. Use `Quiet: !appcli.IsServeCommand` in setup to silence logs for non-serve CLI commands.

### 3. Consumer apps can't `go run ../appbase/cmd/secret` — cross-module restriction

**Context:** The scaffold-app skill's `./tc` template uses `go run "$APPBASE_DIR/cmd/secret"` for secret management and `_load_secrets`. This fails with "directory outside main module or its selected dependencies" because Go won't build source from a different module.

**Workaround in travel-calendar:** Build and install the binary once (`go build -o ~/go/bin/appbase-secret ./cmd/secret` from within appbase), then call the installed binary from `./tc`. The `_ensure_secret_bin` helper handles this automatically.

**Suggestion:** See #4 below — an installable `ab` binary would eliminate this entirely.

### 4. appbase should ship an installable `ab` CLI binary

**The problem:** Building a consumer `./tc` script that uses appbase's shared capabilities requires significant gymnastics:

1. **Deploy functions** — Must source `deploy.sh` from the sibling `../appbase/` directory, but `deploy.sh` resolves paths relative to `SCRIPT_DIR`, so consumer scripts must override `DEPLOY_DIR` before sourcing. Without this, `deploy.sh` looks for `config.sh` in the consumer's (non-existent) `deploy/` directory.

2. **Secret management** — The `cmd/secret` binary lives in the appbase module. Go's module system prevents `go run ../appbase/cmd/secret` from a different module. Workaround: build the binary into `~/go/bin/` on first use and cache it. Fragile — the cached binary can go stale when appbase is updated.

3. **Lint-api** — The logic was reimplemented in `./tc` because (a) `./ab lint-api` has a bug (issue #1), and (b) even if fixed, `./ab` commands read config from appbase's own `app.json`, not the consumer's.

4. **Identity confusion** — `./ab secret` reads the project name from appbase's `app.json`, not the calling consumer's. So consumer scripts can't delegate to `./ab` directly — they need to extract the project name themselves and pass it explicitly.

5. **Assumes sibling directory layout** — Everything depends on `../appbase` existing at a known relative path. Fine for this workstation but breaks in CI, other dev machines, or if the repos aren't siblings.

**The proposal:** Ship `ab` as an installable Go binary (`go install github.com/michaelwinser/appbase/cmd/ab@latest`).

The `ab` binary would:
- Work from any directory — reads `app.json`/`app.yaml` from cwd (or `--project-dir`)
- Handle all shared operations: `ab secret`, `ab codegen`, `ab lint-api`, `ab deploy`, `ab provision`
- Embed deploy scripts or rewrite the key ones in Go
- Eliminate the sibling-directory assumption
- Self-update via `go install`

Consumer `./tc` scripts would shrink to app-specific commands only:
```sh
case "$command" in
    build)   go build -o travel . ;;
    test)    go test -v ./... ;;
    serve)   ab secret env | . /dev/stdin && go run . serve ;;
    codegen) ab codegen ;;
    ci)      ab lint && ab lint-api && go build ./... && go test ./... ;;
    deploy)  ab deploy ;;
    *)       ab "$command" "$@" ;;  # delegate everything else
esac
```

This is probably a significant effort, but the current approach of shell-script sourcing and cross-module gymnastics won't scale as more apps are built on appbase.

## Notes

(none yet)
