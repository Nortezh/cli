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
