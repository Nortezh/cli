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

	cmd.AddCommand(newLoginCmd(g))
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newWhoamiCmd(g))
	cmd.AddCommand(newProjectCmd(g))
	cmd.AddCommand(newDeploymentCmd(g))
	return cmd
}

// Temporary stubs — replaced when Tasks 9 and 10 create login.go, project.go,
// and deployment.go. Remove the matching stub here once the real file exists.
func newLoginCmd(*Globals) *cobra.Command      { return &cobra.Command{Use: "login"} }
func newLogoutCmd() *cobra.Command             { return &cobra.Command{Use: "logout"} }
func newWhoamiCmd(*Globals) *cobra.Command     { return &cobra.Command{Use: "whoami"} }
func newProjectCmd(*Globals) *cobra.Command    { return &cobra.Command{Use: "project"} }
func newDeploymentCmd(*Globals) *cobra.Command { return &cobra.Command{Use: "deployment"} }
