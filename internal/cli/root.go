// Package cli provides the command-line interface for VMTerminal.
package cli

import (
	"fmt"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vmterminal",
	Short: "VMTerminal - Linux VM as your shell",
	Long: `VMTerminal runs a Linux VM as your default shell on macOS and Linux.

Open your terminal and you're in Linux — seamless, fast, with full access
to host files. Like WSL, but for non-Windows systems.

The goal is invisible virtualization — the VM should feel like the native OS.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it
		switch cmd.Name() {
		case "version", "completion", "status", "stop", "install", "attach", "setup", "docker-setup", "podman-setup":
			return nil
		}
		return config.Load()
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
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(setupCmd)
}
