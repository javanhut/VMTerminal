package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/spf13/cobra"
)

var podmanCmd = &cobra.Command{
	Use:                "podman [args...]",
	Short:              "Run Podman commands in the VM",
	Long:               `Execute Podman commands inside the VM via SSH. Passes all arguments to podman.`,
	RunE:               runPodman,
	DisableFlagParsing: true, // Pass all args to podman
}

func init() {
	rootCmd.AddCommand(podmanCmd)
}

func runPodman(cmd *cobra.Command, args []string) error {
	cfg := config.Global
	if cfg == nil {
		if err := config.Load(); err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg = config.Global
	}

	// Determine host to connect to
	host := cfg.VMIP
	if host == "" {
		fmt.Println("VM IP not configured. Please either:")
		fmt.Println("  1. Find VM IP: Run 'ip addr' inside VM")
		fmt.Println("  2. Configure: Add 'vm_ip: <IP>' to ~/.vmterminal/config.yaml")
		fmt.Println()
		fmt.Println("For setup instructions, run: vmterminal podman-setup")
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

	// Add podman command and all arguments
	sshArgs = append(sshArgs, "podman")
	sshArgs = append(sshArgs, args...)

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
