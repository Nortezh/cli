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

	"github.com/nortezh/cli/internal/auth"
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
