package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running VM",
	Long: `Stop the running VM gracefully.

If the VM is running in the foreground, use Ctrl+C instead.
This command attempts to stop a VM that may be running in another terminal.`,
	RunE: runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	vmName := "default"

	// Check for PID file
	pidFile := filepath.Join(baseDir, "data", vmName, "vm.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("VM is not running (no PID file found).")
			fmt.Println("To start the VM, run: vmterminal run")
			return nil
		}
		return fmt.Errorf("read PID file: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("parse PID: %w", err)
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("VM process (PID %d) not found.\n", pid)
		// Clean up stale PID file
		os.Remove(pidFile)
		return nil
	}

	// Try to signal the process to check if it's running
	if err := process.Signal(syscall.Signal(0)); err != nil {
		fmt.Printf("VM process (PID %d) is not running.\n", pid)
		// Clean up stale PID file
		os.Remove(pidFile)
		return nil
	}

	// Send SIGTERM for graceful shutdown
	fmt.Printf("Stopping VM (PID %d)...\n", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	fmt.Println("Stop signal sent.")
	fmt.Println("The VM should shut down gracefully.")

	// Clean up lock file if present
	lockFile := filepath.Join(baseDir, "data", vmName, ".running")
	os.Remove(lockFile)

	return nil
}
