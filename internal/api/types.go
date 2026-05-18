package api

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"no"`
	CreatedAt time.Time `json:"createdAt"`
}

// Deployment is the row shape returned by deployment.list.
// It is a strict subset of the backend ListItem.
type Deployment struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Action         string    `json:"action"`
	ActionStatus   string    `json:"actionStatus"`
	StatusURL      string    `json:"statusUrl"`
	Memory         string    `json:"memory"`
	MinReplicas    int       `json:"minReplicas"`
	MaxReplicas    int       `json:"maxReplicas"`
	Location       string    `json:"location"`
	LastDeployedAt time.Time `json:"lastDeployedAt"`
}

// DeploymentDetail mirrors the backend deployment.get response (GetResult).
// Field names use the backend's camelCase tags exactly.
type DeploymentDetail struct {
	ID               string    `json:"id"`
	Project          string    `json:"project"`
	Location         string    `json:"location"`
	Name             string    `json:"name"`
	Image            string    `json:"image"`
	MinReplica       int       `json:"minReplica"`
	MaxReplica       int       `json:"maxReplica"`
	Type             string    `json:"type"`
	Port             int       `json:"port"`
	Protocol         string    `json:"protocol"`
	Internal         bool      `json:"internal"`
	Revision         int       `json:"revision"`
	LatestDeployedAt time.Time `json:"latestDeployedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	Action           string    `json:"action"`
	ActionStatus     string    `json:"actionStatus"`
	Resources        *Resource `json:"resource"`
	URL              string    `json:"url"`
	InternalURL      string    `json:"internalUrl"`
	LogURL           string    `json:"logUrl"`
	StatusURL        string    `json:"statusUrl"`
	DeployedByEmail  string    `json:"deployedByEmail"`
}

// Memory returns the requested memory string ("Shared" when unset).
func (d *DeploymentDetail) Memory() string {
	if d.Resources == nil || d.Resources.Requests.Memory == "" || d.Resources.Requests.Memory == "0" {
		return "Shared"
	}
	return d.Resources.Requests.Memory
}

type ResourceValue struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type Resource struct {
	Requests ResourceValue `json:"requests"`
	Limits   ResourceValue `json:"limits"`
}

// Route is one row in route.list.
type Route struct {
	ID         string         `json:"id"`
	Domain     string         `json:"domain"`
	Path       string         `json:"path"`
	Status     int            `json:"status"`
	Deployment RouteDeployment `json:"deployment"`
	Location   RouteLocation   `json:"location"`
	Config     RouteConfig     `json:"config"`
}

// RouteDetail mirrors route.get; adds CreatedAt/CreatedBy and richer location.
type RouteDetail struct {
	ID         string         `json:"id"`
	Domain     string         `json:"domain"`
	Path       string         `json:"path"`
	Location   RouteLocation   `json:"location"`
	Deployment RouteDeployment `json:"deployment"`
	Config     RouteConfig     `json:"config"`
	Status     int            `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	CreatedBy  string         `json:"createdBy"`
}

type RouteDeployment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RouteLocation struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

type RouteConfig struct {
	BasicAuth   *RouteBasicAuth `json:"basicAuth"`
	RewritePath *string         `json:"rewritePath"`
}

type RouteBasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Domain is one row in domain.list.
type Domain struct {
	ID        int64     `json:"id"`
	Location  string    `json:"location"`
	Domain    string    `json:"domain"`
	Wildcard  bool      `json:"wildcard"`
	CDN       bool      `json:"cdn"`
	Status    string    `json:"status"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt"`
}

// DomainDetail mirrors domain.get.
type DomainDetail struct {
	Location     string              `json:"location"`
	Domain       string              `json:"domain"`
	Wildcard     bool                `json:"wildcard"`
	CDN          bool                `json:"cdn"`
	Status       string              `json:"status"`
	Action       string              `json:"action"`
	CreatedAt    time.Time           `json:"createdAt"`
	Verification DomainVerification  `json:"verification"`
	DNSConfig    DomainDNSConfig     `json:"dnsConfig"`
}

type DomainVerification struct {
	Ownership DomainVerificationOwnership `json:"ownership"`
	SSL       DomainVerificationSSL       `json:"ssl"`
}

type DomainVerificationOwnership struct {
	Type   string   `json:"type"`
	Name   string   `json:"name"`
	Value  string   `json:"value"`
	Errors []string `json:"errors"`
}

type DomainVerificationSSL struct {
	Pending bool `json:"pending"`
}

type DomainDNSConfig struct {
	CName []string `json:"cname"`
}

// RevisionItem is one entry in deployment.logRevision (revision history).
type RevisionItem struct {
	Revision        int        `json:"revision"`
	Image           string     `json:"image"`
	Status          int        `json:"status"`
	DeployedBy      string     `json:"deployedBy"`
	DeployedByEmail string     `json:"deployedByEmail"`
	SuccessAt       *time.Time `json:"successAt"`
	DeployedAt      time.Time  `json:"deployedAt"`
}
