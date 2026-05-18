package api

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"no"`
	CreatedAt time.Time `json:"createdAt"`
}

type Deployment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ProjectID string    `json:"project_id"`
	Image     string    `json:"image"`
	Status    string    `json:"status"`
	Revision  int       `json:"revision"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Line      string    `json:"line"`
}
