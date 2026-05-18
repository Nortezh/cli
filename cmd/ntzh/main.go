package main

import (
	"fmt"
	"os"

	"nortezh-cli/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cli.FormatCLIError(err))
		os.Exit(1)
	}
}
