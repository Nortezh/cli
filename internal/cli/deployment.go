package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"nortezh-cli/internal/output"
)

func newDeploymentCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "Manage deployments within a project",
		Long: `Deployment commands act on the deployments of a single project.

All subcommands require --project <name|slug> (or NTZH_PROJECT).
Subcommands that target a specific deployment (get, deploy, rollback,
revisions) also need its --location (cluster) — when omitted the CLI
looks it up via 'deployment.list'. Set NTZH_LOCATION to skip the lookup.

A "deployment" is identified by its name (e.g. 'api', 'web') and has
numbered revisions you can roll back to or inspect.`,
		Example: `  ntzh deployment list --project acme
  ntzh deployment get api --project acme --output json
  ntzh deployment deploy api --image ghcr.io/acme/api:v1.2.3 --project acme
  ntzh deployment rollback api --to 17 --project acme
  ntzh deployment revisions api --project acme`,
	}
	cmd.AddCommand(newDeploymentListCmd(g))
	cmd.AddCommand(newDeploymentGetCmd(g))
	cmd.AddCommand(newDeploymentDeployCmd(g))
	cmd.AddCommand(newDeploymentRollbackCmd(g))
	cmd.AddCommand(newDeploymentRevisionsCmd(g))
	return cmd
}

func newDeploymentListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all deployments in the selected project",
		Long: `List every deployment in the project resolved from --project / NTZH_PROJECT.

Output columns (table): name, type, status, location, replicas, last_deployed.
Use '--output json' for the full structured response.`,
		Example: `  ntzh deployment list --project acme
  ntzh deployment list --project acme --output json`,
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
			ds, err := c.ListDeployments(cmd.Context(), pslug)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(ds)
		},
	}
}

func newDeploymentGetCmd(g *Globals) *cobra.Command {
	var location string
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Show details for a single deployment",
		Long: `Fetch one deployment by its name (e.g. 'api', 'web') within the
selected project. Returns the current revision, image reference, status,
replica counts, memory request, and routing URLs.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment get api --project acme
  ntzh deployment get api --project acme --location bkk-1 --output json`,
		Args: cobra.ExactArgs(1),
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
			loc, err := resolveLocation(cmd.Context(), c, pslug, args[0], location)
			if err != nil {
				return err
			}
			d, err := c.GetDeployment(cmd.Context(), pslug, loc, args[0])
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.Print(*d)
		},
	}
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected from deployment list if omitted; honors NTZH_LOCATION)")
	return cmd
}

func newDeploymentDeployCmd(g *Globals) *cobra.Command {
	var (
		image    string
		location string
	)
	cmd := &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy a new container image to a deployment",
		Long: `Create a new revision of <name> in the selected project, running the
container image referenced by --image. The backend pulls the image and
rolls forward; on success the call returns with no body (exit 0).

Use 'ntzh deployment rollback' to revert if needed.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment deploy <deployment> --project=<project> --image=<image> --location=<location>
  ntzh deployment deploy staging-bo --project=acme --image=ghcr.io/acme/api:v1.2.3 --location=bkk-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if image == "" {
				return fmt.Errorf("--image is required")
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
			loc, err := resolveLocation(cmd.Context(), c, pslug, args[0], location)
			if err != nil {
				return err
			}
			if err := c.Deploy(cmd.Context(), pslug, loc, args[0], image); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: deploying %s\n", args[0], image)
			return nil
		},
	}
	cmd.Flags().StringVar(&image, "image", "", "container image reference, e.g. ghcr.io/acme/api:v1.2.3 (required)")
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected if omitted; honors NTZH_LOCATION)")
	return cmd
}

func newDeploymentRollbackCmd(g *Globals) *cobra.Command {
	var (
		to       int
		location string
	)
	cmd := &cobra.Command{
		Use:   "rollback <name>",
		Short: "Roll a deployment back to a previous revision number",
		Long: `Re-promote a previous revision of <name> as the live revision.

Use 'ntzh deployment revisions <name>' to discover revision numbers.
--to must be a positive integer.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment rollback api --to 17 --project acme`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if to <= 0 {
				return fmt.Errorf("--to <revision> is required")
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
			loc, err := resolveLocation(cmd.Context(), c, pslug, args[0], location)
			if err != nil {
				return err
			}
			if err := c.Rollback(cmd.Context(), pslug, loc, args[0], to); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s rolled back to revision %d\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().IntVar(&to, "to", 0, "target revision number to roll back to (must be > 0, required)")
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected if omitted; honors NTZH_LOCATION)")
	return cmd
}

func newDeploymentRevisionsCmd(g *Globals) *cobra.Command {
	var location string
	cmd := &cobra.Command{
		Use:     "revisions <name>",
		Aliases: []string{"logs"},
		Short:   "List the revision history of a deployment (newest first)",
		Long: `List every recorded revision of <name>: revision number, image,
status (1=pending, 2=deploying, 3=deployed), who deployed it, and when.

This is what backend method deployment.logRevision actually returns —
revision history, not pod log lines. For runtime pod logs, fetch the
signed 'logUrl' from 'ntzh deployment get <name>' and stream from there.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment revisions api --project acme
  ntzh deployment revisions api --project acme --output json`,
		Args: cobra.ExactArgs(1),
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
			loc, err := resolveLocation(cmd.Context(), c, pslug, args[0], location)
			if err != nil {
				return err
			}
			items, err := c.ListRevisions(cmd.Context(), pslug, loc, args[0])
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(items)
		},
	}
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected if omitted; honors NTZH_LOCATION)")
	return cmd
}
