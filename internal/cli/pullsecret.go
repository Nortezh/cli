package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/output"
)

func newPullSecretCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pull-secret",
		Aliases: []string{"pullsecret"},
		Short:   "Manage private registry pull secrets for a project",
		Long: `Pull secrets hold the credentials used to pull container images from a
private registry. They are project-scoped; attach one to a deployment with
'ntzh deployment deploy --pull-secret <name>' (or on 'deployment create').`,
		Example: `  ntzh pull-secret list --project=acme
  ntzh pull-secret create ghcr --project=acme --registry=ghcr.io --username=bot --password=$TOKEN
  ntzh pull-secret get ghcr --project=acme
  ntzh pull-secret delete ghcr --project=acme`,
	}
	cmd.AddCommand(newPullSecretListCmd(g))
	cmd.AddCommand(newPullSecretGetCmd(g))
	cmd.AddCommand(newPullSecretCreateCmd(g))
	cmd.AddCommand(newPullSecretDeleteCmd(g))
	return cmd
}

func newPullSecretListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List pull secrets in the selected project",
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
			ss, err := c.ListPullSecrets(cmd.Context(), pslug)
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.PrintList(ss)
		},
	}
}

func newPullSecretGetCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show details for one pull secret (password is never returned)",
		Args:  cobra.ExactArgs(1),
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
			s, err := c.GetPullSecret(cmd.Context(), pslug, args[0])
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return p.Print(*s)
		},
	}
}

func newPullSecretCreateCmd(g *Globals) *cobra.Command {
	var (
		registry string
		username string
		password string
	)
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a pull secret for a private registry",
		Long: `Store credentials for a private container registry under <name>.
Required: --registry, --username, --password. Prefer passing the password
from an environment variable (e.g. --password=$TOKEN) over typing it inline.`,
		Example: `  ntzh pull-secret create ghcr --project=acme --registry=ghcr.io --username=bot --password=$TOKEN`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if registry == "" || username == "" || password == "" {
				return fmt.Errorf("--registry, --username, and --password are required")
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
			if err := c.CreatePullSecret(cmd.Context(), pslug, args[0], registry, username, password); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pull secret created: %s (%s)\n", args[0], registry)
			return nil
		},
	}
	cmd.Flags().StringVar(&registry, "registry", "", "registry host, e.g. ghcr.io (required)")
	cmd.Flags().StringVar(&username, "username", "", "registry username (required)")
	cmd.Flags().StringVar(&password, "password", "", "registry password or token (required)")
	return cmd
}

func newPullSecretDeleteCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a pull secret from the project",
		Args:  cobra.ExactArgs(1),
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
			if err := c.DeletePullSecret(cmd.Context(), pslug, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pull secret deleted: %s\n", args[0])
			return nil
		},
	}
}
