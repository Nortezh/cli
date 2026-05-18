package output

import (
	"fmt"
	"sort"
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
	rows := [][]string{
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
	if len(d.Env) > 0 {
		keys := make([]string, 0, len(d.Env))
		for k := range d.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rows = append(rows, []string{"ENV:" + k, d.Env[k]})
		}
	}
	return rows
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

func actionStatusLabel(s int) string {
	switch s {
	case 0:
		return "pending"
	case 1:
		return "success"
	case 2:
		return "failed"
	default:
		return strconv.Itoa(s)
	}
}

func routeHeaders() []string {
	return []string{"DOMAIN", "PATH", "TARGET", "LOCATION", "STATUS"}
}

func routeRow(r api.Route) []string {
	return []string{
		r.Domain,
		r.Path,
		r.Deployment.Name,
		r.Location.Slug,
		actionStatusLabel(r.Status),
	}
}

func routeDetailRows(r api.RouteDetail) [][]string {
	rewrite := ""
	if r.Config.RewritePath != nil {
		rewrite = *r.Config.RewritePath
	}
	basic := ""
	if r.Config.BasicAuth != nil {
		basic = r.Config.BasicAuth.Username + ":***"
	}
	return [][]string{
		{"ID", r.ID},
		{"DOMAIN", r.Domain},
		{"PATH", r.Path},
		{"DEPLOYMENT", r.Deployment.Name},
		{"LOCATION", r.Location.Slug},
		{"STATUS", actionStatusLabel(r.Status)},
		{"REWRITE_PATH", rewrite},
		{"BASIC_AUTH", basic},
		{"CREATED_AT", r.CreatedAt.Format("2006-01-02 15:04")},
		{"CREATED_BY", r.CreatedBy},
	}
}

func domainHeaders() []string {
	return []string{"DOMAIN", "LOCATION", "WILDCARD", "CDN", "STATUS", "CREATED_AT"}
}

func domainRow(d api.Domain) []string {
	return []string{
		d.Domain,
		d.Location,
		boolLabel(d.Wildcard),
		boolLabel(d.CDN),
		d.Status,
		d.CreatedAt.Format("2006-01-02 15:04"),
	}
}

func domainDetailRows(d api.DomainDetail) [][]string {
	rows := [][]string{
		{"DOMAIN", d.Domain},
		{"LOCATION", d.Location},
		{"WILDCARD", boolLabel(d.Wildcard)},
		{"CDN", boolLabel(d.CDN)},
		{"STATUS", d.Status},
		{"CREATED_AT", d.CreatedAt.Format("2006-01-02 15:04")},
	}
	if len(d.DNSConfig.CName) > 0 {
		rows = append(rows, []string{"DNS_CNAME", d.DNSConfig.CName[0]})
	}
	if d.Verification.Ownership.Name != "" {
		rows = append(rows,
			[]string{"VERIFY_TYPE", d.Verification.Ownership.Type},
			[]string{"VERIFY_NAME", d.Verification.Ownership.Name},
			[]string{"VERIFY_VALUE", d.Verification.Ownership.Value},
		)
	}
	return rows
}

func boolLabel(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
