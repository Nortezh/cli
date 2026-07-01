package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/selfupdate"
)

func newUpgradeCmd(version string) *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Check for and install the latest ntzh release",
		Long: `Check GitHub for the latest ntzh release and, unless --check is given,
download it and replace the running binary in place.

Replacing the binary requires write access to its directory. If ntzh lives in a
system path (e.g. /usr/local/bin), re-run with sudo or use the install script.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			rel, err := selfupdate.LatestRelease(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "current: %s\n", version)
			fmt.Fprintf(w, "latest: %s\n", rel.TagName)

			if !selfupdate.Newer(version, rel.TagName) {
				fmt.Fprintln(w, "status: up to date")
				return nil
			}

			if checkOnly {
				fmt.Fprintln(w, "status: update available")
				fmt.Fprintf(w, "help: run 'ntzh upgrade' to install %s\n", rel.TagName)
				return nil
			}

			fmt.Fprintf(w, "status: upgrading %s -> %s\n", version, rel.TagName)
			if err := selfupdate.Apply(cmd.Context(), rel.TagName); err != nil {
				return err
			}
			fmt.Fprintf(w, "status: upgraded to %s\n", rel.TagName)
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only report whether a newer version exists; do not install")
	return cmd
}
