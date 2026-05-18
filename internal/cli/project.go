package cli

import (
	"github.com/spf13/cobra"

	"nortezh-cli/internal/output"
)

func newProjectCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(newProjectListCmd(g))
	return cmd
}

func newProjectListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
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
			return p.PrintList(ps)
		},
	}
}
