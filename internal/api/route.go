package api

import "context"

func (c *Client) ListRoutes(ctx context.Context, project, search string) ([]Route, error) {
	body := struct {
		Project  string   `json:"project"`
		Search   string   `json:"search"`
		Paginate paginate `json:"paginate"`
	}{project, search, paginate{Page: 1, PerPage: 200}}
	var out struct {
		Items []Route `json:"items"`
	}
	if err := c.Invoke(ctx, "route.list", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetRoute(ctx context.Context, project, domain, path string) (*RouteDetail, error) {
	body := struct {
		Project string `json:"project"`
		Domain  string `json:"domain"`
		Path    string `json:"path"`
	}{project, domain, path}
	var out RouteDetail
	if err := c.Invoke(ctx, "route.get", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateRouteInput struct {
	Project          string
	Location         string
	Domain           string
	Path             string
	Target           string
	RewritePath      *string
	SkipDomainVerify bool
}

func (c *Client) CreateRoute(ctx context.Context, in CreateRouteInput) (string, error) {
	body := struct {
		Project          string      `json:"project"`
		Location         string      `json:"location"`
		Domain           string      `json:"domain"`
		Path             string      `json:"path"`
		Target           string      `json:"target"`
		Config           RouteConfig `json:"config"`
		SkipDomainVerify bool        `json:"skipDomainVerify"`
	}{
		Project:          in.Project,
		Location:         in.Location,
		Domain:           in.Domain,
		Path:             in.Path,
		Target:           in.Target,
		Config:           RouteConfig{RewritePath: in.RewritePath},
		SkipDomainVerify: in.SkipDomainVerify,
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := c.Invoke(ctx, "route.create", body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) DeleteRoute(ctx context.Context, project, domain, path string) error {
	body := struct {
		Project string `json:"project"`
		Domain  string `json:"domain"`
		Path    string `json:"path"`
	}{project, domain, path}
	return c.Invoke(ctx, "route.delete", body, nil)
}
