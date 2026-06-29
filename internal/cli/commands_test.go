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

	"github.com/nortezh/cli/internal/auth"
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

// newFakeBackend serves arpc-style routes under /user/<route>.
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

	cmd := NewRootCmd("test", "none", "unknown")
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
	cmd := NewRootCmd("test", "none", "unknown")
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
			return `{"ok":true,"result":{"items":[{"id":"proj-1","no":"alpha-slug","name":"alpha"}]}}`
		},
		"deployment.list": func(body []byte) string {
			var p struct {
				Project string `json:"project"`
			}
			_ = json.Unmarshal(body, &p)
			if p.Project != "alpha-slug" {
				return `{"ok":false,"error":{"code":"BAD","message":"expected alpha-slug"}}`
			}
			return `{"ok":true,"result":{"items":[{"name":"web","type":"WebService","actionStatus":"running"}]}}`
		},
	})

	cmd := NewRootCmd("test", "none", "unknown")
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
	t.Setenv("NTZH_LOCATION", "bkk-1")
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.get": func(body []byte) string {
			var p struct {
				Project, Location, Name string
			}
			_ = json.Unmarshal(body, &p)
			if p.Location != "bkk-1" || p.Name != "web" {
				return `{"ok":false,"error":{"code":"X","message":"bad body"}}`
			}
			return `{"ok":true,"result":{"name":"web","revision":7,"minReplica":1,"maxReplica":3}}`
		},
	})

	cmd := NewRootCmd("test", "none", "unknown")
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
	t.Setenv("NTZH_LOCATION", "bkk-1")
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","no":"p-slug","name":"alpha"}]}}`
		},
		"deployment.deploy": func(body []byte) string {
			var p struct {
				Project, Location, Name, Image string
			}
			_ = json.Unmarshal(body, &p)
			if p.Project != "p-slug" || p.Location != "bkk-1" || p.Name != "web" || p.Image != "img:2" {
				return `{"ok":false,"error":{"code":"X","message":"bad body"}}`
			}
			return `{"ok":true,"result":null}`
		},
	})

	cmd := NewRootCmd("test", "none", "unknown")
	cmd.SetArgs([]string{"deployment", "deploy", "web",
		"--server", srv.URL, "--project", "alpha", "--image", "img:2"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "deploying img:2") {
		t.Fatalf("got %q", out.String())
	}
}

func TestDeploymentRollback(t *testing.T) {
	_ = setupAuthed(t)
	t.Setenv("NTZH_LOCATION", "bkk-1")
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.rollback": func(body []byte) string {
			var p struct {
				Location string
				Revision int
			}
			_ = json.Unmarshal(body, &p)
			if p.Location != "bkk-1" || p.Revision != 3 {
				return `{"ok":false,"error":{"code":"X","message":"bad body"}}`
			}
			return `{"ok":true,"result":{}}`
		},
	})

	cmd := NewRootCmd("test", "none", "unknown")
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

func TestDeploymentRevisions(t *testing.T) {
	_ = setupAuthed(t)
	t.Setenv("NTZH_LOCATION", "bkk-1")
	srv := newFakeBackend(t, map[string]func([]byte) string{
		"project.list": func([]byte) string {
			return `{"ok":true,"result":{"items":[{"id":"p","name":"alpha"}]}}`
		},
		"deployment.logRevision": func([]byte) string {
			return `{"ok":true,"result":{"items":[
				{"revision":2,"image":"img:2","status":3,"deployedByEmail":"a@b","deployedAt":"2026-05-18T01:00:00Z"},
				{"revision":1,"image":"img:1","status":3,"deployedByEmail":"a@b","deployedAt":"2026-05-18T00:00:00Z"}
			]}}`
		},
	})

	cmd := NewRootCmd("test", "none", "unknown")
	cmd.SetArgs([]string{"deployment", "revisions", "web",
		"--server", srv.URL, "--project", "alpha"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "img:2") || !strings.Contains(out.String(), "img:1") {
		t.Fatalf("got %q", out.String())
	}
}
