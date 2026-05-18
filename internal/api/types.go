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
