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
		fmt.Fprintln(os.Stderr, cli.FormatCLIError(err))
		os.Exit(1)
	}
}
