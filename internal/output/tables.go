package output

import (
	"strconv"

	"nortezh-cli/internal/api"
)

func projectHeaders() []string { return []string{"NAME", "ID", "CREATED"} }

func projectRow(p api.Project) []string {
	return []string{p.Name, p.ID, p.CreatedAt.Format("2006-01-02")}
}

func deploymentHeaders() []string {
	return []string{"NAME", "REVISION", "STATUS", "IMAGE", "UPDATED"}
}

func deploymentRow(d api.Deployment) []string {
	return []string{
		d.Name,
		strconv.Itoa(d.Revision),
		d.Status,
		d.Image,
		d.UpdatedAt.Format("2006-01-02 15:04"),
	}
}
