package api

import "context"

func (c *Client) ListDomains(ctx context.Context, project string) ([]Domain, error) {
	body := struct {
		Project  string   `json:"project"`
		Paginate paginate `json:"paginate"`
	}{project, paginate{Page: 1, PerPage: 200}}
	var out struct {
		Items []Domain `json:"items"`
	}
	if err := c.Invoke(ctx, "domain.list", body, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetDomain(ctx context.Context, project, domain string) (*DomainDetail, error) {
	body := struct {
		Project string `json:"project"`
		Domain  string `json:"domain"`
	}{project, domain}
	var out DomainDetail
	if err := c.Invoke(ctx, "domain.get", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateDomain(ctx context.Context, project, location, domain string, wildcard, cdn bool) error {
	body := struct {
		Project  string `json:"project"`
		Location string `json:"location"`
		Domain   string `json:"domain"`
		Wildcard bool   `json:"wildcard"`
		CDN      bool   `json:"cdn"`
	}{project, location, domain, wildcard, cdn}
	return c.Invoke(ctx, "domain.create", body, nil)
}

func (c *Client) DeleteDomain(ctx context.Context, project, domain string) error {
	body := struct {
		Project string `json:"project"`
		Domain  string `json:"domain"`
	}{project, domain}
	return c.Invoke(ctx, "domain.delete", body, nil)
}
