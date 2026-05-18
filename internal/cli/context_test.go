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
