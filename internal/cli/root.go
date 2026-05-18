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
		Use:   "ntzh",
		Short: "Command-line client for the Nortezh platform",
		Long: `ntzh is the command-line client for the Nortezh platform.

Authentication:
  Run 'ntzh login' for interactive (browser) login, or 'ntzh login
  --service-account <email> --key-file <path>' for CI. Credentials are
  stored at ~/.config/ntzh/credentials.json (mode 0600) and expire after
  7 days; there is no refresh — re-run 'ntzh login' when prompted.

Project selection:
  Project-scoped commands (deployment ...) require --project <name|slug>
  or the NTZH_PROJECT environment variable. There is no "current project"
  state on disk — every invocation resolves the project by name.

Scripting:
  All commands support '--output json' for machine-readable output.
  Errors are written to stderr; the process exits non-zero on failure.

Configuration precedence (highest first): flag > env > config file > default.
  --server / NTZH_SERVER, --project / NTZH_PROJECT,
  NTZH_CONFIG_DIR overrides ~/.config/ntzh.`,
		Example: `  # First-time setup
  ntzh login
  ntzh project list --output json

  # Deploy a new image
  ntzh deployment deploy api --image ghcr.io/acme/api:v1.2.3 --project acme

  # CI usage
  ntzh login --service-account ci@acme.com --key-file ./key.txt
  NTZH_PROJECT=acme ntzh deployment list --output json`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&g.Server, "server", "", "API server URL (overrides config file and NTZH_SERVER)")
	cmd.PersistentFlags().StringVar(&g.Project, "project", "", "project name or slug (overrides NTZH_PROJECT); required for project-scoped commands")
	cmd.PersistentFlags().StringVar(&g.Output, "output", "table", "output format: 'table' (human) or 'json' (machine-readable)")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "log HTTP requests and responses to stderr (Authorization header redacted)")

	cmd.AddCommand(newLoginCmd(g))
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newWhoamiCmd(g))
	cmd.AddCommand(newProjectCmd(g))
	cmd.AddCommand(newDeploymentCmd(g))
	cmd.AddCommand(newSkillCmd())
	return cmd
}

