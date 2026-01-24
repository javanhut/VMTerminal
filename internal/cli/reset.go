package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset [distro]",
	Short: "Reset VM state for fresh start",
	Long: `Reset VM to a clean state. Kills any running VM, clears cache, and removes disk.

This is useful when:
  - Testing changes to VMTerminal
  - Switching distros cleanly
  - Recovering from a broken state

Examples:
  vmt reset          # Reset current distro
  vmt reset ubuntu   # Reset and prepare for Ubuntu
  vmt reset --all    # Reset everything (all distros)`,
	RunE: runReset,
}

var resetAll bool

func init() {
	resetCmd.Flags().BoolVar(&resetAll, "all", false, "Reset all distros and data")
}

func runReset(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	dataDir := filepath.Join(baseDir, "data", "default")
	cacheDir := filepath.Join(baseDir, "cache")

	// Step 1: Kill any running VM
	pidFile := filepath.Join(dataDir, "vm.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(syscall.Signal(0)); err == nil {
					fmt.Printf("Killing VM (PID %d)...\n", pid)
					process.Signal(syscall.SIGKILL)
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
		os.Remove(pidFile)
	}
	// Clean up lock file
	os.Remove(filepath.Join(dataDir, ".running"))

	// Step 2: Clear cache
	if resetAll {
		// Clear all cache
		if _, err := os.Stat(cacheDir); err == nil {
			if err := os.RemoveAll(cacheDir); err != nil {
				return fmt.Errorf("clear cache: %w", err)
			}
			fmt.Println("Cleared all cached assets")
		}
	} else if len(args) > 0 {
		// Clear specific distro cache
		distroID := args[0]
		if !distro.IsRegistered(distro.ID(distroID)) {
			return fmt.Errorf("unknown distro: %s", distroID)
		}
		targetDir := filepath.Join(cacheDir, distroID)
		if _, err := os.Stat(targetDir); err == nil {
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("clear %s cache: %w", distroID, err)
			}
			fmt.Printf("Cleared cache for %s\n", distroID)
		}
	}

	// Step 3: Remove disk image
	diskPath := filepath.Join(dataDir, "disk.raw")
	if _, err := os.Stat(diskPath); err == nil {
		if err := os.Remove(diskPath); err != nil {
			return fmt.Errorf("remove disk: %w", err)
		}
		fmt.Println("Removed disk image")
	}

	// Step 4: Reset state file
	statePath := filepath.Join(dataDir, "state.json")
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state: %w", err)
	}

	fmt.Println("\nReset complete. Run 'vmt run' to start fresh.")
	if len(args) > 0 {
		fmt.Printf("Tip: vmt run --distro %s\n", args[0])
	}

	return nil
}
