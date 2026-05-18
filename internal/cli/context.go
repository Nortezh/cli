package cli

import (
	"context"
	"errors"
	"net/http"
	"time"

	"nortezh-cli/internal/api"
	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

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

func resolveProjectID(ctx context.Context, c *api.Client, name string) (string, error) {
	ps, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range ps {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", errors.New("project not found: " + name)
}
