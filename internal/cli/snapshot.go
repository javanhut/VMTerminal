package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Backup/restore VM state",
	Long:  `Create, list, restore, and delete VM disk snapshots.`,
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a snapshot",
	Long:  `Create a snapshot of the current VM disk state.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotCreate,
}

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List snapshots",
	Long:  `List all snapshots for the VM.`,
	RunE:  runSnapshotList,
}

var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore a snapshot",
	Long:  `Restore the VM disk from a snapshot. VM must be stopped.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotRestore,
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a snapshot",
	Long:  `Delete a snapshot and its associated files.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotDelete,
}

var snapshotShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show snapshot details",
	Long:  `Show detailed information about a snapshot.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotShow,
}

var snapshotDescription string

func init() {
	snapshotCreateCmd.Flags().StringVarP(&snapshotDescription, "description", "d", "", "Description for the snapshot")

	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	snapshotCmd.AddCommand(snapshotShowCmd)
}

// getSnapshotManager returns a SnapshotManager for the default VM.
func getSnapshotManager() (*vm.SnapshotManager, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")

	mgr := vm.NewSnapshotManager(baseDir)
	return mgr, "default", nil
}

// isSnapshotVMRunning checks if the VM appears to be running.
func isSnapshotVMRunning(baseDir, vmName string) bool {
	pidFile := filepath.Join(baseDir, "data", vmName, "vm.pid")
	if _, err := os.Stat(pidFile); err == nil {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			var pid int
			if _, err := fmt.Sscanf(string(data), "%d", &pid); err == nil {
				process, err := os.FindProcess(pid)
				if err == nil {
					if err := process.Signal(os.Signal(nil)); err == nil {
						return true
					}
				}
			}
		}
	}

	lockFile := filepath.Join(baseDir, "data", vmName, ".running")
	if _, err := os.Stat(lockFile); err == nil {
		return true
	}

	return false
}

func runSnapshotCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, vmName, err := getSnapshotManager()
	if err != nil {
		return err
	}

	fmt.Printf("Creating snapshot '%s'...\n", name)
	fmt.Println("This may take a while depending on disk size...")

	if err := mgr.CreateSnapshot(vmName, name, snapshotDescription); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	size, err := mgr.SnapshotFileSize(vmName, name)
	if err == nil {
		fmt.Printf("Snapshot created: %s (%.2f MB compressed)\n", name, float64(size)/(1024*1024))
	} else {
		fmt.Printf("Snapshot created: %s\n", name)
	}

	return nil
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	mgr, vmName, err := getSnapshotManager()
	if err != nil {
		return err
	}

	snapshots, err := mgr.ListSnapshots(vmName)
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found. Create one with: vmterminal snapshot create <name>")
		return nil
	}

	fmt.Println("Snapshots:")
	for _, snap := range snapshots {
		size, _ := mgr.SnapshotFileSize(vmName, snap.Name)
		fmt.Printf("  %s\n", snap.Name)
		fmt.Printf("    Created: %s\n", snap.CreatedAt.Format("2006-01-02 15:04:05"))
		if snap.Description != "" {
			fmt.Printf("    Description: %s\n", snap.Description)
		}
		fmt.Printf("    Original size: %.2f MB\n", float64(snap.DiskSize)/(1024*1024))
		if size > 0 {
			fmt.Printf("    Compressed size: %.2f MB\n", float64(size)/(1024*1024))
		}
	}

	return nil
}

func runSnapshotRestore(cmd *cobra.Command, args []string) error {
	name := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")

	mgr, vmName, err := getSnapshotManager()
	if err != nil {
		return err
	}

	// Check if VM is running
	if isSnapshotVMRunning(baseDir, vmName) {
		fmt.Println("Error: VM appears to be running.")
		fmt.Println("Please stop the VM before restoring a snapshot:")
		fmt.Println("  Press Ctrl+C in the VM terminal, or")
		fmt.Println("  Run 'vmterminal stop'")
		return fmt.Errorf("VM is running")
	}

	// Get snapshot info for confirmation
	snap, err := mgr.GetSnapshot(vmName, name)
	if err != nil {
		return fmt.Errorf("get snapshot: %w", err)
	}

	fmt.Printf("Restoring from snapshot '%s'...\n", name)
	fmt.Printf("  Created: %s\n", snap.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("WARNING: This will overwrite the current disk!")
	fmt.Println("Restoring...")

	if err := mgr.RestoreSnapshot(vmName, name); err != nil {
		return fmt.Errorf("restore snapshot: %w", err)
	}

	fmt.Printf("Snapshot '%s' restored successfully.\n", name)
	fmt.Println("You can now start the VM with: vmterminal run")

	return nil
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, vmName, err := getSnapshotManager()
	if err != nil {
		return err
	}

	if err := mgr.DeleteSnapshot(vmName, name); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}

	fmt.Printf("Snapshot '%s' deleted.\n", name)

	return nil
}

func runSnapshotShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, vmName, err := getSnapshotManager()
	if err != nil {
		return err
	}

	snap, err := mgr.GetSnapshot(vmName, name)
	if err != nil {
		return fmt.Errorf("get snapshot: %w", err)
	}

	size, _ := mgr.SnapshotFileSize(vmName, name)

	fmt.Printf("Snapshot: %s\n", snap.Name)
	fmt.Printf("  Created: %s\n", snap.CreatedAt.Format("2006-01-02 15:04:05"))
	if snap.Description != "" {
		fmt.Printf("  Description: %s\n", snap.Description)
	}
	fmt.Printf("  Original disk size: %.2f MB\n", float64(snap.DiskSize)/(1024*1024))
	if size > 0 {
		fmt.Printf("  Compressed size: %.2f MB\n", float64(size)/(1024*1024))
		ratio := float64(size) / float64(snap.DiskSize) * 100
		fmt.Printf("  Compression ratio: %.1f%%\n", ratio)
	}

	return nil
}
