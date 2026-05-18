package api

import "context"

func (c *Client) ListDeployments(ctx context.Context, projectID string) ([]Deployment, error) {
	var out struct {
		Items []Deployment `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.list",
		map[string]string{"project_id": projectID}, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetDeployment(ctx context.Context, projectID, name string) (*Deployment, error) {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
	}{projectID, name}
	var out Deployment
	if err := c.Invoke(ctx, "deployment.get", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Deploy(ctx context.Context, projectID, name, image string) (*Deployment, error) {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Image     string `json:"image"`
	}{projectID, name, image}
	var out Deployment
	if err := c.Invoke(ctx, "deployment.deploy", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Rollback(ctx context.Context, projectID, name string, revision int) error {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Revision  int    `json:"revision"`
	}{projectID, name, revision}
	return c.Invoke(ctx, "deployment.rollback", body, nil)
}

func (c *Client) LogRevision(ctx context.Context, projectID, name string, revision int) ([]LogLine, error) {
	body := struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Revision  int    `json:"revision"`
	}{projectID, name, revision}
	var out struct {
		Items []LogLine `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.logRevision", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
