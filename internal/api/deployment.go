package api

import "context"

type paginate struct {
	Page    int `json:"page"`
	PerPage int `json:"perPage"`
}

func (c *Client) ListDeployments(ctx context.Context, project string) ([]Deployment, error) {
	body := struct {
		Paginate paginate `json:"paginate"`
		Project  string   `json:"project"`
	}{paginate{Page: 1, PerPage: 40}, project}
	var out struct {
		Items []Deployment `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.list", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetDeployment(ctx context.Context, project, location, name string) (*DeploymentDetail, error) {
	body := struct {
		Project  string `json:"project"`
		Location string `json:"location"`
		Name     string `json:"name"`
	}{project, location, name}
	var out DeploymentDetail
	if err := c.Invoke(ctx, "deployment.get", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deploy creates a new revision of an existing deployment with the given image.
// The backend returns no result body; success is signalled by a nil error.
func (c *Client) Deploy(ctx context.Context, project, location, name, image string) error {
	body := struct {
		Project  string `json:"project"`
		Location string `json:"location"`
		Name     string `json:"name"`
		Image    string `json:"image"`
	}{project, location, name, image}
	return c.Invoke(ctx, "deployment.deploy", body, nil)
}

func (c *Client) Rollback(ctx context.Context, project, location, name string, revision int) error {
	body := struct {
		Project  string `json:"project"`
		Location string `json:"location"`
		Name     string `json:"name"`
		Revision int    `json:"revision"`
	}{project, location, name, revision}
	return c.Invoke(ctx, "deployment.rollback", body, nil)
}

// ListRevisions returns the revision history for a deployment, newest first.
// Backend method: deployment.logRevision (returns revision items, not log lines).
func (c *Client) ListRevisions(ctx context.Context, project, location, name string) ([]RevisionItem, error) {
	body := struct {
		Project  string `json:"project"`
		Location string `json:"location"`
		Name     string `json:"name"`
	}{project, location, name}
	var out struct {
		Items []RevisionItem `json:"items"`
	}
	if err := c.Invoke(ctx, "deployment.logRevision", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}
