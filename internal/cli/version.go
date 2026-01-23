package cli

import (
	"fmt"

	"github.com/javanstorm/vmterminal/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version, commit hash, and build date of VMTerminal.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("VMTerminal %s\n", version.Version)
		fmt.Printf("  Commit:     %s\n", version.Commit)
		fmt.Printf("  Build Date: %s\n", version.BuildDate)
	},
}
