package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/nortezh/cli/internal/api"
)

// toonPrinter renders TOON (Token-Oriented Object Notation): a compact,
// agent-friendly format that is ~40% cheaper than JSON while staying readable.
// Lists become `name[count]{fields}:` blocks; single objects become
// `name:` followed by `key: value` lines. See https://toonformat.dev.
type toonPrinter struct{ w io.Writer }

func (p *toonPrinter) PrintList(items any) error {
	headers, rows, err := tableRows(items)
	if err != nil {
		return err
	}
	name := collectionName(items)
	// Definitive empty state (AXI §5): say "nothing" explicitly.
	if len(rows) == 0 {
		_, err := fmt.Fprintf(p.w, "%s: 0 found\n", name)
		return err
	}
	fields := make([]string, len(headers))
	for i, h := range headers {
		fields[i] = toonKey(h)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s[%d]{%s}:\n", name, len(rows), strings.Join(fields, ","))
	for _, r := range rows {
		cells := make([]string, len(r))
		for i, c := range r {
			cells[i] = toonScalar(c)
		}
		fmt.Fprintf(&b, "  %s\n", strings.Join(cells, ","))
	}
	// Pre-computed aggregate (AXI §4): tell the agent the total up front.
	fmt.Fprintf(&b, "count: %d\n", len(rows))
	_, err = io.WriteString(p.w, b.String())
	return err
}

func (p *toonPrinter) Print(v any) error {
	name, rows, err := objectRows(v)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s:\n", name)
	for _, kv := range rows {
		fmt.Fprintf(&b, "  %s: %s\n", toonKey(kv[0]), toonScalar(kv[1]))
	}
	_, err = io.WriteString(p.w, b.String())
	return err
}

// Hints emits AXI §9 contextual next-step suggestions. It is a no-op for the
// json format (which must stay valid) and when there is nothing to suggest.
func Hints(w io.Writer, format string, lines ...string) {
	if format == "json" || len(lines) == 0 {
		return
	}
	fmt.Fprintf(w, "help[%d]:\n", len(lines))
	for _, l := range lines {
		fmt.Fprintf(w, "  %s\n", l)
	}
}

func collectionName(items any) string {
	switch items.(type) {
	case []api.Project:
		return "projects"
	case []api.Deployment:
		return "deployments"
	case []api.RevisionItem:
		return "revisions"
	case []api.Route:
		return "routes"
	case []api.Domain:
		return "domains"
	case []api.PullSecret:
		return "pullSecrets"
	default:
		return "items"
	}
}

func objectRows(v any) (string, [][]string, error) {
	switch t := v.(type) {
	case api.Project:
		return "project", zipKV(projectHeaders(), projectRow(t)), nil
	case api.Deployment:
		return "deployment", zipKV(deploymentHeaders(), deploymentRow(t)), nil
	case api.DeploymentDetail:
		return "deployment", deploymentDetailRows(t), nil
	case api.Route:
		return "route", zipKV(routeHeaders(), routeRow(t)), nil
	case api.RouteDetail:
		return "route", routeDetailRows(t), nil
	case api.Domain:
		return "domain", zipKV(domainHeaders(), domainRow(t)), nil
	case api.DomainDetail:
		return "domain", domainDetailRows(t), nil
	case api.PullSecret:
		return "pullSecret", zipKV(pullSecretHeaders(), pullSecretRow(t)), nil
	case api.PullSecretDetail:
		return "pullSecret", pullSecretDetailRows(t), nil
	default:
		return "", nil, fmt.Errorf("toon printer: unsupported type %T", v)
	}
}

func zipKV(headers, row []string) [][]string {
	out := make([][]string, len(headers))
	for i := range headers {
		out[i] = []string{headers[i], row[i]}
	}
	return out
}

// toonKey normalizes a column/field header to a TOON key: lowercased, with the
// ENV:KEY style colon flattened to a dot so it can't be mistaken for a value.
func toonKey(k string) string {
	return strings.ToLower(strings.ReplaceAll(k, ":", "."))
}

// toonScalar quotes a value only when needed: empty, surrounding whitespace, or
// containing a comma, quote, or newline (the delimiters TOON cares about).
func toonScalar(s string) string {
	if s == "" {
		return `""`
	}
	if s != strings.TrimSpace(s) || strings.ContainsAny(s, ",\"\n") {
		r := strings.ReplaceAll(s, `"`, `\"`)
		r = strings.ReplaceAll(r, "\n", `\n`)
		return `"` + r + `"`
	}
	return s
}
