package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload VM after config changes",
	Long: `Reload the VM configuration. This command re-sources the shell configuration
to pick up any changes to the vmterminal integration.

If the VM is currently running, you should stop it first and then start it again
to apply configuration changes.`,
	RunE: runReload,
}

func runReload(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	// Determine which shell config file to source
	shell := os.Getenv("SHELL")
	var configFile string

	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(homeDir, ".zshrc")
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(homeDir, ".bashrc")
	default:
		configFile = filepath.Join(homeDir, ".profile")
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("Shell configuration file not found.")
		fmt.Println("No changes to reload.")
		return nil
	}

	fmt.Printf("Reloading shell configuration from %s...\n", configFile)

	// Execute the shell with the source command
	// Note: This won't actually affect the parent shell, but we inform the user
	shellCmd := exec.Command(shell, "-c", fmt.Sprintf("source %s && echo 'Configuration reloaded.'", configFile))
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr
	if err := shellCmd.Run(); err != nil {
		fmt.Printf("Warning: Could not source %s: %v\n", configFile, err)
	}

	fmt.Println()
	fmt.Println("Note: To fully apply changes to your current terminal session,")
	fmt.Println("please run one of the following:")
	fmt.Printf("  source %s\n", configFile)
	fmt.Println("  or restart your terminal")
	fmt.Println()
	fmt.Println("To start the VM with new configuration:")
	fmt.Println("  vmterminal run")

	return nil
}
