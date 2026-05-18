package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"nortezh-cli/internal/api"
)

type Printer interface {
	Print(v any) error
	PrintList(items any) error
}

func NewPrinter(format string, w io.Writer) (Printer, error) {
	switch format {
	case "table":
		return &tablePrinter{w: w}, nil
	case "json":
		return &jsonPrinter{w: w}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s (want table|json)", format)
	}
}

type jsonPrinter struct{ w io.Writer }

func (p *jsonPrinter) Print(v any) error        { return writeJSON(p.w, v) }
func (p *jsonPrinter) PrintList(items any) error { return writeJSON(p.w, items) }

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

type tablePrinter struct{ w io.Writer }

func (p *tablePrinter) Print(v any) error {
	switch t := v.(type) {
	case api.Project:
		return p.PrintList([]api.Project{t})
	case api.Deployment:
		return p.PrintList([]api.Deployment{t})
	default:
		return fmt.Errorf("table printer: unsupported type %T", v)
	}
}

func (p *tablePrinter) PrintList(items any) error {
	headers, rows, err := tableRows(items)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(p.w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, joinTab(headers))
	for _, r := range rows {
		fmt.Fprintln(tw, joinTab(r))
	}
	return tw.Flush()
}

func joinTab(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\t"
		}
		out += p
	}
	return out
}

func tableRows(items any) ([]string, [][]string, error) {
	switch v := items.(type) {
	case []api.Project:
		rows := make([][]string, 0, len(v))
		for _, it := range v {
			rows = append(rows, projectRow(it))
		}
		return projectHeaders(), rows, nil
	case []api.Deployment:
		rows := make([][]string, 0, len(v))
		for _, it := range v {
			rows = append(rows, deploymentRow(it))
		}
		return deploymentHeaders(), rows, nil
	default:
		return nil, nil, fmt.Errorf("table printer: unsupported type %T", items)
	}
}
