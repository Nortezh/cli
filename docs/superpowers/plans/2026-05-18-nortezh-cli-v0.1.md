# nortezh-cli v0.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `ntzh` v0.1 — a Go CLI binary that authenticates against `nortezh-backend` via the existing Google-OAuth relay (or service-account basic auth), lists projects, and performs the day-to-day deploy flow (list/get/deploy/rollback/logs).

**Architecture:** Cobra-based command tree → `internal/cli/*` thin handlers → `internal/api.Client.Invoke()` (single arpc POST + envelope unwrap) → `internal/auth.Creds` (injects Bearer or Basic). Config and credentials live as separate JSON files under `~/.config/ntzh/`. No persistent state besides credentials; every project-scoped command requires `--project`.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra`, stdlib only otherwise (`net/http`, `text/tabwriter`, `encoding/json`, `crypto/rand`, `crypto/sha256`). Tests use `testing` + `net/http/httptest`. Lint: `golangci-lint`.

**Reference docs:**
- Spec: `docs/superpowers/specs/2026-05-18-nortezh-cli-design.md`
- Backend auth endpoint: `nortezh-backend/api/auth/auth.go` (Google relay, mints 7-day token)
- Backend middleware: `nortezh-backend/api/handler.go:246` (`authMiddleware` accepts Bearer + Basic)
- Backend arpc error: `arpc.NewErrorCode("UNAUTHORIZED", "no authorization")`

---

## File Structure

```
nortezh-cli/
├── cmd/ntzh/main.go                       # entrypoint
├── internal/
│   ├── cli/
│   │   ├── root.go                        # NewRootCmd(), global flags
│   │   ├── context.go                     # build api.Client from flags+config+creds
│   │   ├── login.go                       # login, logout, whoami
│   │   ├── project.go                     # project list
│   │   ├── deployment.go                  # deployment {list,get,deploy,rollback,logs}
│   │   └── errors.go                      # Run() wrapper: format *api.Error, exit codes
│   ├── api/
│   │   ├── client.go                      # Client, Invoke()
│   │   ├── errors.go                      # *api.Error, ErrUnauthenticated
│   │   ├── types.go                       # Project, Deployment, LogLine
│   │   ├── project.go                     # ListProjects
│   │   └── deployment.go                  # ListDeployments, GetDeployment, Deploy, Rollback, LogRevision
│   ├── auth/
│   │   ├── creds.go                       # Creds interface + Bearer/ServiceAccount + JSON marshal
│   │   ├── store.go                       # Load/Save/Wipe credentials.json (0600)
│   │   └── loopback.go                    # Login(ctx, server) -> Creds via loopback callback
│   ├── config/
│   │   └── config.go                      # Config, Load/Save, Dir() (XDG)
│   └── output/
│       ├── printer.go                     # Printer iface, NewPrinter(format, w)
│       └── tables.go                      # Headers/Row for Project + Deployment
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── .gitignore
```

Each file has one responsibility. Tests live next to their unit as `*_test.go` in the same package (internal tests are fine — we want package access for fakes).

---

## Conventions used in every task

- Test-first. Write the failing test, run it, see it fail with the expected message, then implement.
- One commit per task at the end of the task. Commit message format: `<area>: <imperative>` (e.g. `api: add Invoke envelope unwrap`).
- All Go files start with `package <name>` and no copyright header.
- Run `go test ./...` before every commit — never partial paths.

---

## Task 1: Bootstrap module + cobra root + hello

**Files:**
- Create: `go.mod`
- Create: `cmd/ntzh/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`
- Create: `.gitignore`
- Create: `Makefile`

- [ ] **Step 1: Init module**

Run:
```bash
go mod init nortezh-cli
go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Write the failing test**

Create `internal/cli/root_test.go`:
```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Help(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "ntzh") {
		t.Fatalf("expected help to mention 'ntzh', got: %s", got)
	}
	if !strings.Contains(got, "--server") {
		t.Fatalf("expected --server flag in help, got: %s", got)
	}
	if !strings.Contains(got, "--project") {
		t.Fatalf("expected --project flag in help, got: %s", got)
	}
	if !strings.Contains(got, "--output") {
		t.Fatalf("expected --output flag in help, got: %s", got)
	}
	if !strings.Contains(got, "--debug") {
		t.Fatalf("expected --debug flag in help, got: %s", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL — `undefined: NewRootCmd`.

- [ ] **Step 4: Implement minimal root**

Create `internal/cli/root.go`:
```go
package cli

import "github.com/spf13/cobra"

// Globals holds parsed values of the global persistent flags.
type Globals struct {
	Server  string
	Project string
	Output  string
	Debug   bool
}

func NewRootCmd() *cobra.Command {
	g := &Globals{}
	cmd := &cobra.Command{
		Use:           "ntzh",
		Short:         "Command-line client for the Nortezh platform",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&g.Server, "server", "", "server URL (overrides config and NTZH_SERVER)")
	cmd.PersistentFlags().StringVar(&g.Project, "project", "", "project name (or NTZH_PROJECT)")
	cmd.PersistentFlags().StringVar(&g.Output, "output", "table", "output format: table|json")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "log HTTP traffic to stderr (token redacted)")
	return cmd
}
```

- [ ] **Step 5: Implement main**

Create `cmd/ntzh/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"nortezh-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Add tooling**

Create `.gitignore`:
```
/ntzh
/dist/
*.test
```

Create `Makefile`:
```makefile
.PHONY: test build lint install

test:
	go test ./...

build:
	go build -o ntzh ./cmd/ntzh

lint:
	golangci-lint run

install:
	go install ./cmd/ntzh
```

- [ ] **Step 7: Run tests and build**

Run: `go mod tidy && go test ./... && go build ./...`
Expected: PASS, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum cmd internal Makefile .gitignore
git commit -m "cli: bootstrap cobra root with global flags"
```

---

## Task 2: Config package (XDG paths, env, precedence)

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir_HonorsNTZHConfigDir(t *testing.T) {
	t.Setenv("NTZH_CONFIG_DIR", "/tmp/ntzh-test-abc")
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got != "/tmp/ntzh-test-abc" {
		t.Fatalf("Dir: got %q, want %q", got, "/tmp/ntzh-test-abc")
	}
}

func TestDir_FallsBackToUserConfigDir(t *testing.T) {
	t.Setenv("NTZH_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test-xyz")
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	want := filepath.Join("/tmp/xdg-test-xyz", "ntzh")
	if got != want {
		t.Fatalf("Dir: got %q, want %q", got, want)
	}
}

func TestLoadSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	c := &Config{Server: "https://example.com"}
	if err := Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server != "https://example.com" {
		t.Fatalf("Load server: got %q, want %q", got.Server, "https://example.com")
	}

	// File should exist at config.json
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("config.json missing: %v", err)
	}
}

func TestLoad_MissingReturnsZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load on empty dir should not error: %v", err)
	}
	if got.Server != "" {
		t.Fatalf("expected zero Server, got %q", got.Server)
	}
}

func TestResolveServer_Precedence(t *testing.T) {
	t.Setenv("NTZH_SERVER", "https://from-env")
	got := ResolveServer("https://from-flag", &Config{Server: "https://from-file"})
	if got != "https://from-flag" {
		t.Fatalf("flag wins: got %q", got)
	}

	got = ResolveServer("", &Config{Server: "https://from-file"})
	if got != "https://from-env" {
		t.Fatalf("env beats file: got %q", got)
	}

	t.Setenv("NTZH_SERVER", "")
	got = ResolveServer("", &Config{Server: "https://from-file"})
	if got != "https://from-file" {
		t.Fatalf("file wins: got %q", got)
	}

	got = ResolveServer("", &Config{})
	if got != DefaultServer {
		t.Fatalf("default wins: got %q", got)
	}
}

func TestResolveProject_Precedence(t *testing.T) {
	t.Setenv("NTZH_PROJECT", "envproj")
	if got := ResolveProject("flagproj"); got != "flagproj" {
		t.Fatalf("flag wins: got %q", got)
	}
	if got := ResolveProject(""); got != "envproj" {
		t.Fatalf("env: got %q", got)
	}
	t.Setenv("NTZH_PROJECT", "")
	if got := ResolveProject(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement**

Create `internal/config/config.go`:
```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultServer = "https://api.nortezh.com"
	envConfigDir  = "NTZH_CONFIG_DIR"
	envServer     = "NTZH_SERVER"
	envProject    = "NTZH_PROJECT"
	fileName      = "config.json"
)

type Config struct {
	Server string `json:"server,omitempty"`
}

// Dir returns the directory where ntzh stores its files.
// Precedence: $NTZH_CONFIG_DIR > os.UserConfigDir()/ntzh.
func Dir() (string, error) {
	if d := os.Getenv(envConfigDir); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ntzh"), nil
}

func path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fileName), nil
}

func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(c *Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, fileName), b, 0o644)
}

func ResolveServer(flag string, c *Config) string {
	if flag != "" {
		return flag
	}
	if v := os.Getenv(envServer); v != "" {
		return v
	}
	if c != nil && c.Server != "" {
		return c.Server
	}
	return DefaultServer
}

func ResolveProject(flag string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv(envProject)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/config/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "config: load/save with XDG paths and flag/env/file precedence"
```

---

## Task 3: Credentials store + Creds interface

**Files:**
- Create: `internal/auth/creds.go`
- Create: `internal/auth/store.go`
- Create: `internal/auth/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/auth/store_test.go`:
```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_BearerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	in := &BearerCreds{Token: "abc123", ExpiresAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	b, ok := got.(*BearerCreds)
	if !ok {
		t.Fatalf("expected *BearerCreds, got %T", got)
	}
	if b.Token != "abc123" {
		t.Fatalf("token: got %q", b.Token)
	}
	if !b.ExpiresAt.Equal(in.ExpiresAt) {
		t.Fatalf("expires_at: got %v want %v", b.ExpiresAt, in.ExpiresAt)
	}
}

func TestStore_ServiceAccountRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	in := &ServiceAccountCreds{Email: "ci@example.com", Key: "k"}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sa, ok := got.(*ServiceAccountCreds)
	if !ok {
		t.Fatalf("expected *ServiceAccountCreds, got %T", got)
	}
	if sa.Email != "ci@example.com" || sa.Key != "k" {
		t.Fatalf("got %+v", sa)
	}
}

func TestStore_FileMode0600(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	if err := Save(&BearerCreds{Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Fatalf("credentials.json mode: got %o want 600", mode)
	}
}

func TestStore_LoadMissingReturnsErrNoCreds(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	_, err := Load()
	if err != ErrNoCreds {
		t.Fatalf("got %v, want ErrNoCreds", err)
	}
}

func TestStore_Wipe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	if err := Save(&BearerCreds{Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Wipe(); err != nil {
		t.Fatalf("Wipe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, got %v", err)
	}
	// Wipe on missing is a no-op.
	if err := Wipe(); err != nil {
		t.Fatalf("Wipe on missing: %v", err)
	}
}

func TestBearerCreds_Apply(t *testing.T) {
	c := &BearerCreds{Token: "abc"}
	r := httptest.NewRequest(http.MethodPost, "http://x/", nil)
	c.Apply(r)
	if got := r.Header.Get("Authorization"); got != "Bearer abc" {
		t.Fatalf("got %q", got)
	}
}

func TestServiceAccountCreds_Apply(t *testing.T) {
	c := &ServiceAccountCreds{Email: "u@x", Key: "p"}
	r := httptest.NewRequest(http.MethodPost, "http://x/", nil)
	c.Apply(r)
	got := r.Header.Get("Authorization")
	if !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("got %q", got)
	}
	u, p, ok := r.BasicAuth()
	if !ok || u != "u@x" || p != "p" {
		t.Fatalf("basic auth parse: u=%q p=%q ok=%v", u, p, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/...`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement creds**

Create `internal/auth/creds.go`:
```go
package auth

import (
	"net/http"
	"time"
)

// Creds applies authentication to an outgoing request.
type Creds interface {
	Apply(*http.Request)
}

type BearerCreds struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (c *BearerCreds) Apply(r *http.Request) {
	r.Header.Set("Authorization", "Bearer "+c.Token)
}

type ServiceAccountCreds struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

func (c *ServiceAccountCreds) Apply(r *http.Request) {
	r.SetBasicAuth(c.Email, c.Key)
}
```

- [ ] **Step 4: Implement store**

Create `internal/auth/store.go`:
```go
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"nortezh-cli/internal/config"
)

const fileName = "credentials.json"

var ErrNoCreds = errors.New("auth: no credentials (run 'ntzh login')")

type fileShape struct {
	Kind                string               `json:"kind"`
	*BearerCreds        `json:",omitempty"`
	*ServiceAccountCreds `json:",omitempty"`
}

func path() (string, error) {
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, fileName), nil
}

func Load() (Creds, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, ErrNoCreds
	}
	if err != nil {
		return nil, err
	}
	// Decode kind first.
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return nil, err
	}
	switch head.Kind {
	case "bearer":
		var c BearerCreds
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		return &c, nil
	case "service_account":
		var c ServiceAccountCreds
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, err
		}
		return &c, nil
	default:
		return nil, errors.New("auth: unknown credentials kind: " + head.Kind)
	}
}

func Save(c Creds) error {
	d, err := config.Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}

	var payload []byte
	switch v := c.(type) {
	case *BearerCreds:
		payload, err = json.MarshalIndent(struct {
			Kind string `json:"kind"`
			*BearerCreds
		}{"bearer", v}, "", "  ")
	case *ServiceAccountCreds:
		payload, err = json.MarshalIndent(struct {
			Kind string `json:"kind"`
			*ServiceAccountCreds
		}{"service_account", v}, "", "  ")
	default:
		return errors.New("auth: unsupported creds type")
	}
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, fileName), payload, 0o600)
}

func Wipe() error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/auth/... ./internal/config/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/auth
git commit -m "auth: add Creds interface, credentials store with 0600 mode"
```

---

## Task 4: API client — Invoke + envelope + errors

**Files:**
- Create: `internal/api/errors.go`
- Create: `internal/api/types.go`
- Create: `internal/api/client.go`
- Create: `internal/api/client_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/api/client_test.go`:
```go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeCreds struct{ token string }

func (f *fakeCreds) Apply(r *http.Request) { r.Header.Set("Authorization", "Bearer "+f.token) }

func newClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Creds:      &fakeCreds{token: "tkn"},
	}
}

func TestInvoke_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s", r.Method)
		}
		if r.URL.Path != "/deployment.list" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tkn" {
			t.Errorf("auth: got %q", r.Header.Get("Authorization"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type: got %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(bytes.TrimSpace(body), []byte(`{"project_id":"p1"}`)) {
			t.Errorf("body: got %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[{"name":"a"}]}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	var out struct {
		Items []struct{ Name string } `json:"items"`
	}
	if err := c.Invoke(context.Background(), "deployment.list",
		map[string]string{"project_id": "p1"}, &out); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Name != "a" {
		t.Fatalf("decode: %+v", out)
	}
}

func TestInvoke_NilBodySendsEmptyObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if got := string(bytes.TrimSpace(body)); got != "{}" {
			t.Errorf("nil body should send {}, got %s", got)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":null}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	if err := c.Invoke(context.Background(), "x.y", nil, nil); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
}

func TestInvoke_ErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"BAD_INPUT","message":"name required"}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "BAD_INPUT" || apiErr.Message != "name required" {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestInvoke_UnauthorizedMapsToErrUnauthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"UNAUTHORIZED","message":"no authorization"}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestInvoke_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newClient(t, srv)
	err := c.Invoke(context.Background(), "x.y", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "http_error" || apiErr.HTTPStatus != 500 {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestInvoke_DebugRedactsAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	c := newClient(t, srv)
	c.Debug = true
	c.DebugWriter = &buf
	_ = c.Invoke(context.Background(), "x.y", nil, nil)

	if strings.Contains(buf.String(), "tkn") {
		t.Fatalf("debug log leaked token: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "POST") {
		t.Fatalf("expected POST in debug log, got %s", buf.String())
	}
}

func TestInvoke_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hang briefly so context cancel wins
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.Invoke(ctx, "x.y", nil, nil)
	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
}

// Confirm marshalling rules: errors.go encodes Error.Error() including code.
func TestError_String(t *testing.T) {
	e := &Error{Code: "X", Message: "y"}
	if got := e.Error(); !strings.Contains(got, "X") || !strings.Contains(got, "y") {
		t.Fatalf("got %q", got)
	}
}

// Sanity: types in types.go compile and marshal as expected.
func TestTypes_MarshalRoundtrip(t *testing.T) {
	in := Deployment{Name: "d1", Revision: 3, Status: "running"}
	b, _ := json.Marshal(in)
	var out Deployment
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Name != "d1" || out.Revision != 3 {
		t.Fatalf("roundtrip: %+v", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/...`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement errors**

Create `internal/api/errors.go`:
```go
package api

import (
	"errors"
	"fmt"
)

// ErrUnauthenticated indicates the server rejected the credentials.
// CLI handlers map this to "Error: not logged in. Run 'ntzh login'.".
var ErrUnauthenticated = errors.New("not authenticated")

type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string {
	if e.HTTPStatus != 0 && e.Code == "http_error" {
		return fmt.Sprintf("http_error: status %d", e.HTTPStatus)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

- [ ] **Step 4: Implement types**

Create `internal/api/types.go`:
```go
package api

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SID       string    `json:"sid"`
	CreatedAt time.Time `json:"created_at"`
}

type Deployment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ProjectID string    `json:"project_id"`
	Image     string    `json:"image"`
	Status    string    `json:"status"`
	Revision  int       `json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Line      string    `json:"line"`
}
```

- [ ] **Step 5: Implement client**

Create `internal/api/client.go`:
```go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"nortezh-cli/internal/auth"
)

const debugBodyLimit = 4 * 1024

type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	Creds       auth.Creds
	Debug       bool
	DebugWriter io.Writer // defaults to os.Stderr
}

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Invoke(ctx context.Context, method string, body, out any) error {
	var reqBody []byte
	if body == nil {
		reqBody = []byte("{}")
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = b
	}

	url := c.BaseURL + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Creds != nil {
		c.Creds.Apply(req)
	}

	c.debugReq(req, reqBody)

	httpc := c.HTTPClient
	if httpc == nil {
		httpc = http.DefaultClient
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.debugResp(resp, respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Error{Code: "http_error", HTTPStatus: resp.StatusCode, Message: string(truncate(respBody, debugBodyLimit))}
	}

	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if !env.OK {
		if env.Error == nil {
			return &Error{Code: "unknown", Message: "ok=false with no error block"}
		}
		if env.Error.Code == "UNAUTHORIZED" {
			return ErrUnauthenticated
		}
		return &Error{Code: env.Error.Code, Message: env.Error.Message}
	}
	if out == nil || len(env.Result) == 0 || string(env.Result) == "null" {
		return nil
	}
	return json.Unmarshal(env.Result, out)
}

func (c *Client) debugWriter() io.Writer {
	if c.DebugWriter != nil {
		return c.DebugWriter
	}
	return os.Stderr
}

func (c *Client) debugReq(r *http.Request, body []byte) {
	if !c.Debug {
		return
	}
	fmt.Fprintf(c.debugWriter(), "[ntzh] -> %s %s\n", r.Method, r.URL)
	// Headers, with Authorization redacted.
	for k, v := range r.Header {
		val := v
		if k == "Authorization" {
			val = []string{"[REDACTED]"}
		}
		fmt.Fprintf(c.debugWriter(), "[ntzh]    %s: %s\n", k, val)
	}
	fmt.Fprintf(c.debugWriter(), "[ntzh]    body: %s\n", truncate(body, debugBodyLimit))
}

func (c *Client) debugResp(r *http.Response, body []byte) {
	if !c.Debug {
		return
	}
	fmt.Fprintf(c.debugWriter(), "[ntzh] <- %d %s\n", r.StatusCode, r.Status)
	fmt.Fprintf(c.debugWriter(), "[ntzh]    body: %s\n", truncate(body, debugBodyLimit))
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return append(append([]byte{}, b[:n]...), []byte("...[truncated]")...)
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/api
git commit -m "api: add Invoke, envelope unwrap, typed errors"
```

---

## Task 5: Typed API wrappers — project + deployment

**Files:**
- Create: `internal/api/project.go`
- Create: `internal/api/deployment.go`
- Create: `internal/api/wrappers_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/api/wrappers_test.go`:
```go
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func captureClient(t *testing.T, method, respJSON string) (*Client, *string) {
	t.Helper()
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+method {
			t.Errorf("path: got %s want /%s", r.URL.Path, method)
		}
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		_, _ = w.Write([]byte(respJSON))
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), Creds: &fakeCreds{token: "x"}}, &seen
}

func TestListProjects(t *testing.T) {
	c, _ := captureClient(t, "project.list",
		`{"ok":true,"result":{"items":[{"id":"p1","name":"alpha"},{"id":"p2","name":"beta"}]}}`)

	ps, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(ps) != 2 || ps[0].Name != "alpha" || ps[1].ID != "p2" {
		t.Fatalf("got %+v", ps)
	}
}

func TestListDeployments(t *testing.T) {
	c, seen := captureClient(t, "deployment.list",
		`{"ok":true,"result":{"items":[{"name":"web","revision":2,"status":"running"}]}}`)

	ds, err := c.ListDeployments(context.Background(), "proj-123")
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if len(ds) != 1 || ds[0].Name != "web" || ds[0].Revision != 2 {
		t.Fatalf("got %+v", ds)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project_id"] != "proj-123" {
		t.Fatalf("expected project_id in body, got %v", sent)
	}
}

func TestGetDeployment(t *testing.T) {
	c, seen := captureClient(t, "deployment.get",
		`{"ok":true,"result":{"name":"web","status":"running"}}`)

	d, err := c.GetDeployment(context.Background(), "proj", "web")
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if d.Name != "web" {
		t.Fatalf("got %+v", d)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project_id"] != "proj" || sent["name"] != "web" {
		t.Fatalf("body: %v", sent)
	}
}

func TestDeploy(t *testing.T) {
	c, seen := captureClient(t, "deployment.deploy",
		`{"ok":true,"result":{"name":"web","status":"deploying","revision":3,"image":"img:1"}}`)

	d, err := c.Deploy(context.Background(), "proj", "web", "img:1")
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if d.Revision != 3 || d.Image != "img:1" {
		t.Fatalf("got %+v", d)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project_id"] != "proj" || sent["name"] != "web" || sent["image"] != "img:1" {
		t.Fatalf("body: %v", sent)
	}
}

func TestRollback(t *testing.T) {
	c, seen := captureClient(t, "deployment.rollback",
		`{"ok":true,"result":{}}`)

	if err := c.Rollback(context.Background(), "proj", "web", 2); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project_id"] != "proj" || sent["name"] != "web" {
		t.Fatalf("body: %v", sent)
	}
	if sent["revision"].(float64) != 2 {
		t.Fatalf("revision: %v", sent)
	}
}

func TestLogRevision(t *testing.T) {
	c, seen := captureClient(t, "deployment.logRevision",
		`{"ok":true,"result":{"items":[{"timestamp":"2026-05-18T00:00:00Z","line":"hello"}]}}`)

	lines, err := c.LogRevision(context.Background(), "proj", "web", 2)
	if err != nil {
		t.Fatalf("LogRevision: %v", err)
	}
	if len(lines) != 1 || lines[0].Line != "hello" {
		t.Fatalf("got %+v", lines)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project_id"] != "proj" || sent["name"] != "web" || sent["revision"].(float64) != 2 {
		t.Fatalf("body: %v", sent)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/...`
Expected: FAIL — undefined methods.

- [ ] **Step 3: Implement project wrappers**

Create `internal/api/project.go`:
```go
package api

import "context"

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out struct {
		Items []Project `json:"items"`
	}
	if err := c.Invoke(ctx, "project.list", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
```

- [ ] **Step 4: Implement deployment wrappers**

Create `internal/api/deployment.go`:
```go
package api

import "context"

type deploymentScope struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name,omitempty"`
}

func (c *Client) ListDeployments(ctx context.Context, projectID string) ([]Deployment, error) {
	var out struct {
		Items []Deployment `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.list",
		map[string]string{"project_id": projectID}, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetDeployment(ctx context.Context, projectID, name string) (*Deployment, error) {
	var out Deployment
	if err := c.Invoke(ctx, "deployment.get",
		deploymentScope{ProjectID: projectID, Name: name}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Deploy(ctx context.Context, projectID, name, image string) (*Deployment, error) {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Image     string `json:"image"`
	}{projectID, name, image}
	var out Deployment
	if err := c.Invoke(ctx, "deployment.deploy", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Rollback(ctx context.Context, projectID, name string, revision int) error {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Revision  int    `json:"revision"`
	}{projectID, name, revision}
	return c.Invoke(ctx, "deployment.rollback", body, nil)
}

func (c *Client) LogRevision(ctx context.Context, projectID, name string, revision int) ([]LogLine, error) {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Revision  int    `json:"revision"`
	}{projectID, name, revision}
	var out struct {
		Items []LogLine `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.logRevision", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/api/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/project.go internal/api/deployment.go internal/api/wrappers_test.go
git commit -m "api: add typed wrappers for project and deployment"
```

---

## Task 6: Output printer (table + JSON)

**Files:**
- Create: `internal/output/printer.go`
- Create: `internal/output/tables.go`
- Create: `internal/output/printer_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/output/printer_test.go`:
```go
package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"nortezh-cli/internal/api"
)

func TestPrinter_JSON_List(t *testing.T) {
	var buf bytes.Buffer
	p, err := NewPrinter("json", &buf)
	if err != nil {
		t.Fatalf("NewPrinter: %v", err)
	}
	items := []api.Project{{ID: "p1", Name: "alpha"}}
	if err := p.PrintList(items); err != nil {
		t.Fatalf("PrintList: %v", err)
	}
	var got []api.Project
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %s", buf.String())
	}
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("got %+v", got)
	}
}

func TestPrinter_Table_Project(t *testing.T) {
	var buf bytes.Buffer
	p, err := NewPrinter("table", &buf)
	if err != nil {
		t.Fatalf("NewPrinter: %v", err)
	}
	items := []api.Project{{ID: "p1", Name: "alpha"}}
	if err := p.PrintList(items); err != nil {
		t.Fatalf("PrintList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Fatalf("missing header, got: %s", out)
	}
	if !strings.Contains(out, "alpha") {
		t.Fatalf("missing row, got: %s", out)
	}
}

func TestPrinter_Table_Deployment(t *testing.T) {
	var buf bytes.Buffer
	p, _ := NewPrinter("table", &buf)
	items := []api.Deployment{{Name: "web", Revision: 3, Status: "running", Image: "img:1"}}
	if err := p.PrintList(items); err != nil {
		t.Fatalf("PrintList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAME", "REVISION", "STATUS", "IMAGE", "web", "running", "img:1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %s", want, out)
		}
	}
}

func TestNewPrinter_UnknownFormat(t *testing.T) {
	if _, err := NewPrinter("xml", &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestPrinter_JSON_Single(t *testing.T) {
	var buf bytes.Buffer
	p, _ := NewPrinter("json", &buf)
	if err := p.Print(api.Project{ID: "p1", Name: "alpha"}); err != nil {
		t.Fatalf("Print: %v", err)
	}
	var got api.Project
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %s", buf.String())
	}
	if got.Name != "alpha" {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/output/...`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement printer**

Create `internal/output/printer.go`:
```go
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"nortezh-cli/internal/api"
)

type Printer interface {
	Print(v any) error
	PrintList(items any) error
}

func NewPrinter(format string, w io.Writer) (Printer, error) {
	switch format {
	case "table":
		return &tablePrinter{w: w}, nil
	case "json":
		return &jsonPrinter{w: w}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s (want table|json)", format)
	}
}

type jsonPrinter struct{ w io.Writer }

func (p *jsonPrinter) Print(v any) error      { return writeJSON(p.w, v) }
func (p *jsonPrinter) PrintList(items any) error { return writeJSON(p.w, items) }

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

type tablePrinter struct{ w io.Writer }

func (p *tablePrinter) Print(v any) error {
	return p.PrintList([]any{v})
}

func (p *tablePrinter) PrintList(items any) error {
	headers, rows, err := tableRows(items)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, joinTab(headers))
	for _, r := range rows {
		fmt.Fprintln(tw, joinTab(r))
	}
	return tw.Flush()
}

func joinTab(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\t"
		}
		out += p
	}
	return out
}

// tableRows extracts headers/rows for a known typed slice.
func tableRows(items any) ([]string, [][]string, error) {
	switch v := items.(type) {
	case []api.Project:
		rows := make([][]string, 0, len(v))
		for _, it := range v {
			rows = append(rows, projectRow(it))
		}
		return projectHeaders(), rows, nil
	case []api.Deployment:
		rows := make([][]string, 0, len(v))
		for _, it := range v {
			rows = append(rows, deploymentRow(it))
		}
		return deploymentHeaders(), rows, nil
	default:
		return nil, nil, fmt.Errorf("table printer: unsupported type %T", items)
	}
}
```

- [ ] **Step 4: Implement table rows**

Create `internal/output/tables.go`:
```go
package output

import (
	"strconv"

	"nortezh-cli/internal/api"
)

func projectHeaders() []string { return []string{"NAME", "ID", "CREATED"} }

func projectRow(p api.Project) []string {
	return []string{p.Name, p.ID, p.CreatedAt.Format("2006-01-02")}
}

func deploymentHeaders() []string {
	return []string{"NAME", "REVISION", "STATUS", "IMAGE", "UPDATED"}
}

func deploymentRow(d api.Deployment) []string {
	return []string{
		d.Name,
		strconv.Itoa(d.Revision),
		d.Status,
		d.Image,
		d.UpdatedAt.Format("2006-01-02 15:04"),
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/output
git commit -m "output: add table and json printers"
```

---

## Task 7: Loopback login flow

**Files:**
- Create: `internal/auth/loopback.go`
- Create: `internal/auth/loopback_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/auth/loopback_test.go`:
```go
package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLogin_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	// Fake browser: when the CLI opens the URL, parse `callback` and `state`,
	// then GET callback?state=<state>&code=tok123.
	openBrowser := func(rawURL string) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		state := u.Query().Get("state")
		cb := u.Query().Get("callback")
		go func() {
			// brief delay so the loopback server is up
			time.Sleep(20 * time.Millisecond)
			resp, err := http.Get(cb + "?state=" + url.QueryEscape(state) + "&code=tok123")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	creds, err := Login(context.Background(), "https://server.example", openBrowser)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	b, ok := creds.(*BearerCreds)
	if !ok {
		t.Fatalf("expected *BearerCreds, got %T", creds)
	}
	if b.Token != "tok123" {
		t.Fatalf("token: got %q", b.Token)
	}
	if time.Until(b.ExpiresAt) < 6*24*time.Hour {
		t.Fatalf("expires_at should be ~7 days out, got %v", b.ExpiresAt)
	}
}

func TestLogin_RejectsBadState(t *testing.T) {
	openBrowser := func(rawURL string) error {
		u, _ := url.Parse(rawURL)
		cb := u.Query().Get("callback")
		go func() {
			time.Sleep(20 * time.Millisecond)
			resp, err := http.Get(cb + "?state=WRONG&code=tok")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Login(ctx, "https://server.example", openBrowser)
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}

func TestLogin_ContextCancel(t *testing.T) {
	openBrowser := func(string) error { return nil } // never hits callback
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := Login(ctx, "https://server.example", openBrowser)
	if err == nil {
		t.Fatal("expected error when context expires")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/...`
Expected: FAIL — `undefined: Login`.

- [ ] **Step 3: Implement loopback**

Create `internal/auth/loopback.go`:
```go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// OpenBrowserFunc is the function used to open the login URL.
// Tests inject a fake.
type OpenBrowserFunc func(url string) error

// Login starts a loopback server and walks the user through the existing
// /user/auth/?state&callback flow on the backend. Returns a *BearerCreds
// on success.
func Login(ctx context.Context, server string, open OpenBrowserFunc) (Creds, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	callback := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	type result struct {
		token string
		err   error
	}
	resultCh := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		gotState := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if gotState != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			resultCh <- result{err: errors.New("loopback: state mismatch")}
			return
		}
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			resultCh <- result{err: errors.New("loopback: empty code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h2>Logged in.</h2><p>You can close this window.</p></body></html>`)
		resultCh <- result{token: code}
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	loginURL := fmt.Sprintf("%s/user/auth/?state=%s&callback=%s",
		server, url.QueryEscape(state), url.QueryEscape(callback))

	if err := open(loginURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return &BearerCreds{
			Token:     res.token,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}, nil
	}
}

func randomState() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/auth/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/loopback.go internal/auth/loopback_test.go
git commit -m "auth: add loopback login against existing /user/auth flow"
```

---

## Task 8: CLI context — build api.Client from flags/config/creds

**Files:**
- Create: `internal/cli/context.go`
- Create: `internal/cli/errors.go`
- Modify: `internal/cli/root.go`
- Create: `internal/cli/context_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/context_test.go`:
```go
package cli

import (
	"errors"
	"testing"

	"nortezh-cli/internal/api"
)

func TestRequireProject(t *testing.T) {
	t.Setenv("NTZH_PROJECT", "")
	if _, err := requireProject(""); err == nil {
		t.Fatal("expected error when no project given")
	}
	if got, err := requireProject("flagproj"); err != nil || got != "flagproj" {
		t.Fatalf("flag: got %q err=%v", got, err)
	}
	t.Setenv("NTZH_PROJECT", "envproj")
	if got, err := requireProject(""); err != nil || got != "envproj" {
		t.Fatalf("env: got %q err=%v", got, err)
	}
}

func TestFormatCLIError(t *testing.T) {
	if got := formatCLIError(api.ErrUnauthenticated); got != "Error: not logged in. Run 'ntzh login'." {
		t.Fatalf("ErrUnauthenticated: got %q", got)
	}
	apiErr := &api.Error{Code: "BAD", Message: "x"}
	if got := formatCLIError(apiErr); got != "Error: BAD: x" {
		t.Fatalf("api err: got %q", got)
	}
	if got := formatCLIError(errors.New("boom")); got != "Error: boom" {
		t.Fatalf("plain: got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL — undefined helpers.

- [ ] **Step 3: Implement context helpers**

Create `internal/cli/context.go`:
```go
package cli

import (
	"errors"
	"net/http"
	"time"

	"nortezh-cli/internal/api"
	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

func buildClient(g *Globals) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	server := config.ResolveServer(g.Server, cfg)
	creds, err := auth.Load()
	if err != nil {
		// Allow construction without creds; commands that need them will fail
		// at Invoke() with ErrUnauthenticated.
		if !errors.Is(err, auth.ErrNoCreds) {
			return nil, err
		}
		creds = nil
	}
	return &api.Client{
		BaseURL:    server + "/user",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Creds:      creds,
		Debug:      g.Debug,
	}, nil
}

func requireProject(flag string) (string, error) {
	p := config.ResolveProject(flag)
	if p == "" {
		return "", errors.New("--project is required (or set NTZH_PROJECT)")
	}
	return p, nil
}

// resolveProjectID turns a project name into its server-side ID.
func resolveProjectID(c *api.Client, name string) (string, error) {
	ps, err := c.ListProjects(contextBackground())
	if err != nil {
		return "", err
	}
	for _, p := range ps {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", errors.New("project not found: " + name)
}
```

Note: `contextBackground()` is intentionally a separate helper so tests could inject ctx later; for now it's just `context.Background()`. Add it now to avoid a churn later.

Append to `internal/cli/context.go`:
```go
import_ctx_stub /* see below */
```

Replace the whole file with:
```go
package cli

import (
	"context"
	"errors"
	"net/http"
	"time"

	"nortezh-cli/internal/api"
	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

func buildClient(g *Globals) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	server := config.ResolveServer(g.Server, cfg)
	creds, err := auth.Load()
	if err != nil {
		if !errors.Is(err, auth.ErrNoCreds) {
			return nil, err
		}
		creds = nil
	}
	return &api.Client{
		BaseURL:    server + "/user",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Creds:      creds,
		Debug:      g.Debug,
	}, nil
}

func requireProject(flag string) (string, error) {
	p := config.ResolveProject(flag)
	if p == "" {
		return "", errors.New("--project is required (or set NTZH_PROJECT)")
	}
	return p, nil
}

func resolveProjectID(ctx context.Context, c *api.Client, name string) (string, error) {
	ps, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range ps {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", errors.New("project not found: " + name)
}
```

- [ ] **Step 4: Implement error formatter**

Create `internal/cli/errors.go`:
```go
package cli

import (
	"errors"
	"fmt"

	"nortezh-cli/internal/api"
)

func formatCLIError(err error) string {
	if errors.Is(err, api.ErrUnauthenticated) {
		return "Error: not logged in. Run 'ntzh login'."
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("Error: %s: %s", apiErr.Code, apiErr.Message)
	}
	return fmt.Sprintf("Error: %s", err.Error())
}
```

- [ ] **Step 5: Wire global Globals access**

Modify `internal/cli/root.go`. Replace the file with:
```go
package cli

import "github.com/spf13/cobra"

type Globals struct {
	Server  string
	Project string
	Output  string
	Debug   bool
}

func NewRootCmd() *cobra.Command {
	g := &Globals{}
	cmd := &cobra.Command{
		Use:           "ntzh",
		Short:         "Command-line client for the Nortezh platform",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&g.Server, "server", "", "server URL (overrides config and NTZH_SERVER)")
	cmd.PersistentFlags().StringVar(&g.Project, "project", "", "project name (or NTZH_PROJECT)")
	cmd.PersistentFlags().StringVar(&g.Output, "output", "table", "output format: table|json")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "log HTTP traffic to stderr (token redacted)")

	cmd.AddCommand(newLoginCmd(g))
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newWhoamiCmd(g))
	cmd.AddCommand(newProjectCmd(g))
	cmd.AddCommand(newDeploymentCmd(g))
	return cmd
}
```

- [ ] **Step 6: Update main to print errors using formatCLIError**

Modify `cmd/ntzh/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"nortezh-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cli.FormatCLIError(err))
		os.Exit(1)
	}
}
```

Export the formatter — append to `internal/cli/errors.go`:
```go
// FormatCLIError is the exported entry point used by cmd/ntzh.
func FormatCLIError(err error) string { return formatCLIError(err) }
```

(This task adds calls to `newLoginCmd`, `newLogoutCmd`, `newWhoamiCmd`, `newProjectCmd`, `newDeploymentCmd` that don't exist yet — Tasks 9 and 10 add them. **Add empty stubs now so the package builds:**)

Append to `internal/cli/root.go` (temporary stubs, replaced in later tasks):
```go
func newLoginCmd(*Globals) *cobra.Command      { return &cobra.Command{Use: "login"} }
func newLogoutCmd() *cobra.Command              { return &cobra.Command{Use: "logout"} }
func newWhoamiCmd(*Globals) *cobra.Command     { return &cobra.Command{Use: "whoami"} }
func newProjectCmd(*Globals) *cobra.Command    { return &cobra.Command{Use: "project"} }
func newDeploymentCmd(*Globals) *cobra.Command { return &cobra.Command{Use: "deployment"} }
```

- [ ] **Step 7: Run tests**

Run: `go test ./... && go build ./...`
Expected: PASS, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add internal/cli cmd/ntzh
git commit -m "cli: add client builder, project resolver, error formatter"
```

---

## Task 9: login, logout, whoami commands

**Files:**
- Modify: `internal/cli/root.go` (remove login/logout/whoami stubs)
- Create: `internal/cli/login.go`
- Create: `internal/cli/login_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/login_test.go`:
```go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"nortezh-cli/internal/auth"
)

func TestLoginCmd_Bearer_HappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	// Fake backend isn't actually contacted in login; only the loopback is.
	// We override OpenBrowser to act as the backend redirect.
	originalOpen := openBrowser
	t.Cleanup(func() { openBrowser = originalOpen })

	openBrowser = func(rawURL string) error {
		u, _ := url.Parse(rawURL)
		go func() {
			time.Sleep(20 * time.Millisecond)
			resp, err := http.Get(u.Query().Get("callback") +
				"?state=" + url.QueryEscape(u.Query().Get("state")) +
				"&code=tok123")
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"login", "--server", "https://server.example"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("execute: %v", err)
	}
	creds, err := auth.Load()
	if err != nil {
		t.Fatalf("Load creds: %v", err)
	}
	b, ok := creds.(*auth.BearerCreds)
	if !ok || b.Token != "tok123" {
		t.Fatalf("got %+v", creds)
	}
}

func TestLoginCmd_ServiceAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"login",
		"--service-account", "ci@example.com",
		"--key", "secret-key",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	creds, err := auth.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sa, ok := creds.(*auth.ServiceAccountCreds)
	if !ok {
		t.Fatalf("expected *ServiceAccountCreds, got %T", creds)
	}
	if sa.Email != "ci@example.com" || sa.Key != "secret-key" {
		t.Fatalf("got %+v", sa)
	}
}

func TestLogoutCmd_WipesCreds(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)
	if err := auth.Save(&auth.BearerCreds{Token: "x"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"logout"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, err := auth.Load(); err != auth.ErrNoCreds {
		t.Fatalf("expected ErrNoCreds, got %v", err)
	}
}

func TestWhoamiCmd_PrintsEmail(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)
	if err := auth.Save(&auth.BearerCreds{Token: "x"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/auth.me" {
			t.Errorf("path: %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = b
		_, _ = w.Write([]byte(`{"ok":true,"result":{"email":"me@example.com"}}`))
	}))
	defer srv.Close()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"whoami", "--server", srv.URL})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "me@example.com") {
		t.Fatalf("got %q", out.String())
	}
}

func TestWhoamiCmd_Unauthenticated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"UNAUTHORIZED","message":"no authorization"}}`))
	}))
	defer srv.Close()
	// Need creds present so the request is made.
	_ = auth.Save(&auth.BearerCreds{Token: "x"})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"whoami", "--server", srv.URL})
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	// The CLI returns the error; main.go formats it. We test the formatter here.
	if got := FormatCLIError(err); !strings.Contains(got, "not logged in") {
		t.Fatalf("formatter: %q", got)
	}

	// Use json output to discard
	_ = json.NewEncoder(io.Discard).Encode(nil)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL — login/logout/whoami stubs do nothing.

- [ ] **Step 3: Implement login.go**

Remove the three stub lines from `internal/cli/root.go`:
```go
func newLoginCmd(*Globals) *cobra.Command  { return &cobra.Command{Use: "login"} }
func newLogoutCmd() *cobra.Command          { return &cobra.Command{Use: "logout"} }
func newWhoamiCmd(*Globals) *cobra.Command { return &cobra.Command{Use: "whoami"} }
```

Create `internal/cli/login.go`:
```go
package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

// openBrowser is overridable in tests.
var openBrowser auth.OpenBrowserFunc = defaultOpenBrowser

func defaultOpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func newLoginCmd(g *Globals) *cobra.Command {
	var saEmail, saKey, keyFile string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate against the Nortezh backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			if saEmail != "" || saKey != "" || keyFile != "" {
				return runServiceAccountLogin(cmd, saEmail, saKey, keyFile)
			}
			return runBrowserLogin(cmd, g)
		},
	}
	cmd.Flags().StringVar(&saEmail, "service-account", "", "service account email (headless mode)")
	cmd.Flags().StringVar(&saKey, "key", "", "service account key value (use - to read from stdin)")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "path to file containing the service account key")
	return cmd
}

func runBrowserLogin(cmd *cobra.Command, g *Globals) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	server := config.ResolveServer(g.Server, cfg)
	fmt.Fprintln(cmd.OutOrStdout(), "Opening browser at", server, "...")
	creds, err := auth.Login(cmd.Context(), server, openBrowser)
	if err != nil {
		return err
	}
	if err := auth.Save(creds); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Logged in.")
	return nil
}

func runServiceAccountLogin(cmd *cobra.Command, email, key, keyFile string) error {
	if email == "" {
		return errors.New("--service-account is required")
	}
	if key == "" && keyFile == "" {
		return errors.New("--key or --key-file is required")
	}
	resolved := key
	if keyFile != "" {
		b, err := readKeyFile(keyFile)
		if err != nil {
			return err
		}
		resolved = b
	}
	if resolved == "" {
		return errors.New("empty key")
	}
	if err := auth.Save(&auth.ServiceAccountCreds{Email: email, Key: resolved}); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Saved service-account credentials for", email)
	return nil
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Forget local credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Wipe(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}

func newWhoamiCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			var out struct {
				Email string `json:"email"`
				ID    string `json:"id"`
			}
			if err := c.Invoke(cmd.Context(), "auth.me", nil, &out); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Email)
			return nil
		},
	}
}
```

- [ ] **Step 4: Implement keyfile reader**

Append to `internal/cli/login.go`:
```go
func readKeyFile(path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		return strings.TrimSpace(string(b)), err
	}
	b, err := os.ReadFile(path)
	return strings.TrimSpace(string(b)), err
}
```

And add imports `"io"`, `"os"`, `"strings"` at the top of the file.

- [ ] **Step 5: Run tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli
git commit -m "cli: implement login (browser + service-account), logout, whoami"
```

---

## Task 10: project + deployment commands

**Files:**
- Modify: `internal/cli/root.go` (remove last two stubs)
- Create: `internal/cli/project.go`
- Create: `internal/cli/deployment.go`
- Create: `internal/cli/commands_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/commands_test.go`:
```go
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nortezh-cli/internal/auth"
)

func setupAuthed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("NTZH_CONFIG_DIR", dir)
	if err := auth.Save(&auth.BearerCreds{Token: "tkn"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return dir
}

// project.list responds with two projects; first command call.
func newFakeBackend(t *testing.T, routes map[string]func(body []byte) string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range routes {
		h := handler
		mux.HandleFunc("/user/"+path, func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_, _ = w.Write([]byte(h(b)))
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestProjectList_Table(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p1","name":"alpha"},{"id":"p2","name":"beta"}]}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"project", "list", "--server", srv.URL})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "alpha") || !strings.Contains(out.String(), "beta") {
		t.Fatalf("table missing rows: %s", out.String())
	}
}

func TestDeploymentList_RequiresProject(t *testing.T) {
	_ = setupAuthed(t)
	t.Setenv("NTZH_PROJECT", "")
	srv := newFakeBackend(t, nil)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "list", "--server", srv.URL})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "project") {
		t.Fatalf("expected project required error, got %v", err)
	}
}

func TestDeploymentList_Happy(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"proj-1","name":"alpha"}]}}`
		},
		"deployment.list": func(body []byte) string {
			var p struct{ ProjectID string `json:"project_id"` }
			_ = json.Unmarshal(body, &p)
			if p.ProjectID != "proj-1" {
				return `{"ok":false,"error":{"code":"BAD","message":"expected proj-1"}}`
			}
			return `{"ok":true,"result":{"items":[{"name":"web","status":"running","revision":1,"image":"img:1"}]}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "list", "--server", srv.URL, "--project", "alpha"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "web") || !strings.Contains(out.String(), "running") {
		t.Fatalf("got %q", out.String())
	}
}

func TestDeploymentGet_JSON(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.get": func([]byte) string {
			return `{"ok":true,"result":{"name":"web","revision":7,"status":"running"}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "get", "web",
		"--server", srv.URL, "--project", "alpha", "--output", "json"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	var d map[string]any
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("not json: %s", out.String())
	}
	if d["name"] != "web" {
		t.Fatalf("got %v", d)
	}
}

func TestDeploymentDeploy(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.deploy": func(body []byte) string {
			var p struct {
				ProjectID string `json:"project_id"`
				Name      string `json:"name"`
				Image     string `json:"image"`
			}
			_ = json.Unmarshal(body, &p)
			if p.ProjectID != "p" || p.Name != "web" || p.Image != "img:2" {
				return `{"ok":false,"error":{"code":"X","message":"bad body"}}`
			}
			return `{"ok":true,"result":{"name":"web","revision":2,"image":"img:2","status":"deploying"}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "deploy", "web",
		"--server", srv.URL, "--project", "alpha", "--image", "img:2"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "deploying") {
		t.Fatalf("got %q", out.String())
	}
}

func TestDeploymentRollback(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.rollback": func(body []byte) string {
			return `{"ok":true,"result":{}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "rollback", "web",
		"--server", srv.URL, "--project", "alpha", "--to", "3"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "rolled back") {
		t.Fatalf("got %q", out.String())
	}
}

func TestDeploymentLogs(t *testing.T) {
	_ = setupAuthed(t)
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.logRevision": func([]byte) string {
			return `{"ok":true,"result":{"items":[
				{"timestamp":"2026-05-18T01:00:00Z","line":"first"},
				{"timestamp":"2026-05-18T01:00:01Z","line":"second"}
			]}}`
		},
	})

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"deployment", "logs", "web",
		"--server", srv.URL, "--project", "alpha", "--revision", "1"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "first") || !strings.Contains(out.String(), "second") {
		t.Fatalf("got %q", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL — stubs for project/deployment are empty.

- [ ] **Step 3: Remove the last stubs from `internal/cli/root.go`**

Delete these two lines:
```go
func newProjectCmd(*Globals) *cobra.Command    { return &cobra.Command{Use: "project"} }
func newDeploymentCmd(*Globals) *cobra.Command { return &cobra.Command{Use: "deployment"} }
```

- [ ] **Step 4: Implement project commands**

Create `internal/cli/project.go`:
```go
package cli

import (
	"github.com/spf13/cobra"

	"nortezh-cli/internal/output"
)

func newProjectCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(newProjectListCmd(g))
	return cmd
}

func newProjectListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			ps, err := c.ListProjects(cmd.Context())
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(ps)
		},
	}
}
```

- [ ] **Step 5: Implement deployment commands**

Create `internal/cli/deployment.go`:
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"nortezh-cli/internal/output"
)

func newDeploymentCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{Use: "deployment", Short: "Manage deployments"}
	cmd.AddCommand(newDeploymentListCmd(g))
	cmd.AddCommand(newDeploymentGetCmd(g))
	cmd.AddCommand(newDeploymentDeployCmd(g))
	cmd.AddCommand(newDeploymentRollbackCmd(g))
	cmd.AddCommand(newDeploymentLogsCmd(g))
	return cmd
}

func newDeploymentListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List deployments in the project",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			ds, err := c.ListDeployments(cmd.Context(), pid)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(ds)
		},
	}
}

func newDeploymentGetCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show a single deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			d, err := c.GetDeployment(cmd.Context(), pid, args[0])
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.Print(*d)
		},
	}
}

func newDeploymentDeployCmd(g *Globals) *cobra.Command {
	var image string
	cmd := &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy a new image to a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if image == "" {
				return fmt.Errorf("--image is required")
			}
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			d, err := c.Deploy(cmd.Context(), pid, args[0], image)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.Print(*d)
		},
	}
	cmd.Flags().StringVar(&image, "image", "", "container image reference")
	return cmd
}

func newDeploymentRollbackCmd(g *Globals) *cobra.Command {
	var to int
	cmd := &cobra.Command{
		Use:   "rollback <name>",
		Short: "Roll a deployment back to a previous revision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if to <= 0 {
				return fmt.Errorf("--to <revision> is required")
			}
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			if err := c.Rollback(cmd.Context(), pid, args[0], to); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s rolled back to revision %d\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().IntVar(&to, "to", 0, "revision to roll back to")
	return cmd
}

func newDeploymentLogsCmd(g *Globals) *cobra.Command {
	var revision int
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Print logs for a deployment revision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			lines, err := c.LogRevision(cmd.Context(), pid, args[0], revision)
			if err != nil {
				return err
			}
			for _, l := range lines {
				fmt.Fprintln(cmd.OutOrStdout(), l.Line)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&revision, "revision", 0, "revision number (0 = latest)")
	return cmd
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/cli
git commit -m "cli: implement project list and deployment subcommands"
```

---

## Task 11: README + final polish

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Create `README.md`:
```markdown
# ntzh — Nortezh CLI

CLI for the Nortezh deployment platform.

## Install

    go install ./cmd/ntzh

## Auth

    ntzh login                                     # browser-based Google login
    ntzh login --service-account ci@x --key-file k # headless (CI)
    ntzh logout
    ntzh whoami

## Commands

    ntzh project list
    ntzh deployment list   --project <name>
    ntzh deployment get    --project <name> <deployment>
    ntzh deployment deploy --project <name> <deployment> --image <ref>
    ntzh deployment rollback --project <name> <deployment> --to <revision>
    ntzh deployment logs   --project <name> <deployment> [--revision N]

Project is required on every project-scoped command (no stored default).
Use `--output json` for machine-readable output, `--debug` to log HTTP traffic
to stderr (Authorization header is redacted).

## Configuration

    ~/.config/ntzh/config.json       # { "server": "..." }
    ~/.config/ntzh/credentials.json  # 0600, bearer or service_account

Env: NTZH_SERVER, NTZH_PROJECT, NTZH_CONFIG_DIR. Flag > env > file > default.
```

- [ ] **Step 2: Build and smoke-test**

Run:
```bash
go build -o ntzh ./cmd/ntzh
./ntzh --help
./ntzh project --help
./ntzh deployment deploy --help
```
Expected: help text for each command, listing the documented flags.

- [ ] **Step 3: Final test sweep**

Run: `go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

---

## Self-review

**Spec coverage** (each §X mapped to tasks):
- §1 Purpose / §2 Non-goals — informational, no task.
- §3 Layout — Tasks 1–10 create every file in the tree.
- §4 Commands — login/logout/whoami in Task 9, `project list` in Task 10, all `deployment` subcommands in Task 10.
- §5 Auth flow — loopback in Task 7, login/logout commands wire it in Task 9.
- §5.2 Service-account login — Task 9 `--service-account/--key/--key-file`.
- §5.3 Logout — Task 9.
- §6 API client — Task 4 (client+envelope), Task 5 (typed wrappers).
- §7 Config & precedence — Task 2.
- §8 Output — Task 6.
- §9 Testing — every package has a `*_test.go` covering the required cases in its task.
- §10 Tooling — Task 1 (Makefile, .gitignore, Cobra dep).
- §11 Risks — informational, no task. Headless UX message surfaces when interactive login is run without browser (deferred — `defaultOpenBrowser` will simply return the OS error; acceptable).

**Placeholder scan:** all code blocks contain full code, no "TODO" / "TBD" left.

**Type consistency:**
- `auth.Creds` interface, `BearerCreds`, `ServiceAccountCreds` — consistent in Tasks 3, 4, 7, 9.
- `api.Client.Invoke`, `api.Error`, `api.ErrUnauthenticated` — consistent in Tasks 4, 5, 8, 9, 10.
- `config.Dir`, `config.Load`, `config.ResolveServer`, `config.ResolveProject` — consistent in Tasks 2, 8, 9, 10.
- `cli.Globals`, `cli.NewRootCmd`, `cli.FormatCLIError` — consistent in Tasks 1, 8, 9, 10.
- `output.Printer`, `output.NewPrinter` — consistent in Tasks 6, 10.

No drift detected.
