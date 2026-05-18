package api

import "context"

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out struct {
		Items []Project `json:"items"`
	}
	if err := c.Invoke(ctx, "project.list", nil, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
