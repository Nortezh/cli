# nortezh-cli — Handoff

**Last touched:** 2026-05-18
**Branch:** `main` (12 commits, no remote configured)

## Status

v0.1 is implemented and green. `ntzh` binary builds, 51 tests pass across 6 packages, no integration against a live backend yet.

```
$ go test ./...      # 51 passed in 6 packages
$ go build ./...     # clean
$ go build -o ntzh ./cmd/ntzh && ./ntzh --help
```

## What ships in v0.1

| Command | Backend route | Notes |
|---|---|---|
| `ntzh login` | `GET /user/auth/?state&callback` (existing Google relay) | Loopback callback; 7-day token. |
| `ntzh login --service-account <email> --key-file <path>` | — | Headless via `Authorization: Basic`. |
| `ntzh logout` | — | Wipes credentials file. |
| `ntzh whoami` | `auth.me` | |
| `ntzh project list` | `project.list` | |
| `ntzh deployment list --project <p>` | `deployment.list` | |
| `ntzh deployment get <name> --project <p>` | `deployment.get` | |
| `ntzh deployment deploy <name> --project <p> --image <ref>` | `deployment.deploy` | |
| `ntzh deployment rollback <name> --project <p> --to <rev>` | `deployment.rollback` | |
| `ntzh deployment logs <name> --project <p> [--revision N]` | `deployment.logRevision` | |

Global flags: `--server`, `--project`, `--output table|json`, `--debug`.
Env: `NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_CONFIG_DIR`.
Files: `~/.config/ntzh/config.json` (0644), `~/.config/ntzh/credentials.json` (0600).

## Package layout

```
cmd/ntzh/main.go                     # entrypoint, FormatCLIError on error
internal/cli/                        # cobra commands, context, error formatter
internal/api/                        # Client.Invoke + typed wrappers (project, deployment)
internal/auth/                       # Creds (Bearer/ServiceAccount), store, loopback login
internal/config/                     # Dir/Load/Save, Resolve{Server,Project}
internal/output/                     # Printer interface (table + json)
```

## Key design decisions (full context in `docs/superpowers/specs/`)

1. **No backend changes needed.** `nortezh-backend` already has Bearer + Basic auth middleware (`api/handler.go:246`) and a callback-relay Google login (`api/auth/auth.go`). The CLI just acts as the callback target.
2. **7-day hard expiry, no refresh.** Backend doesn't issue refresh tokens. On 401 the CLI prints `Error: not logged in. Run 'ntzh login'.` and exits 1 — there is no refresh-and-retry path.
3. **No stored project state.** Every project-scoped command requires `--project` (resolved to ID via `project.list` on each invocation). No `project use`, no `default_project` in config.
4. **`api.Client.Invoke(ctx, method, body, out)`** is the single arpc shim. Typed wrappers in `internal/api/{project,deployment}.go` are 3-line shims around it.

## Reference docs (read these first next session)

- `docs/superpowers/specs/2026-05-18-nortezh-cli-design.md` — full v0.1 design
- `docs/superpowers/plans/2026-05-18-nortezh-cli-v0.1.md` — task-by-task plan (all completed)

## Known risks / deferred work

From spec §11:

1. **No end-to-end test against a real backend.** Everything is `httptest.Server`-based. First real run will probably surface envelope-field-name drift (e.g. `revision` vs `to`, `image_ref` vs `image`). Wire an integration test behind `//go:build integration` once a staging URL is available.
2. **Backend `/user/auth/?callback=...` has no callback allowlist.** Open-redirect on the backend side. Loopback works because only the CLI owns the port, but worth raising as a backend hardening item.
3. **`deployment.logRevision` payload shape is assumed.** Spec assumed `{items:[{timestamp, stream, line}]}`. If it streams or paginates, `LogLine` and the logs command both need work.
4. **Public client registration** (`ntzh-cli` as an OAuth-style client) — not strictly required since we use the Google relay, but document it if/when the backend later moves to proper OAuth.
5. **Headless login UX.** `ntzh login` without a browser will fail with the OS error from `exec.Command`. Could improve with a clearer "no browser; use --service-account" message.

## Suggested next moves (pick one)

- **Smoke against staging.** Point `--server` at a real Nortezh URL, do `ntzh login`, `ntzh project list`, fix any payload mismatches.
- **Add `--watch` to `deployment logs`** (poll vs stream — depends on whether the backend route is paginated).
- **Add `ntzh route`, `ntzh domain`, `ntzh disk`** — same pattern as deployment (typed wrapper + cobra subcommand). Spec §2 lists these as non-goals; lift once v0.1 is validated.
- **Release pipeline.** `goreleaser` config + GitHub Actions, signed checksums.
- **Shell completion.** Cobra generates this; wire `ntzh completion`.

## Repo hygiene notes

- No remote configured. If you `git push`, expect to set one up first.
- Commits use squash-merge friendly messages (`area: imperative`). 11 implementation commits + 3 docs commits.
- Co-author trailer is on every commit (`Claude Opus 4.7`).

## Quick verification on a fresh session

```bash
cd /Users/xkamail/workspaces/nortezh-cli
go test ./...                     # expect: 51 passed
go build -o ntzh ./cmd/ntzh
./ntzh --help                     # 5 subcommands listed
./ntzh deployment deploy --help   # confirms --image flag exists
```
