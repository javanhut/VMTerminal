// Package cli provides the command-line interface for VMTerminal.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vmterminal",
	Short: "VMTerminal - Linux VM with built-in GUI terminal",
	Long: `VMTerminal runs a Linux VM with a built-in GUI terminal emulator.

One intelligent command that handles everything interactively.
No separate setup steps, no static config files.

Just run 'vmterminal' or 'vmterminal run --distro arch' and it does everything.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	// When run without subcommand, execute 'run'
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRun(cmd, args)
	},
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(versionCmd)
}
