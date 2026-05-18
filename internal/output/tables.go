package output

import (
	"fmt"
	"strconv"

	"github.com/nortezh/cli/internal/api"
)

func projectHeaders() []string { return []string{"NAME", "PROJECT_ID", "CREATED_AT"} }

func projectRow(p api.Project) []string {
	return []string{p.Name, p.Slug, p.CreatedAt.Format("2006-01-02 15:04")}
}

func deploymentHeaders() []string {
	return []string{"NAME", "TYPE", "STATUS", "LOCATION", "REPLICAS", "LAST_DEPLOYED"}
}

func deploymentRow(d api.Deployment) []string {
	last := ""
	if !d.LastDeployedAt.IsZero() {
		last = d.LastDeployedAt.Format("2006-01-02 15:04")
	}
	return []string{
		d.Name,
		d.Type,
		d.ActionStatus,
		d.Location,
		fmt.Sprintf("%d-%d", d.MinReplicas, d.MaxReplicas),
		last,
	}
}

func deploymentDetailHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

func deploymentDetailRows(d api.DeploymentDetail) [][]string {
	last := ""
	if !d.LatestDeployedAt.IsZero() {
		last = d.LatestDeployedAt.Format("2006-01-02 15:04")
	}
	return [][]string{
		{"NAME", d.Name},
		{"PROJECT", d.Project},
		{"LOCATION", d.Location},
		{"TYPE", d.Type},
		{"REVISION", strconv.Itoa(d.Revision)},
		{"IMAGE", d.Image},
		{"STATUS", d.ActionStatus},
		{"REPLICAS", fmt.Sprintf("%d-%d", d.MinReplica, d.MaxReplica)},
		{"MEMORY", d.Memory()},
		{"URL", d.URL},
		{"LAST_DEPLOYED", last},
		{"DEPLOYED_BY", d.DeployedByEmail},
	}
}

func revisionHeaders() []string {
	return []string{"REVISION", "IMAGE", "STATUS", "DEPLOYED_BY", "DEPLOYED_AT"}
}

func revisionRow(r api.RevisionItem) []string {
	return []string{
		strconv.Itoa(r.Revision),
		r.Image,
		revisionStatusLabel(r.Status),
		r.DeployedByEmail,
		r.DeployedAt.Format("2006-01-02 15:04"),
	}
}

func revisionStatusLabel(s int) string {
	switch s {
	case 1:
		return "pending"
	case 2:
		return "deploying"
	case 3:
		return "deployed"
	default:
		return strconv.Itoa(s)
	}
}
