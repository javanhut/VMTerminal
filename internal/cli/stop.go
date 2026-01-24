package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running VM",
	Long: `Stop the running VM gracefully.

If the VM is running in the foreground, use Ctrl+C instead.
This command attempts to stop a VM that may be running in another terminal.

Examples:
  vmt stop           # Graceful shutdown (SIGTERM)
  vmt stop --force   # Force kill (SIGKILL)
  vmt stop -f        # Force kill (short form)`,
	RunE: runStop,
}

var stopForce bool

func init() {
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force kill the VM (SIGKILL)")
}

func runStop(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	vmName := "default"
	dataDir := filepath.Join(baseDir, "data", vmName)

	// Check for PID file
	pidFile := filepath.Join(dataDir, "vm.pid")
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
		cleanupVMFiles(dataDir, pidFile)
		return nil
	}

	// Try to signal the process to check if it's running
	if err := process.Signal(syscall.Signal(0)); err != nil {
		fmt.Printf("VM process (PID %d) is not running (stale PID).\n", pid)
		cleanupVMFiles(dataDir, pidFile)
		fmt.Println("Cleaned up stale PID file.")
		return nil
	}

	if stopForce {
		// Force kill with SIGKILL
		fmt.Printf("Force killing VM (PID %d)...\n", pid)
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("send SIGKILL: %w", err)
		}
		// Wait a moment for process to die
		time.Sleep(500 * time.Millisecond)
		cleanupVMFiles(dataDir, pidFile)
		fmt.Println("VM force killed.")
	} else {
		// Send SIGTERM for graceful shutdown
		fmt.Printf("Stopping VM (PID %d)...\n", pid)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("send SIGTERM: %w", err)
		}
		fmt.Println("Stop signal sent.")
		fmt.Println("The VM should shut down gracefully.")
		fmt.Println("Use 'vmt stop --force' if it doesn't respond.")
	}

	return nil
}

// cleanupVMFiles removes PID file and lock file
func cleanupVMFiles(dataDir, pidFile string) {
	os.Remove(pidFile)
	lockFile := filepath.Join(dataDir, ".running")
	os.Remove(lockFile)
}
