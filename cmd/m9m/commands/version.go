package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("m9m version %s\n", version)
		if commit != "unknown" {
			fmt.Printf("  commit: %s\n", commit)
		}
		if buildDate != "unknown" {
			fmt.Printf("  built:  %s\n", buildDate)
		}
	},
}
