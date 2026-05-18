package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"nortezh-cli/internal/output"
)

func newDeploymentCmd(g *Globals) *cobra.Command {
	cmd := &cobra.Command{Use: "deployment", Short: "Manage deployments"}
	cmd.AddCommand(newDeploymentListCmd(g))
	cmd.AddCommand(newDeploymentGetCmd(g))
	cmd.AddCommand(newDeploymentDeployCmd(g))
	cmd.AddCommand(newDeploymentRollbackCmd(g))
	cmd.AddCommand(newDeploymentLogsCmd(g))
	return cmd
}

func newDeploymentListCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List deployments in the project",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireProject(g.Project)
			if err != nil {
				return err
			}
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			ds, err := c.ListDeployments(cmd.Context(), pid)
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
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Show a single deployment",
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
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			d, err := c.GetDeployment(cmd.Context(), pid, args[0])
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

func newDeploymentDeployCmd(g *Globals) *cobra.Command {
	var image string
	cmd := &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy a new image to a deployment",
		Args:  cobra.ExactArgs(1),
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
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			d, err := c.Deploy(cmd.Context(), pid, args[0], image)
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
	cmd.Flags().StringVar(&image, "image", "", "container image reference")
	return cmd
}

func newDeploymentRollbackCmd(g *Globals) *cobra.Command {
	var to int
	cmd := &cobra.Command{
		Use:   "rollback <name>",
		Short: "Roll a deployment back to a previous revision",
		Args:  cobra.ExactArgs(1),
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
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			if err := c.Rollback(cmd.Context(), pid, args[0], to); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s rolled back to revision %d\n", args[0], to)
			return nil
		},
	}
	cmd.Flags().IntVar(&to, "to", 0, "revision to roll back to")
	return cmd
}

func newDeploymentLogsCmd(g *Globals) *cobra.Command {
	var revision int
	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Print logs for a deployment revision",
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
			pid, err := resolveProjectID(cmd.Context(), c, name)
			if err != nil {
				return err
			}
			lines, err := c.LogRevision(cmd.Context(), pid, args[0], revision)
			if err != nil {
				return err
			}
			for _, l := range lines {
				fmt.Fprintln(cmd.OutOrStdout(), l.Line)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&revision, "revision", 0, "revision number (0 = latest)")
	return cmd
}
