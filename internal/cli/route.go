package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/api"
	"github.com/nortezh/cli/internal/output"
)

func newRouteCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route",
		Short: "Manage routes (domain + path → deployment) within a project",
		Long: `Routes bind a (domain, path) pair to a web-service deployment.
The owning project must already have the domain registered via
'ntzh domain create'.`,
		Example: `  ntzh route list --project=acme
  ntzh route create --project=acme --location=bkk-1 --domain=api.acme.com --path=/ --target=api-prod
  ntzh route delete --project=acme --domain=api.acme.com --path=/`,
	}
	cmd.AddCommand(newRouteListCmd(g))
	cmd.AddCommand(newRouteGetCmd(g))
	cmd.AddCommand(newRouteCreateCmd(g))
	cmd.AddCommand(newRouteDeleteCmd(g))
	return cmd
}

func newRouteListCmd(g *Globals) *cobra.Command {
	var search string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List routes in the selected project",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pslug, err := resolveProjectSlug(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			rs, err := c.ListRoutes(cmd.Context(), pslug, search)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(rs)
		},
	}
	cmd.Flags().StringVar(&search, "search", "", "filter routes whose domain or path matches the substring")
	return cmd
}

func newRouteGetCmd(g *Globals) *cobra.Command {
	var (
		domain string
		path   string
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show a single route by (domain, path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" || path == "" {
				return fmt.Errorf("--domain and --path are required")
			}
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pslug, err := resolveProjectSlug(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			r, err := c.GetRoute(cmd.Context(), pslug, domain, path)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.Print(*r)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "route domain (required)")
	cmd.Flags().StringVar(&path, "path", "", "route path, must start with / (required)")
	return cmd
}

func newRouteCreateCmd(g *Globals) *cobra.Command {
	var (
		location    string
		domain      string
		path        string
		target      string
		rewritePath string
		skipVerify  bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a route binding (domain, path) → deployment",
		Long: `--target accepts a deployment name (e.g. 'api-prod'); the CLI prepends
the 'deployment://' scheme automatically. The target must be a web-service
deployment in the same project and location.

If --location is omitted, the CLI resolves it from the target deployment
via 'deployment.list'. Set NTZH_LOCATION to skip the lookup.`,
		Example: `  ntzh route create --project=acme --domain=api.acme.com --path=/ --target=api-prod
  ntzh route create --project=acme --domain=api.acme.com --path=/v1 --target=api-prod --rewrite-path=/$1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" || path == "" || target == "" {
				return fmt.Errorf("--domain, --path, and --target are required")
			}
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pslug, err := resolveProjectSlug(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			loc, err := resolveLocation(cmd.Context(), c, pslug, target, location)
			if err != nil {
				return err
			}
			in := api.CreateRouteInput{
				Project:          pslug,
				Location:         loc,
				Domain:           domain,
				Path:             path,
				Target:           normalizeRouteTarget(target),
				SkipDomainVerify: skipVerify,
			}
			if rewritePath != "" {
				in.RewritePath = &rewritePath
			}
			id, err := c.CreateRoute(cmd.Context(), in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "route created: %s %s%s -> %s (%s)\n", id, domain, path, target, loc)
			return nil
		},
	}
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected from target deployment if omitted; honors NTZH_LOCATION)")
	cmd.Flags().StringVar(&domain, "domain", "", "route domain, e.g. api.acme.com (required)")
	cmd.Flags().StringVar(&path, "path", "", "route path, must start with / (required)")
	cmd.Flags().StringVar(&target, "target", "", "target deployment name (required); 'deployment://' prefix is added if omitted")
	cmd.Flags().StringVar(&rewritePath, "rewrite-path", "", "optional URL path rewrite, e.g. /$1")
	cmd.Flags().BoolVar(&skipVerify, "skip-domain-verify", false, "skip the project-owns-domain check (admin/test only)")
	return cmd
}

func newRouteDeleteCmd(g *Globals) *cobra.Command {
	var (
		domain string
		path   string
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a route by (domain, path)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" || path == "" {
				return fmt.Errorf("--domain and --path are required")
			}
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pslug, err := resolveProjectSlug(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			if err := c.DeleteRoute(cmd.Context(), pslug, domain, path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "route deleted: %s%s\n", domain, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "route domain (required)")
	cmd.Flags().StringVar(&path, "path", "", "route path (required)")
	return cmd
}

func normalizeRouteTarget(t string) string {
	if strings.Contains(t, "://") {
		return t
	}
	return "deployment://" + t
}
