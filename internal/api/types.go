package api

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SID       string    `json:"sid"`
	CreatedAt time.Time `json:"created_at"`
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
