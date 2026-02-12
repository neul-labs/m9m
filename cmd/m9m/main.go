package main

import (
	"os"

	"github.com/neul-labs/m9m/cmd/m9m/commands"
)

// Version info (set by build flags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Set version info
	commands.SetVersionInfo(Version, Commit, BuildDate)

	// Execute root command
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
