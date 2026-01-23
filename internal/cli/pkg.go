package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Package management for the VM",
	Long:  `Install, remove, search, and manage packages inside the VM via SSH.`,
}

var pkgInstallCmd = &cobra.Command{
	Use:   "install <packages...>",
	Short: "Install packages",
	Long:  `Install packages in the VM using apk.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPkgInstall,
}

var pkgRemoveCmd = &cobra.Command{
	Use:   "remove <packages...>",
	Short: "Remove packages",
	Long:  `Remove packages from the VM using apk.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPkgRemove,
}

var pkgSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages",
	Long:  `Search for packages in the VM using apk.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPkgSearch,
}

var pkgUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update package index",
	Long:  `Update the package index in the VM using apk.`,
	RunE:  runPkgUpdate,
}

var pkgUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade all packages",
	Long:  `Upgrade all installed packages in the VM using apk.`,
	RunE:  runPkgUpgrade,
}

var pkgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long:  `List all installed packages in the VM using apk.`,
	RunE:  runPkgList,
}

var pkgVMName string

func init() {
	// Add --vm flag to all pkg commands
	pkgCmd.PersistentFlags().StringVar(&pkgVMName, "vm", "", "VM to target (default: active VM)")

	// Add subcommands
	pkgCmd.AddCommand(pkgInstallCmd)
	pkgCmd.AddCommand(pkgRemoveCmd)
	pkgCmd.AddCommand(pkgSearchCmd)
	pkgCmd.AddCommand(pkgUpdateCmd)
	pkgCmd.AddCommand(pkgUpgradeCmd)
	pkgCmd.AddCommand(pkgListCmd)

	// Register pkg command
	rootCmd.AddCommand(pkgCmd)
}

// runSSHApkCommand runs an apk command in the VM via SSH
func runSSHApkCommand(apkArgs ...string) error {
	cfg := config.Global
	if cfg == nil {
		if err := config.Load(); err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = config.Global
	}

	// Get VM info if --vm specified
	if pkgVMName != "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}
		baseDir := filepath.Join(homeDir, ".vmterminal")
		registry := vm.NewRegistry(baseDir)

		_, err = registry.GetVM(pkgVMName)
		if err != nil {
			return fmt.Errorf("get VM: %w", err)
		}
		// Note: VM-specific SSH settings would go here in the future
	}

	// Determine host to connect to
	host := cfg.VMIP
	if host == "" {
		fmt.Println("VM IP not configured. Please either:")
		fmt.Println("  1. Find VM IP: Run 'ip addr' inside VM")
		fmt.Println("  2. Configure: Add 'vm_ip: <IP>' to ~/.vmterminal/config.yaml")
		fmt.Println()
		fmt.Println("Note: The VM must be running with SSH enabled.")
		return fmt.Errorf("VM IP not configured")
	}

	// Determine port
	port := cfg.SSHPort
	if host == "localhost" && cfg.SSHHostPort > 0 {
		port = cfg.SSHHostPort
	}

	// Build ssh command args
	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", strconv.Itoa(port),
	}

	if cfg.SSHKeyPath != "" {
		sshArgs = append(sshArgs, "-i", cfg.SSHKeyPath)
	}

	// Add user@host
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", cfg.SSHUser, host))

	// Add apk command with arguments
	sshArgs = append(sshArgs, "apk")
	sshArgs = append(sshArgs, apkArgs...)

	// Find ssh binary
	sshBin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

	// Execute ssh with full terminal control
	sshExec := exec.Command(sshBin, sshArgs...)
	sshExec.Stdin = os.Stdin
	sshExec.Stdout = os.Stdout
	sshExec.Stderr = os.Stderr

	// Run and return any error
	if err := sshExec.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

func runPkgInstall(cmd *cobra.Command, args []string) error {
	apkArgs := append([]string{"add"}, args...)
	return runSSHApkCommand(apkArgs...)
}

func runPkgRemove(cmd *cobra.Command, args []string) error {
	apkArgs := append([]string{"del"}, args...)
	return runSSHApkCommand(apkArgs...)
}

func runPkgSearch(cmd *cobra.Command, args []string) error {
	return runSSHApkCommand("search", args[0])
}

func runPkgUpdate(cmd *cobra.Command, args []string) error {
	return runSSHApkCommand("update")
}

func runPkgUpgrade(cmd *cobra.Command, args []string) error {
	return runSSHApkCommand("upgrade")
}

func runPkgList(cmd *cobra.Command, args []string) error {
	return runSSHApkCommand("list", "--installed")
}
