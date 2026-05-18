package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/output"
)

func newDomainCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage custom domains for a project",
		Long: `Register and inspect custom domains that the project can serve
routes from. Domains are tied to a single cluster (--location).`,
		Example: `  ntzh domain list --project=acme
  ntzh domain create --project=acme --location=bkk-1 api.acme.com
  ntzh domain get --project=acme api.acme.com
  ntzh domain delete --project=acme api.acme.com`,
	}
	cmd.AddCommand(newDomainListCmd(g))
	cmd.AddCommand(newDomainGetCmd(g))
	cmd.AddCommand(newDomainCreateCmd(g))
	cmd.AddCommand(newDomainDeleteCmd(g))
	return cmd
}

func newDomainListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List domains in the selected project",
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
			ds, err := c.ListDomains(cmd.Context(), pslug)
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

func newDomainGetCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "get <domain>",
		Short: "Show details for one domain (verification, DNS hints)",
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
			d, err := c.GetDomain(cmd.Context(), pslug, args[0])
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
}

func newDomainCreateCmd(g *Globals) *cobra.Command {
	var (
		location string
		wildcard bool
		cdn      bool
	)
	cmd := &cobra.Command{
		Use:   "create <domain>",
		Short: "Register a domain for the project",
		Long: `Register <domain> on the given --location (cluster). For wildcard
domains (e.g. *.acme.com) pass --wildcard. Pass --cdn to enable the
Nortezh CDN (paid feature; trial billing is rejected).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if location == "" {
				return fmt.Errorf("--location is required")
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
			if err := c.CreateDomain(cmd.Context(), pslug, location, args[0], wildcard, cdn); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "domain registered: %s (%s)\n", args[0], location)
			return nil
		},
	}
	cmd.Flags().StringVar(&location, "location", "", "cluster/location ID (required)")
	cmd.Flags().BoolVar(&wildcard, "wildcard", false, "treat <domain> as a wildcard (e.g. *.acme.com)")
	cmd.Flags().BoolVar(&cdn, "cdn", false, "enable Nortezh CDN for this domain (paid)")
	return cmd
}

func newDomainDeleteCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain>",
		Short: "Delete a domain from the project",
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
			if err := c.DeleteDomain(cmd.Context(), pslug, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "domain deleted: %s\n", args[0])
			return nil
		},
	}
}
