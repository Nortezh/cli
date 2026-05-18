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
