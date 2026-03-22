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

**Suggestion:** See #4 below — an installable `ab` binary would eliminate this entirely. **RESOLVED** in `cc92587`.

### ~~4. appbase should ship an installable `ab` CLI binary~~ RESOLVED

Resolved in appbase commit `cc92587` (renamed to `appbase` in `034d989`). Install with `go install github.com/michaelwinser/appbase/cmd/appbase@latest`. The `./tc` script should be updated to use this instead of sourcing shell scripts.

The following was the original proposal:

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

### 5. Tag releases and register with Go module proxy

**Context:** Consumer apps currently need `GONOSUMCHECK` and `GONOSUMDB` env var overrides to `go get` appbase, because the module isn't indexed by the Go module proxy (proxy.golang.org). This is fragile and non-standard.

**Fix:**
1. Tag a release: `git tag v0.1.0 && git push origin v0.1.0`
2. Trigger proxy indexing: `GOPROXY=https://proxy.golang.org go get github.com/michaelwinser/appbase@v0.1.0`
3. After that, all consumers can `go get` normally with no env var overrides

Semantic version tags also give consumers stable dependency points and make `go get -u` safer.

### 6. E2E shell script testing needs a way to create sessions without a browser

**Context:** The Go test harness works great for API-level tests — it creates sessions directly via `testSessions.Create()`. But for shell script e2e tests (testing the actual CLI binary against a running server, simulating real user workflows), there's no way to authenticate without a browser.

The current auth flow is browser-only: `travel login` opens a browser → Google OAuth → session stored in keychain. None of that works in an automated script.

**What's needed:** A way for e2e test scripts to obtain a valid session. Options in order of preference:

1. **Dev auth mode in appbase** — e.g., `AUTH_MODE=dev` that accepts a fixed test token or auto-creates a session for a configured email. Clean, no raw SQL, works across SQLite and Firestore. Could be restricted to `APP_ENV=local`.

2. **`ab test-session` command** — A CLI command that creates a session and prints the cookie value. Consumer apps call it in their test scripts. Keeps the test helper in appbase rather than every consumer reimplementing it.

3. **Direct SQLite seeding** — The e2e script inserts a row into the `sessions` table via `sqlite3`. Pragmatic but couples the test to the SQLite schema, won't work with Firestore, and breaks if the session table structure changes.

**Update (2026-03-22):** appbase `164bbe5` added `appbase test-session` and `AUTH_MODE=dev`. Then `7c68301`/`cda4f8f` added `appbase test-login`/`test-logout`. All resolved — see #7 below.

### ~~7. `appbase test-login` — seed both database and keychain for CLI e2e tests~~ RESOLVED

Resolved in appbase commits `7c68301` and `cda4f8f`. The `appbase test-login` command creates a session in the database AND stores it in the OS keychain, and `appbase test-logout` cleans up. Smoke test uses this successfully — no platform-specific keychain code needed in consumer apps.

The following was the original request:

### 7. `appbase test-login` — seed both database and keychain for CLI e2e tests

**Context:** The `appbase test-session` command creates a session in the database and prints the cookie value — great for curl-based tests. But when testing the actual CLI binary (e.g., `travel add`, `travel list`), the CLI reads its session from the OS keychain via `appcli.AuthenticatedClient()`, not from a cookie header.

To test the CLI end-to-end, you currently need to:
1. Create a session in the DB (`appbase test-session`)
2. Manually seed the keychain with the session ID and server URL (`security add-generic-password` on macOS)
3. Clean up the keychain after the test

This is fragile, platform-specific, and clutter that every consumer app would need to reimplement.

**Proposal:** `appbase test-login --server URL --app NAME [--email EMAIL]`

This would:
1. Create a session in the local database (like `test-session`)
2. Store the session ID and server URL in the keychain (like `login` does after OAuth)

Consumer smoke tests become clean CLI usage with no platform-specific code:
```sh
./travel serve &
appbase test-login --server http://localhost:3000 --app travel-calendar
./travel add "Trip" --from 2026-04-01 --loc Paris --type travel
./travel list
./travel check 2026-04-01
appbase logout --app travel-calendar   # cleanup
```

A corresponding `appbase logout --app NAME` (or extending the existing logout) would clean up.

## Notes

(none yet)
