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
