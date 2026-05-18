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
		`{"ok":true,"result":{"items":[{"name":"web","type":"WebService","actionStatus":"success"}]}}`)

	ds, err := c.ListDeployments(context.Background(), "proj-123")
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if len(ds) != 1 || ds[0].Name != "web" || ds[0].Type != "WebService" {
		t.Fatalf("got %+v", ds)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj-123" {
		t.Fatalf("expected project in body, got %v", sent)
	}
	pg, _ := sent["paginate"].(map[string]any)
	if pg["page"].(float64) != 1 || pg["perPage"].(float64) != 40 {
		t.Fatalf("paginate: %v", sent)
	}
}

func TestGetDeployment(t *testing.T) {
	c, seen := captureClient(t, "deployment.get",
		`{"ok":true,"result":{"name":"web","minReplica":1,"maxReplica":3,"resource":{"requests":{"memory":"128M"}}}}`)

	d, err := c.GetDeployment(context.Background(), "proj", "bkk-1", "web")
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if d.Name != "web" || d.MinReplica != 1 || d.MaxReplica != 3 || d.Memory() != "128M" {
		t.Fatalf("got %+v", d)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj" || sent["name"] != "web" || sent["location"] != "bkk-1" {
		t.Fatalf("body: %v", sent)
	}
}

func TestDeploy(t *testing.T) {
	c, seen := captureClient(t, "deployment.deploy",
		`{"ok":true,"result":null}`)

	if err := c.Deploy(context.Background(), "proj", "bkk-1", "web", "img:1"); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj" || sent["location"] != "bkk-1" || sent["name"] != "web" || sent["image"] != "img:1" {
		t.Fatalf("body: %v", sent)
	}
}

func TestRollback(t *testing.T) {
	c, seen := captureClient(t, "deployment.rollback",
		`{"ok":true,"result":{}}`)

	if err := c.Rollback(context.Background(), "proj", "bkk-1", "web", 2); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj" || sent["location"] != "bkk-1" || sent["name"] != "web" {
		t.Fatalf("body: %v", sent)
	}
	if sent["revision"].(float64) != 2 {
		t.Fatalf("revision: %v", sent)
	}
}

func TestListRevisions(t *testing.T) {
	c, seen := captureClient(t, "deployment.logRevision",
		`{"ok":true,"result":{"items":[{"revision":2,"image":"img:2","status":3,"deployedByEmail":"a@b","deployedAt":"2026-05-18T00:00:00Z"}]}}`)

	items, err := c.ListRevisions(context.Background(), "proj", "bkk-1", "web")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(items) != 1 || items[0].Revision != 2 || items[0].Image != "img:2" || items[0].Status != 3 {
		t.Fatalf("got %+v", items)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj" || sent["location"] != "bkk-1" || sent["name"] != "web" {
		t.Fatalf("body: %v", sent)
	}
}
