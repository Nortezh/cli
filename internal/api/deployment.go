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

// CreateDeploymentOptions carries the fields backend deployment.create accepts.
// Required: image. Type defaults to WebService when empty.
// Nil pointer fields are omitted and the backend applies its defaults.
type CreateDeploymentOptions struct {
	Type       string
	Port       *int
	Protocol   *string
	Internal   *bool
	MinReplica *int
	MaxReplica *int
	Env        map[string]string
	EnvGroups  []string
	Schedule   *string
	PullSecret *string
}

// CreateDeployment creates a new deployment (revision 1) and returns its id.
// Backend method: deployment.create.
func (c *Client) CreateDeployment(ctx context.Context, project, location, name, image string, opts CreateDeploymentOptions) (string, error) {
	body := struct {
		Project    string            `json:"project"`
		Location   string            `json:"location"`
		Name       string            `json:"name"`
		Image      string            `json:"image"`
		Type       string            `json:"type"`
		MinReplica *int              `json:"minReplica"`
		MaxReplica *int              `json:"maxReplica"`
		Port       *int              `json:"port"`
		Protocol   *string           `json:"protocol"`
		Internal   *bool             `json:"internal"`
		Env        map[string]string `json:"env"`
		EnvGroups  []string          `json:"envGroups"`
		Schedule   *string           `json:"schedule"`
		PullSecret *string           `json:"pullSecret"`
	}{
		Project:    project,
		Location:   location,
		Name:       name,
		Image:      image,
		Type:       opts.Type,
		MinReplica: opts.MinReplica,
		MaxReplica: opts.MaxReplica,
		Port:       opts.Port,
		Protocol:   opts.Protocol,
		Internal:   opts.Internal,
		Env:        opts.Env,
		EnvGroups:  opts.EnvGroups,
		Schedule:   opts.Schedule,
		PullSecret: opts.PullSecret,
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := c.Invoke(ctx, "deployment.create", body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
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

// DeployOptions carries the optional fields the backend accepts as a partial
// update on deployment.deploy. nil pointers mean "no change".
//
// EnvGroups follows the backend's replace semantics: nil leaves the linked
// groups unchanged, a non-nil slice replaces them (an empty non-nil slice
// clears all links).
type DeployOptions struct {
	MinReplica *int
	MaxReplica *int
	Port       *int
	Protocol   *string
	Internal   *bool
	AddEnv     map[string]string
	RemoveEnv  []string
	EnvGroups  []string
	PullSecret *string
}

// Deploy creates a new revision of an existing deployment with the given image.
// Optional fields on opts patch the deployment (nil = leave unchanged).
// The backend returns no result body; success is signalled by a nil error.
func (c *Client) Deploy(ctx context.Context, project, location, name, image string, opts DeployOptions) error {
	body := struct {
		Project    string            `json:"project"`
		Location   string            `json:"location"`
		Name       string            `json:"name"`
		Image      string            `json:"image"`
		MinReplica *int              `json:"minReplica"`
		MaxReplica *int              `json:"maxReplica"`
		Port       *int              `json:"port"`
		Protocol   *string           `json:"protocol"`
		Internal   *bool             `json:"internal"`
		AddEnv     map[string]string `json:"addEnv"`
		RemoveEnv  []string          `json:"removeEnv"`
		EnvGroups  []string          `json:"envGroups"`
		PullSecret *string           `json:"pullSecret"`
	}{
		Project:    project,
		Location:   location,
		Name:       name,
		Image:      image,
		MinReplica: opts.MinReplica,
		MaxReplica: opts.MaxReplica,
		Port:       opts.Port,
		Protocol:   opts.Protocol,
		Internal:   opts.Internal,
		AddEnv:     opts.AddEnv,
		RemoveEnv:  opts.RemoveEnv,
		EnvGroups:  opts.EnvGroups,
		PullSecret: opts.PullSecret,
	}
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
