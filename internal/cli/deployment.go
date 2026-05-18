package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/api"
	"github.com/nortezh/cli/internal/output"
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
		Example: `  ntzh deployment list --project=acme
  ntzh deployment get staging-bo --project=acme --location=bkk-1
  ntzh deployment deploy staging-bo --project=acme --image=ghcr.io/acme/api:v1.2.3 --location=bkk-1
  ntzh deployment rollback staging-bo --project=acme --to=17 --location=bkk-1
  ntzh deployment revisions staging-bo --project=acme --location=bkk-1`,
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
		Example: `  ntzh deployment list --project=<project>
  ntzh deployment list --project=acme --output=json`,
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
		Example: `  ntzh deployment get <deployment> --project=<project> --location=<location>
  ntzh deployment get staging-bo --project=acme --location=bkk-1 --output=json`,
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
		image      string
		location   string
		setEnv     []string
		removeEnv  []string
		port       int
		protocol   string
		internal   bool
		minReplica int
		maxReplica int
	)
	cmd := &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy a new container image to a deployment",
		Long: `Create a new revision of <name> in the selected project, running the
container image referenced by --image. The backend pulls the image and
rolls forward; on success the call returns with no body (exit 0).

Optional --set-env / --remove-env / --port / --protocol / --internal /
--min-replica / --max-replica flags patch the deployment in the same
revision; omitted flags leave the existing value unchanged.

Use 'ntzh deployment rollback' to revert if needed.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment deploy <deployment> --project=<project> --image=<image> --location=<location>
  ntzh deployment deploy staging-bo --project=acme --image=ghcr.io/acme/api:v1.2.3 --location=bkk-1
  ntzh deployment deploy api --project=acme --image=img:v2 --set-env DB_URL=postgres://... --set-env DEBUG=true
  ntzh deployment deploy api --project=acme --image=img:v2 --remove-env STALE_FLAG --port 8080`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if image == "" {
				return fmt.Errorf("--image is required")
			}
			addEnv, err := parseSetEnv(setEnv)
			if err != nil {
				return err
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
			opts := api.DeployOptions{AddEnv: addEnv, RemoveEnv: removeEnv}
			if cmd.Flags().Changed("port") {
				opts.Port = &port
			}
			if cmd.Flags().Changed("protocol") {
				opts.Protocol = &protocol
			}
			if cmd.Flags().Changed("internal") {
				opts.Internal = &internal
			}
			if cmd.Flags().Changed("min-replica") {
				opts.MinReplica = &minReplica
			}
			if cmd.Flags().Changed("max-replica") {
				opts.MaxReplica = &maxReplica
			}
			if err := c.Deploy(cmd.Context(), pslug, loc, args[0], image, opts); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: deploying %s\n", args[0], image)
			return nil
		},
	}
	cmd.Flags().StringVar(&image, "image", "", "container image reference, e.g. ghcr.io/acme/api:v1.2.3 (required)")
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected if omitted; honors NTZH_LOCATION)")
	cmd.Flags().StringArrayVar(&setEnv, "set-env", nil, "merge env var KEY=VALUE (repeatable)")
	cmd.Flags().StringArrayVar(&removeEnv, "remove-env", nil, "remove env var by KEY (repeatable)")
	cmd.Flags().IntVar(&port, "port", 0, "container port the service listens on")
	cmd.Flags().StringVar(&protocol, "protocol", "", "service protocol (http, http2, tcp, ...)")
	cmd.Flags().BoolVar(&internal, "internal", false, "expose only inside the cluster (no public URL)")
	cmd.Flags().IntVar(&minReplica, "min-replica", 0, "minimum replica count")
	cmd.Flags().IntVar(&maxReplica, "max-replica", 0, "maximum replica count")
	return cmd
}

// parseSetEnv parses KEY=VALUE pairs from repeated --set-env flags.
func parseSetEnv(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(pairs))
	for _, kv := range pairs {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			return nil, fmt.Errorf("--set-env: expected KEY=VALUE, got %q", kv)
		}
		out[kv[:i]] = kv[i+1:]
	}
	return out, nil
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

Pass --to <revision> to pick non-interactively. When omitted (and stdin
is a terminal) the CLI fetches the revision history and prompts you to
choose one.

Use 'ntzh deployment revisions <name>' to inspect revision numbers up
front.

If --location is omitted the CLI resolves it from 'deployment.list'.`,
		Example: `  ntzh deployment rollback <deployment> --project=<project> --to=<revision> --location=<location>
  ntzh deployment rollback staging-bo --project=acme --to=17 --location=bkk-1
  ntzh deployment rollback staging-bo --project=acme  # interactive picker`,
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
			if to <= 0 {
				revs, err := c.ListRevisions(cmd.Context(), pslug, loc, args[0])
				if err != nil {
					return err
				}
				to, err = pickRevisionInteractive(cmd, revs)
				if err != nil {
					return err
				}
			}
			if err := c.Rollback(cmd.Context(), pslug, loc, args[0], to); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s rolled back to revision %d\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().IntVar(&to, "to", 0, "target revision number (omit for an interactive picker)")
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (auto-detected if omitted; honors NTZH_LOCATION)")
	return cmd
}

// pickRevisionInteractive prints a numbered list of revisions to stderr and
// reads the user's choice from stdin. Returns the chosen revision number.
// Errors out when stdin is not a terminal so non-interactive callers must
// pass --to explicitly.
func pickRevisionInteractive(cmd *cobra.Command, items []api.RevisionItem) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("no revisions available to roll back to")
	}
	if fi, err := os.Stdin.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return 0, fmt.Errorf("--to <revision> is required (stdin is not a terminal)")
	}
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w, "Select a revision to roll back to:")
	for i, it := range items {
		when := it.DeployedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "  [%d] revision %d  %s  %s  by %s\n", i+1, it.Revision, when, it.Image, it.DeployedByEmail)
	}
	fmt.Fprintf(w, "Enter choice (1-%d): ", len(items))
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && line == "" {
		return 0, fmt.Errorf("read choice: %w", err)
	}
	line = strings.TrimSpace(line)
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(items) {
		return 0, fmt.Errorf("invalid choice %q", line)
	}
	return items[idx-1].Revision, nil
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
		Example: `  ntzh deployment revisions <deployment> --project=<project> --location=<location>
  ntzh deployment revisions staging-bo --project=acme --location=bkk-1 --output=json`,
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
