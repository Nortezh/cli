package cli

import (
	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/output"
)

func newProjectCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "List and inspect projects you have access to",
		Long: `Project commands operate on the projects visible to the authenticated
account. Use the 'name' or 'slug' field from 'project list' as the value
for --project / NTZH_PROJECT on other commands.`,
	}
	cmd.AddCommand(newProjectListCmd(g))
	return cmd
}

func newProjectListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects the authenticated account can access",
		Long: `List every project visible to the current credentials.

Use '--output json' to get the full structured response, including each
project's 'slug' which is the stable identifier accepted by --project.`,
		Example: `  ntzh project list
  ntzh project list --output json | jq '.[].slug'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			ps, err := c.ListProjects(cmd.Context())
			if err != nil {
				return err
			}
			p, err := output.NewPrinter(g.Output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if err := p.PrintList(ps); err != nil {
				return err
			}
			output.Hints(cmd.OutOrStdout(), g.Output,
				"ntzh deployment list --project=<name>")
			return nil
		},
	}
}
