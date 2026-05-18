# nortezh-cli — v0.1 Design

**Date:** 2026-05-18
**Status:** Approved (sections 1–4)
**Scope:** First usable release of `ntzh`, a Go CLI for the `nortezh-backend` PaaS API. (Repository: `nortezh-cli`. Binary: `ntzh`.)

## 1. Purpose

`nortezh-cli` is a command-line client for the Nortezh deployment platform (Kubernetes-based PaaS: projects, clusters, deployments, routes, domains, disks, postgres, pull secrets, service accounts). v0.1 targets the **day-to-day deploy flow** for an authenticated user: log in, pick a project, list/inspect deployments, deploy a new image, roll back, and tail revision logs.

## 2. Non-goals (v0.1)

- Cluster, route, domain, disk, postgres, pullsecret, serviceaccount, billing, payment, phone commands.
- Shell completion, self-update, color/TTY-aware tables.
- Windows-specific path handling beyond `os.UserConfigDir()`.
- Server-side token revocation on `logout`.
- Service-account-key auth (alternative path; deferred).
- Backend integration tests (covered later, behind a build tag).

## 3. Layout

```
nortezh-cli/
├── cmd/ntzh/main.go                 # cobra root, wires subcommands (binary: ntzh)
├── internal/
│   ├── cli/
│   │   ├── root.go                  # global flags: --server, --project, --output, --debug
│   │   ├── login.go                 # login, logout, whoami
│   │   ├── project.go               # project list
│   │   └── deployment.go            # deployment list/get/deploy/rollback/logs
│   ├── api/
│   │   ├── client.go                # Invoke(ctx, method, body, out) error
│   │   ├── errors.go                # *api.Error, ErrUnauthenticated
│   │   ├── deployment.go            # typed wrappers
│   │   ├── project.go               # typed wrappers
│   │   └── types.go                 # Project, Deployment, LogLine
│   ├── auth/
│   │   ├── loopback.go              # local http server, browser launch, state check
│   │   ├── creds.go                 # Creds interface; Bearer + ServiceAccount impls
│   │   └── store.go                 # load/save credentials.json
│   ├── config/config.go             # ~/.config/ntzh/{config,credentials}.json
│   └── output/
│       ├── printer.go               # Printer interface, table|json
│       └── tables.go                # Headers()/Row() per type
├── go.mod
├── Makefile
└── README.md
```

## 4. Commands (v0.1)

| Command | Backend call(s) | Notes |
|---|---|---|
| `ntzh login` | `GET /user/auth/?state&callback` (existing Google relay) | Loopback callback; 7-day token. |
| `ntzh logout` | — | Wipes `credentials.json`. |
| `ntzh whoami` | `auth.me` | Prints user identity. |
| `ntzh project list` | `project.list` | Table or JSON. |
| `ntzh deployment list --project <name>` | `deployment.list` | `--project` required. |
| `ntzh deployment get <name> --project <p>` | `deployment.get` | `--project` required. |
| `ntzh deployment deploy <name> --project <p> --image <ref>` | `deployment.deploy` | `--project` required. |
| `ntzh deployment rollback <name> --project <p> --to <revision>` | `deployment.rollback` | `--project` required. |
| `ntzh deployment logs <name> --project <p> [--revision N]` | `deployment.logRevision` | `--project` required. Streams to stdout. |

**Global flags:** `--server`, `--project`, `--output table|json` (default `table`), `--debug`.

**No stored project state.** Every command that operates on a project requires `--project <name>` (or env `NTZH_PROJECT`). There is no `project use`, no `default_project` in config. Project name is resolved to ID on each invocation via `project.list`. Rationale: explicit > implicit; avoids "wrong-project" foot-guns when scripting or switching contexts.

## 5. Auth flow

**No backend changes required.** The existing `nortezh-backend` already exposes (a) a Google-OAuth callback-relay endpoint (`/user/auth/`) that mints a random bearer token after Google sign-in, and (b) an arpc auth middleware that accepts `Authorization: Bearer <token>` on every protected route (`api/handler.go:246`). The CLI just acts as a callback target.

### 5.1 `ntzh login` (interactive, default)

1. CLI generates a random `state` (32 bytes, base64url).
2. CLI starts an HTTP server on `127.0.0.1:<random port>` with a single `/callback` endpoint.
3. Opens the user's browser to:
   ```
   {server}/user/auth/?state=<state>&callback=http://127.0.0.1:<port>/callback
   ```
4. The user completes Google sign-in. The backend mints a 32-byte random token, stores `sha256(token)` in `user_auth_tokens` with a 7-day expiry, then redirects to:
   ```
   http://127.0.0.1:<port>/callback?state=<state>&code=<token>
   ```
5. CLI's loopback handler verifies `state` matches, then saves `{ token: <code>, expires_at: now()+7d }` to `~/.config/ntzh/credentials.json` (`0600`).
6. Browser sees a small "You can close this window. Logged in as <email>." page; CLI shuts the loopback server down.

`expires_at` is a local hint only — the server is the source of truth. On 401 the CLI prints `Error: not logged in. Run 'ntzh login'.` and exits 1. **There is no refresh-and-retry path:** the backend does not issue refresh tokens (the minted token is opaque, 7-day hard expiry).

### 5.2 `ntzh login --service-account <email> --key-file <path>` (headless, optional)

For CI / SSH where a browser isn't available. The backend already supports basic-auth service accounts (`api/auth/serviceaccount.go`).

- CLI reads the key from `<path>` (or `--key -` for stdin), saves `{ email, key }` to `credentials.json` (`0600`).
- API client sends `Authorization: Basic base64(email:key)` instead of `Bearer`.
- No expiry; rotated via the existing `serviceaccount.createKey` / `serviceaccount.deleteKey` endpoints.

Both modes write to the same `credentials.json`; the file's shape carries which mode is active.

### 5.3 `ntzh logout`

Wipes `credentials.json`. No server-side revocation in v0.1. (The backend exposes `/user/auth/signout?token=...` for the interactive mode — wire it as a follow-up.)

## 6. API client

Single entry point in `internal/api`:

```go
type Client struct {
    BaseURL    string         // {server}/user
    HTTPClient *http.Client
    Creds      auth.Creds     // bearer token OR service-account email+key
    Debug      bool
}

func (c *Client) Invoke(ctx context.Context, method string, body, out any) error
```

- `method` is the arpc route name, e.g. `"deployment.list"`.
- Always `POST` to `BaseURL + "/" + method`, `Content-Type: application/json`.
- `body == nil` → sends `{}`. `out == nil` → result is discarded.
- `Authorization` header injected by `Creds`: `Bearer <token>` for interactive login, `Basic base64(email:key)` for service-account mode.

**Envelope unwrap:**

```go
type envelope struct {
    OK     bool             `json:"ok"`
    Result json.RawMessage  `json:"result"`
    Error  *apiError        `json:"error"`
}
```

- `ok=true` → `json.Unmarshal(env.Result, out)`.
- `ok=false` and `error.code == "UNAUTHORIZED"` → return `ErrUnauthenticated`.
- `ok=false` (other) → return `*api.Error{Code, Message, HTTPStatus}`.
- Non-2xx HTTP → return `*api.Error{Code: "http_error", HTTPStatus: ...}`.

No refresh-and-retry: the backend issues opaque 7-day tokens with no refresh, so 401 always means "log in again".

**Typed wrappers** in `internal/api/deployment.go` and `project.go` — only the v0.1 calls. Each is a 3-line shim around `Invoke`.

**Types** (`internal/api/types.go`) — only fields v0.1 commands render:

```go
type Project    struct { ID, Name, SID string; CreatedAt time.Time }
type Deployment struct { ID, Name, ProjectID, Image, Status string; Revision int; UpdatedAt time.Time }
type LogLine    struct { Timestamp time.Time; Stream, Line string }
```

**Debug mode** (`--debug`): logs method, request body, response status, response body to stderr, truncated to 4 KB per direction. **Never** logs the `Authorization` header.

**User-facing errors:** CLI renders `*api.Error` as `Error: {Code}: {Message}` to stderr with exit 1. `ErrUnauthenticated` → `Error: not logged in. Run 'ntzh login'.`

## 7. Config & precedence

- `~/.config/ntzh/config.json` (`0644`):
  ```json
  { "server": "https://api.nortezh.com" }
  ```
- `~/.config/ntzh/credentials.json` (`0600`), one of two shapes:
  ```json
  // interactive (Google) login
  { "kind": "bearer", "token": "<opaque>", "expires_at": "2026-05-25T07:30:00Z" }
  ```
  ```json
  // service account
  { "kind": "service_account", "email": "ci@example.com", "key": "<opaque>" }
  ```

Path resolution via `os.UserConfigDir()` (honors `XDG_CONFIG_HOME`).

**Precedence:** flag > env (`NTZH_SERVER`, `NTZH_PROJECT`, `NTZH_CONFIG_DIR`) > config file > built-in default.

## 8. Output

`internal/output` exposes:

```go
type Printer interface {
    Print(v any) error
    PrintList(items any) error
}
```

- `--output table` (default): `text/tabwriter`, no color. Each typed result implements `Headers() []string` and `Row() []string`.
- `--output json`: `json.MarshalIndent(v, "", "  ")`.
- `deployment logs` bypasses the printer and streams lines to stdout verbatim.

## 9. Testing

| Package | What to cover |
|---|---|
| `internal/api` | success, error envelope, `UNAUTHORIZED` → `ErrUnauthenticated`, non-2xx HTTP, Bearer vs Basic header selection, debug redaction of `Authorization`. Uses `httptest.Server`. |
| `internal/auth` | loopback picks a free port, state mismatch rejected with 400, callback parses `code`, browser-open is injected (`OpenBrowser func(url string) error`), credentials round-trip both kinds (`bearer`, `service_account`). |
| `internal/config` | round-trip read/write; `credentials.json` is `0600`; env/flag/file precedence. |
| `internal/cli` | per-subcommand. Use a fake `api.Client` via an interface, not the struct. Instantiate `cobra.Command` per test — no globals. Cover happy path + one error path per command. |

No backend integration tests in v0.1. Future: `_integration_test.go` behind a `//go:build integration` tag, hitting a staging backend with a fixed service-account credential.

`go test ./...` is the standard run.

## 10. Tooling

- Go 1.25.
- `spf13/cobra` for command tree.
- `Makefile` targets: `test`, `build`, `lint` (golangci-lint), `install`.
- No release pipeline in v0.1.

## 11. Risks & open questions

1. **7-day hard token expiry, no refresh.** Users will hit re-login every week. Acceptable for v0.1; revisit if it bites. Backend issuing longer-lived or refreshable tokens is the proper fix.
2. **Open-redirect on backend `/user/auth/?callback=...`.** The `callback` query param is not allowlisted server-side. Loopback works because only the CLI process holds the port, but a future hardening pass should restrict allowed callback prefixes (incl. `http://127.0.0.1:*`).
3. **arpc envelope schema.** Spec assumes `{ok, result, error: {code, message}}` and the `UNAUTHORIZED` code string from `api/handler.go:244`. Confirm shape against `backend/pkg/api` before wiring the client.
4. **`deployment.logRevision` payload shape.** Currently assumed to return an array; if it streams or paginates, `LogLine` and the logs command need revisiting.
5. **Headless login UX.** Service-account mode requires the user to create a key in the web UI first. Document this in `README` and surface a helpful error when interactive login is attempted in a no-browser environment.
