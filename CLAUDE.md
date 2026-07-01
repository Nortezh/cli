# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make test                              # go test ./...  (all tests; ~51 across 6 packages)
make build                             # builds ./ntzh
make install                           # go install ./cmd/ntzh
make lint                              # golangci-lint run

go test ./internal/api/...             # single package
go test -run TestInvoke ./internal/api # single test
go build ./...                         # compile check
```

No integration tests yet — everything is `httptest.Server`-based. There is no live-backend smoke test wired up.

## Architecture

`ntzh` is a thin CLI over the `nortezh-backend` arpc HTTP API. The backend is not modified by this repo; the CLI just speaks its envelope protocol.

Module path: `github.com/nortezh/cli` (installable via `go install github.com/nortezh/cli/cmd/ntzh@latest`).

### Request flow
`cmd/ntzh/main.go` → cobra command in `internal/cli/` → typed wrapper in `internal/api/{project,deployment}.go` → `api.Client.Invoke(ctx, method, body, out)` → POST `BaseURL/<method>` with `{}` body if nil → unwrap `{ok, result, error}` envelope.

`api.Client.Invoke` is the single shim every backend call goes through. Typed wrappers are intentionally 3-line passthroughs — do not add logic there. Add new endpoints by writing one wrapper + one cobra subcommand.

Envelope handling: non-2xx → `*Error{Code:"http_error"}`; `ok:false` with `code:"UNAUTHORIZED"` → sentinel `ErrUnauthenticated` (mapped to "not logged in. Run 'ntzh login'" by `FormatCLIError` in `cmd/ntzh/main.go`).

### Auth (`internal/auth/`)
Two credential shapes, both implement `Creds.Apply(*http.Request)`:
- **Bearer** — written by `ntzh login` via Google relay + loopback HTTP server. Backend route is `GET /user/auth/?state=&callback=http://127.0.0.1:<port>/cb`. 7-day hard expiry, **no refresh** — on 401 the user re-runs `ntzh login`.
- **ServiceAccount** — `Authorization: Basic <email:key>` for CI.

Credentials live at `~/.config/ntzh/credentials.json` (mode 0600). `store.go` enforces the mode.

### Config (`internal/config/`)
`~/.config/ntzh/config.json` (0644) holds `{server}`. Resolution precedence for both server and project: **flag > env > file > default**.

Env vars: `NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_LOCATION`, `NTZH_CONFIG_DIR` (the last overrides the whole config dir; used by tests).

**No stored project state.** Every project-scoped command requires `--project` (or `NTZH_PROJECT`); the CLI calls `project.list` on each invocation and resolves the user-supplied identifier to a **slug** by matching against Name, Slug (`no`), then ID — see `resolveProjectSlug` in `internal/cli/context.go`. All deployment API calls send the **slug** in the `project` field (not the internal ID). Do not introduce `default_project` or `project use`.

### Backend payload shapes
- All request/response JSON uses **camelCase** (`createdAt`, `actionStatus`, `lastDeployedAt`, `minReplicas`, `perPage`). Snake_case caused silent unmarshal failures (zero-value rows) before — keep struct tags camelCase.
- Pagination on `deployment.list`: `{"paginate":{"page":1,"perPage":40},"project":"<slug>"}`.
- Deployment-scoped methods (`deployment.get`, `deployment.deploy`, `deployment.rollback`, `deployment.logRevision`) all take `{project, location, name, ...}`. `--location` is auto-resolved from `deployment.list` if omitted (see `resolveLocation`); `NTZH_LOCATION` short-circuits the lookup.
- `deployment.deploy` accepts partial-update fields beyond `image`: `addEnv` (merge), `removeEnv` (delete keys), and pointer-shaped `port`, `protocol`, `internal`, `minReplica`, `maxReplica`. The CLI surfaces these via `api.DeployOptions` and `--set-env / --remove-env / --port / --protocol / --internal / --min-replica / --max-replica`. Only send pointer fields when the user passed the flag (use `cmd.Flags().Changed(...)` — never send zero values, they would overwrite real state).
- `envGroups` (project-scoped env group names) uses **replace** semantics, not merge: `nil` (field absent/`null`) leaves the linked groups unchanged, a non-nil slice replaces them, and an empty non-nil slice (`[]`) clears all links. Surfaced via `--env-group` (repeatable, replace) on both `deploy` and `create`, plus `--clear-env-groups` (bool, sends `[]`) on `deploy`; `cleanStrings` in `deployment.go` trims/drops blanks. On `deploy` `--env-group` is only sent when `Changed`, so omitting both flags preserves; the two flags are mutually exclusive. The backend merges the linked groups' vars into the revision spec env (inline `env` overrides); it does **not** write them into the deployment's `env` column.
- `deployment.get` returns `env` as `map[string]string` (deployment **inline** env only — group-provided vars live in the linked groups, not here) plus `envGroups` as `[]string` (ordered linked group names). `DeploymentDetail.Env` / `.EnvGroups` carry them; the printer emits an `ENV_GROUPS` row (comma-joined) then one `ENV:KEY  VALUE` row per key (sorted) after the main fields.

### Output (`internal/output/`)
`Printer` interface with three implementations selected by `--output`: `toon` (**default**, compact/agent-friendly — `toon.go`), `table` (`printer.go`), and `json`. Per-resource headers/rows live in `tables.go`; the TOON and table printers share them via `tableRows`/the `*DetailRows` funcs. TOON lists append a `count:` line and render empty as `<resource>: 0 found`; `output.Hints(w, format, ...)` appends AXI `help[]` next-step lines (no-op for `json`). Errors print to **stdout** via `FormatCLIError` (structured `error:` / `help:` lines), not stderr.

### CLI wiring (`internal/cli/`)
Global flags `--server`, `--project`, `--output`, `--debug` are defined on the root command. `context.go` builds an `api.Client` (+ `Creds` from the store) per invocation using the resolved config. `--debug` logs request/response to stderr with the `Authorization` header redacted.

Deployment subcommands: `list`, `get`, `deploy`, `rollback`, `revisions` (aliased as `logs`). `revisions` calls `deployment.logRevision` and returns **revision history**, not pod log lines.

## Conventions specific to this repo

- Go 1.26. Cobra is the only direct dependency.
- `internal/` packages **are** used here (this is a CLI, not a backend service — the user's global "no internal/" rule applies to the haabiz backend, not this repo). Keep the existing layout.
- Backend method naming is `{module}.{action}` (e.g. `deployment.deploy`, `project.list`). Do not prepend `/api/v1`.
- All cobra commands return errors up to `main.go`, which calls `FormatCLIError` for user-friendly messages and exits non-zero. Don't `os.Exit` from subcommands.
- Backend payload shapes are assumed from the design spec; field-name drift is a known risk — verify against the real backend before changing types.
- Commit style: **Conventional Commits** (`<type>(<scope>): <summary>`, imperative, lowercase, no period, ≤72 chars). Types: feat, fix, refactor, docs, test, chore, perf, build, ci, style, revert.
- Do not put personal/machine-specific paths or identifiers into tracked files — this repo is public.

## Reference docs

- `README.md` — user-facing command reference
