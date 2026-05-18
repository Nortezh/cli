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
