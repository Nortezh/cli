package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nortezh/cli/internal/api"
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
	items := []api.Deployment{{Name: "web", Type: "WebService", ActionStatus: "success", Location: "olufy-0", MinReplicas: 2, MaxReplicas: 5}}
	if err := p.PrintList(items); err != nil {
		t.Fatalf("PrintList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAME", "TYPE", "STATUS", "LOCATION", "web", "WebService", "success", "olufy-0", "2-5"} {
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
