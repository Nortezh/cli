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

	port := 8080
	if err := c.Deploy(context.Background(), "proj", "bkk-1", "web", "img:1", DeployOptions{
		AddEnv:    map[string]string{"FOO": "bar"},
		RemoveEnv: []string{"OLD"},
		EnvGroups: []string{"shared", "prod"},
		Port:      &port,
	}); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "proj" || sent["location"] != "bkk-1" || sent["name"] != "web" || sent["image"] != "img:1" {
		t.Fatalf("body: %v", sent)
	}
	add, _ := sent["addEnv"].(map[string]any)
	if add["FOO"] != "bar" {
		t.Fatalf("addEnv: %v", sent)
	}
	rm, _ := sent["removeEnv"].([]any)
	if len(rm) != 1 || rm[0] != "OLD" {
		t.Fatalf("removeEnv: %v", sent)
	}
	eg, _ := sent["envGroups"].([]any)
	if len(eg) != 2 || eg[0] != "shared" || eg[1] != "prod" {
		t.Fatalf("envGroups: %v", sent)
	}
	if sent["port"].(float64) != 8080 {
		t.Fatalf("port: %v", sent)
	}
}

// TestDeployPreservesEnvGroups verifies that a nil EnvGroups marshals to JSON
// null so the backend leaves the linked groups unchanged.
func TestDeployPreservesEnvGroups(t *testing.T) {
	c, seen := captureClient(t, "deployment.deploy",
		`{"ok":true,"result":null}`)

	if err := c.Deploy(context.Background(), "proj", "bkk-1", "web", "img:1", DeployOptions{}); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if v, ok := sent["envGroups"]; !ok || v != nil {
		t.Fatalf("envGroups should be null when unset, got %#v", sent["envGroups"])
	}
}

func TestCreateDeployment(t *testing.T) {
	c, seen := captureClient(t, "deployment.create",
		`{"ok":true,"result":{"id":"d-123"}}`)

	port := 8080
	min := 1
	max := 3
	id, err := c.CreateDeployment(context.Background(), "acme", "bkk-1", "api", "img:1", CreateDeploymentOptions{
		Type:       "WebService",
		Port:       &port,
		MinReplica: &min,
		MaxReplica: &max,
		Env:        map[string]string{"FOO": "bar"},
		EnvGroups:  []string{"shared"},
	})
	if err != nil {
		t.Fatalf("CreateDeployment: %v", err)
	}
	if id != "d-123" {
		t.Fatalf("id: %s", id)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["location"] != "bkk-1" || sent["name"] != "api" || sent["image"] != "img:1" {
		t.Fatalf("body: %v", sent)
	}
	if sent["type"] != "WebService" {
		t.Fatalf("type: %v", sent)
	}
	if sent["port"].(float64) != 8080 || sent["minReplica"].(float64) != 1 || sent["maxReplica"].(float64) != 3 {
		t.Fatalf("nums: %v", sent)
	}
	env, _ := sent["env"].(map[string]any)
	if env["FOO"] != "bar" {
		t.Fatalf("env: %v", sent)
	}
	eg, _ := sent["envGroups"].([]any)
	if len(eg) != 1 || eg[0] != "shared" {
		t.Fatalf("envGroups: %v", sent)
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

func TestListRoutes(t *testing.T) {
	c, seen := captureClient(t, "route.list",
		`{"ok":true,"result":{"items":[{"id":"r1","domain":"api.acme.com","path":"/","status":1,"deployment":{"name":"api-prod"},"location":{"slug":"bkk-1"}}]}}`)

	rs, err := c.ListRoutes(context.Background(), "acme", "api")
	if err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if len(rs) != 1 || rs[0].Domain != "api.acme.com" || rs[0].Deployment.Name != "api-prod" {
		t.Fatalf("got %+v", rs)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["search"] != "api" {
		t.Fatalf("body: %v", sent)
	}
}

func TestCreateRoute(t *testing.T) {
	c, seen := captureClient(t, "route.create",
		`{"ok":true,"result":{"id":"r123"}}`)

	rw := "/$1"
	id, err := c.CreateRoute(context.Background(), CreateRouteInput{
		Project:     "acme",
		Location:    "bkk-1",
		Domain:      "api.acme.com",
		Path:        "/",
		Target:      "deployment://api-prod",
		RewritePath: &rw,
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if id != "r123" {
		t.Fatalf("id: %s", id)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["domain"] != "api.acme.com" || sent["target"] != "deployment://api-prod" {
		t.Fatalf("body: %v", sent)
	}
	cfg, _ := sent["config"].(map[string]any)
	if cfg["rewritePath"] != "/$1" {
		t.Fatalf("config: %v", sent)
	}
}

func TestDeleteRoute(t *testing.T) {
	c, seen := captureClient(t, "route.delete", `{"ok":true,"result":null}`)

	if err := c.DeleteRoute(context.Background(), "acme", "api.acme.com", "/"); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["domain"] != "api.acme.com" || sent["path"] != "/" {
		t.Fatalf("body: %v", sent)
	}
}

func TestListDomains(t *testing.T) {
	c, seen := captureClient(t, "domain.list",
		`{"ok":true,"result":{"items":[{"id":1,"location":"bkk-1","domain":"acme.com","wildcard":false,"cdn":false,"status":"active","action":"none","createdAt":"2026-05-18T00:00:00Z"}]}}`)

	ds, err := c.ListDomains(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(ds) != 1 || ds[0].Domain != "acme.com" {
		t.Fatalf("got %+v", ds)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" {
		t.Fatalf("body: %v", sent)
	}
}

func TestCreateDomain(t *testing.T) {
	c, seen := captureClient(t, "domain.create", `{"ok":true,"result":null}`)

	if err := c.CreateDomain(context.Background(), "acme", "bkk-1", "api.acme.com", true, false); err != nil {
		t.Fatalf("CreateDomain: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["location"] != "bkk-1" || sent["domain"] != "api.acme.com" {
		t.Fatalf("body: %v", sent)
	}
	if sent["wildcard"] != true || sent["cdn"] != false {
		t.Fatalf("flags: %v", sent)
	}
}

func TestDeleteDomain(t *testing.T) {
	c, seen := captureClient(t, "domain.delete", `{"ok":true,"result":null}`)

	if err := c.DeleteDomain(context.Background(), "acme", "api.acme.com"); err != nil {
		t.Fatalf("DeleteDomain: %v", err)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(*seen), &sent)
	if sent["project"] != "acme" || sent["domain"] != "api.acme.com" {
		t.Fatalf("body: %v", sent)
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
