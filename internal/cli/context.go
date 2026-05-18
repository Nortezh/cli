package cli

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"nortezh-cli/internal/api"
	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

// osLookup is a tiny shim so tests can swap env lookup if needed.
var osLookup = os.Getenv

func buildClient(g *Globals) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	server := config.ResolveServer(g.Server, cfg)
	creds, err := auth.Load()
	if err != nil {
		// Allow construction without creds; commands that need them will fail
		// at Invoke() with ErrUnauthenticated.
		if !errors.Is(err, auth.ErrNoCreds) {
			return nil, err
		}
		creds = nil
	}
	return &api.Client{
		BaseURL:    server + "/user",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Creds:      creds,
		Debug:      g.Debug,
	}, nil
}

func requireProject(flag string) (string, error) {
	p := config.ResolveProject(flag)
	if p == "" {
		return "", errors.New("--project is required (or set NTZH_PROJECT)")
	}
	return p, nil
}

func resolveProjectSlug(ctx context.Context, c *api.Client, name string) (string, error) {
	ps, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range ps {
		if p.Name == name || p.Slug == name || p.ID == name {
			return p.Slug, nil
		}
	}
	return "", errors.New("project not found: " + name)
}

// resolveLocation returns flag if set, otherwise NTZH_LOCATION, otherwise
// looks up the deployment by name in the project to discover its location.
func resolveLocation(ctx context.Context, c *api.Client, project, name, flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env := osLookup("NTZH_LOCATION"); env != "" {
		return env, nil
	}
	ds, err := c.ListDeployments(ctx, project)
	if err != nil {
		return "", err
	}
	for _, d := range ds {
		if d.Name == name {
			return d.Location, nil
		}
	}
	return "", errors.New("could not resolve location for deployment " + name + "; pass --location")
}
