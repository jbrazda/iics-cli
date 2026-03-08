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
