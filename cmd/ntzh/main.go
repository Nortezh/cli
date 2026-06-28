package main

import (
	"fmt"
	"os"

	"github.com/nortezh/cli/internal/cli"
)

// Injected at build time via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cli.NewRootCmd(version, commit, date).Execute(); err != nil {
		// AXI §6: structured errors go to stdout so agents can read them.
		fmt.Fprintln(os.Stdout, cli.FormatCLIError(err))
		os.Exit(1)
	}
}
