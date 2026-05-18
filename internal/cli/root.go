package cli

import "github.com/spf13/cobra"

// Globals holds parsed values of the global persistent flags.
type Globals struct {
	Server  string
	Project string
	Output  string
	Debug   bool
}

func NewRootCmd() *cobra.Command {
	g := &Globals{}
	cmd := &cobra.Command{
		Use:           "ntzh",
		Short:         "Command-line client for the Nortezh platform",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&g.Server, "server", "", "server URL (overrides config and NTZH_SERVER)")
	cmd.PersistentFlags().StringVar(&g.Project, "project", "", "project name (or NTZH_PROJECT)")
	cmd.PersistentFlags().StringVar(&g.Output, "output", "table", "output format: table|json")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "log HTTP traffic to stderr (token redacted)")
	return cmd
}
