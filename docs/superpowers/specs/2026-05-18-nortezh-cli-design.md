# nortezh-cli — v0.1 Design

**Date:** 2026-05-18
**Status:** Approved (sections 1–4)
**Scope:** First usable release of `nortezh`, a Go CLI for the `nortezh-backend` PaaS API.

## 1. Purpose

`nortezh-cli` is a command-line client for the Nortezh deployment platform (Kubernetes-based PaaS: projects, clusters, deployments, routes, domains, disks, postgres, pull secrets, service accounts). v0.1 targets the **day-to-day deploy flow** for an authenticated user: log in, pick a project, list/inspect deployments, deploy a new image, roll back, and tail revision logs.

## 2. Non-goals (v0.1)

- Cluster, route, domain, disk, postgres, pullsecret, serviceaccount, billing, payment, phone commands.
- Shell completion, self-update, color/TTY-aware tables.
- Windows-specific path handling beyond `os.UserConfigDir()`.
- Server-side token revocation on `logout`.
- Service-account-key auth (alternative path; deferred).
- Backend integration tests (the OAuth endpoints don't exist yet).

## 3. Layout

```
nortezh-cli/
├── cmd/nortezh/main.go              # cobra root, wires subcommands
├── internal/
│   ├── cli/
│   │   ├── root.go                  # global flags: --server, --project, --output, --debug
│   │   ├── login.go                 # login, logout, whoami
│   │   ├── project.go               # project list, project use <name>
│   │   └── deployment.go            # deployment list/get/deploy/rollback/logs
│   ├── api/
│   │   ├── client.go                # Invoke(ctx, method, body, out) error
│   │   ├── errors.go                # *api.Error, ErrUnauthenticated
│   │   ├── deployment.go            # typed wrappers
│   │   ├── project.go               # typed wrappers
│   │   └── types.go                 # Project, Deployment, LogLine
│   ├── auth/
│   │   ├── pkce.go                  # PKCE pair (S256)
│   │   ├── loopback.go              # local http server, browser launch
│   │   └── token.go                 # TokenSource: load/save/refresh, refresh-on-401
│   ├── config/config.go             # ~/.config/nortezh/{config,credentials}.json
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
| `nortezh login` | `/oauth/authorize` + `/oauth/token` (PKCE) | Loopback redirect. |
| `nortezh logout` | — | Wipes `credentials.json`. |
| `nortezh whoami` | `auth.me` | Prints user identity. |
| `nortezh project list` | `project.list` | Table or JSON. |
| `nortezh project use <name>` | `project.list` (resolve to ID) | Stores resolved `{id, name}` as `default_project` in config. |
| `nortezh deployment list [--project X]` | `deployment.list` | |
| `nortezh deployment get <name>` | `deployment.get` | |
| `nortezh deployment deploy <name> --image <ref>` | `deployment.deploy` | |
| `nortezh deployment rollback <name> --to <revision>` | `deployment.rollback` | |
| `nortezh deployment logs <name> [--revision N]` | `deployment.logRevision` | Streams to stdout. |

**Global flags:** `--server`, `--project`, `--output table|json` (default `table`), `--debug`.

## 5. Auth flow

OAuth2 authorization-code grant with **PKCE (S256)** and **loopback redirect**.

1. CLI generates `code_verifier` (random 64 bytes, base64url, no padding) and `code_challenge = base64url(SHA256(verifier))`.
2. CLI starts an HTTP server on `127.0.0.1:<random port>` with a single `/callback` endpoint.
3. Opens browser to:
   ```
   {server}/user/auth/oauth/authorize
     ?client_id=nortezh-cli
     &response_type=code
     &redirect_uri=http://127.0.0.1:<port>/callback
     &code_challenge=<challenge>
     &code_challenge_method=S256
     &state=<random32>
     &scope=cli
   ```
4. Backend redirects to `http://127.0.0.1:<port>/callback?code=...&state=...`.
5. CLI verifies `state`, POSTs to `{server}/user/auth/oauth/token`:
   ```
   grant_type=authorization_code
   code=<code>
   code_verifier=<verifier>
   redirect_uri=http://127.0.0.1:<port>/callback
   client_id=nortezh-cli
   ```
   Receives `{access_token, refresh_token, expires_in, token_type: "Bearer"}`.
6. Stores tokens in `~/.config/nortezh/credentials.json` (`0600`).
7. Browser sees "You can close this window"; CLI shuts the loopback server down.

**Refresh:** wrapped inside `api.Client`. On any 401, attempt one refresh (`grant_type=refresh_token`); retry the request once. On failure, return `ErrUnauthenticated` and instruct the user to re-run `nortezh login`.

**Logout:** wipe `credentials.json`. No server-side revocation in v0.1.

### 5.1 Backend dependency

The CLI cannot complete a real login until `nortezh-backend` ships:

1. `POST /user/auth/oauth/authorize` — issues code, requires PKCE, accepts loopback redirect URIs.
2. `POST /user/auth/oauth/token` — supports `authorization_code` and `refresh_token` grants.
3. Bearer-token middleware that accepts `Authorization: Bearer <access_token>` alongside session cookies on all protected routes.
4. Registered public client `nortezh-cli` (no secret; native app + PKCE).

Implementation against a `httptest.Server` fake unblocks the CLI work; real end-to-end usage waits on the backend.

## 6. API client

Single entry point in `internal/api`:

```go
type Client struct {
    BaseURL    string             // {server}/user
    HTTPClient *http.Client
    Tokens     *auth.TokenSource  // refresh-aware
    Debug      bool
}

func (c *Client) Invoke(ctx context.Context, method string, body, out any) error
```

- `method` is the arpc route name, e.g. `"deployment.list"`.
- Always `POST` to `BaseURL + "/" + method`, `Content-Type: application/json`.
- `body == nil` → sends `{}`. `out == nil` → result is discarded.
- `Authorization: Bearer <access_token>` injected by `TokenSource`.

**Envelope unwrap:**

```go
type envelope struct {
    OK     bool             `json:"ok"`
    Result json.RawMessage  `json:"result"`
    Error  *apiError        `json:"error"`
}
```

- `ok=true` → `json.Unmarshal(env.Result, out)`.
- `ok=false` → return `*api.Error{Code, Message, HTTPStatus}`.
- Non-2xx HTTP → return `*api.Error{Code: "http_error", HTTPStatus: ...}`.
- 401 → refresh-and-retry path (one attempt only).

**Typed wrappers** in `internal/api/deployment.go` and `project.go` — only the v0.1 calls. Each is a 3-line shim around `Invoke`.

**Types** (`internal/api/types.go`) — only fields v0.1 commands render:

```go
type Project    struct { ID, Name, SID string; CreatedAt time.Time }
type Deployment struct { ID, Name, ProjectID, Image, Status string; Revision int; UpdatedAt time.Time }
type LogLine    struct { Timestamp time.Time; Stream, Line string }
```

**Debug mode** (`--debug`): logs method, request body, response status, response body to stderr, truncated to 4 KB per direction. **Never** logs the `Authorization` header.

**User-facing errors:** CLI renders `*api.Error` as `Error: {Code}: {Message}` to stderr with exit 1. `ErrUnauthenticated` → `Error: not logged in. Run 'nortezh login'.`

## 7. Config & precedence

- `~/.config/nortezh/config.json` (`0644`):
  ```json
  {
    "server": "https://api.nortezh.com",
    "default_project": { "id": "prj_01H...", "name": "myproj" }
  }
  ```
- `~/.config/nortezh/credentials.json` (`0600`):
  ```json
  { "access_token": "...", "refresh_token": "...", "expires_at": "2026-05-18T07:30:00Z" }
  ```

Path resolution via `os.UserConfigDir()` (honors `XDG_CONFIG_HOME`).

**Precedence:** flag > env (`NORTEZH_SERVER`, `NORTEZH_CONFIG_DIR`) > config file > built-in default.

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
| `internal/api` | success, error envelope, 401-then-refresh-then-retry, 401-after-refresh-fails, non-2xx HTTP, debug redaction of `Authorization`. Uses `httptest.Server`. |
| `internal/auth` | PKCE pair length/charset, state mismatch rejection, callback parses `code`, loopback server picks a free port. Browser-open is injected: `OpenBrowser func(url string) error`. |
| `internal/config` | round-trip read/write; `credentials.json` is `0600`; env/flag/file precedence. |
| `internal/cli` | per-subcommand. Use a fake `api.Client` via an interface, not the struct. Instantiate `cobra.Command` per test — no globals. Cover happy path + one error path per command. |

No backend integration tests in v0.1. Future: `_integration_test.go` behind a `//go:build integration` tag once the backend ships OAuth endpoints.

`go test ./...` is the standard run.

## 10. Tooling

- Go 1.25.
- `spf13/cobra` for command tree.
- `Makefile` targets: `test`, `build`, `lint` (golangci-lint), `install`.
- No release pipeline in v0.1.

## 11. Risks & open questions

1. **Auth backend doesn't exist.** Tracked in §5.1. Until it lands, `nortezh login` is testable only against fakes.
2. **arpc envelope schema.** Spec assumes `{ok, result, error: {code, message}}`. Confirm against `backend/pkg/api` before wiring the client.
3. **`deployment.logRevision` payload shape.** Currently assumed to return an array; if it streams or paginates, `LogLine` and the logs command need revisiting.
4. **Public client registration.** `nortezh-cli` as a registered OAuth client (no secret) requires a backend admin step; document as a prerequisite.
