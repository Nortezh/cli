package api

import "context"

func (c *Client) ListPullSecrets(ctx context.Context, project string) ([]PullSecret, error) {
	body := struct {
		Project  string   `json:"project"`
		Paginate paginate `json:"paginate"`
	}{project, paginate{Page: 1, PerPage: 200}}
	var out struct {
		Items []PullSecret `json:"items"`
	}
	if err := c.Invoke(ctx, "pullSecret.list", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetPullSecret(ctx context.Context, project, name string) (*PullSecretDetail, error) {
	body := struct {
		Project string `json:"project"`
		Name    string `json:"name"`
	}{project, name}
	var out PullSecretDetail
	if err := c.Invoke(ctx, "pullSecret.get", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreatePullSecret(ctx context.Context, project, name, registry, username, password string) error {
	body := struct {
		Project  string `json:"project"`
		Name     string `json:"name"`
		Registry string `json:"registry"`
		Username string `json:"username"`
		Password string `json:"password"`
	}{project, name, registry, username, password}
	return c.Invoke(ctx, "pullSecret.create", body, nil)
}

func (c *Client) DeletePullSecret(ctx context.Context, project, name string) error {
	body := struct {
		Project string `json:"project"`
		Name    string `json:"name"`
	}{project, name}
	return c.Invoke(ctx, "pullSecret.delete", body, nil)
}
