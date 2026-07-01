package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/auth"
	"github.com/nortezh/cli/internal/output"
)

// Globals holds parsed values of the global persistent flags.
type Globals struct {
	Server  string
	Project string
	Output  string
	Debug   bool
}

func NewRootCmd(version, commit, date string) *cobra.Command {
	g := &Globals{}
	cmd := &cobra.Command{
		Use:     "ntzh",
		Version: version,
		Short:   "Command-line client for the Nortezh platform",
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
  Output defaults to TOON (compact, agent-friendly). Use '--output json'
  for raw JSON or '--output table' for aligned columns. Errors print to
  stdout in the same format; the process exits non-zero on failure.

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
			return runHome(cmd, g)
		},
	}
	cmd.SetVersionTemplate(fmt.Sprintf("ntzh {{.Version}} (commit %s, built %s)\n", commit, date))

	cmd.PersistentFlags().StringVar(&g.Server, "server", "", "API server URL (overrides config file and NTZH_SERVER)")
	cmd.PersistentFlags().StringVar(&g.Project, "project", "", "project name or slug (overrides NTZH_PROJECT); required for project-scoped commands")
	cmd.PersistentFlags().StringVar(&g.Output, "output", "toon", "output format: 'toon' (default, compact), 'table' (human), or 'json' (machine-readable)")
	cmd.PersistentFlags().BoolVar(&g.Debug, "debug", false, "log HTTP requests and responses to stderr (Authorization header redacted)")

	cmd.AddCommand(newLoginCmd(g))
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newWhoamiCmd(g))
	cmd.AddCommand(newProjectCmd(g))
	cmd.AddCommand(newDeploymentCmd(g))
	cmd.AddCommand(newRouteCmd(g))
	cmd.AddCommand(newDomainCmd(g))
	cmd.AddCommand(newPullSecretCmd(g))
	cmd.AddCommand(newSkillCmd())
	cmd.AddCommand(newUpgradeCmd(version))
	return cmd
}

// runHome renders the content-first home view (AXI §8/§10): the tool's identity
// followed by live state when the user is authenticated, or a login hint when
// not. It never makes a network call for an unauthenticated user.
func runHome(cmd *cobra.Command, g *Globals) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "bin: %s\n", execPath())
	fmt.Fprintf(w, "description: %s\n", cmd.Short)

	if _, err := auth.Load(); err == nil {
		if c, err := buildClient(g); err == nil {
			if ps, err := c.ListProjects(cmd.Context()); err == nil {
				if p, err := output.NewPrinter(g.Output, w); err == nil {
					_ = p.PrintList(ps)
				}
				output.Hints(w, g.Output,
					"ntzh deployment list --project=<name>",
					"ntzh project list")
				return nil
			}
		}
	}
	output.Hints(w, g.Output,
		"ntzh login    # authenticate (browser or --service-account)",
		"ntzh project list")
	return nil
}

// execPath returns the absolute path of the running binary with the user's
// home directory collapsed to ~, falling back to the bare command name.
func execPath() string {
	p, err := os.Executable()
	if err != nil || p == "" {
		return "ntzh"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
