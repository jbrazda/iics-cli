// Command iics is the command-line interface for Informatica Intelligent Cloud
// Services (IICS). It sets the build version and runs the root command from
// package cmd.
package main

import (
	"os"

	"github.com/jbrazda/iics-cli/cmd"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	os.Exit(cmd.Execute())
}
