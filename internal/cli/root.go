// Package cli provides the command-line interface for VMTerminal.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vmterminal",
	Short: "VMTerminal - Linux VM as your terminal",
	Long: `VMTerminal runs a Linux VM as your default terminal on macOS and Linux.

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

// IsLoginShell returns true if invoked as a login shell.
// Login shells are invoked with argv[0] prefixed with '-' by convention.
func IsLoginShell() bool {
	if len(os.Args) == 0 {
		return false
	}
	arg0 := os.Args[0]
	if len(arg0) == 0 {
		return false
	}
	return arg0[0] == '-'
}

// RunShellMode runs vmterminal as a login shell.
// This is called from main.go when detected as login shell.
func RunShellMode() {
	// Enable quiet mode for login shell
	SetQuietMode(true)

	// Run the VM
	if err := runRun(nil, nil); err != nil {
		// Only print errors that aren't expected conditions
		errMsg := err.Error()
		if errMsg != "no TTY detected; vmterminal requires a terminal" {
			fmt.Fprintf(os.Stderr, "vmterminal: %v\n", err)
		}
		os.Exit(1)
	}
}

func init() {
	// Add subcommands - minimal set per new design
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(defaultCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(versionCmd)
}
