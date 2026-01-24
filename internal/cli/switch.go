package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Interactive distro switching",
	Long: `Switch to a different Linux distribution.

This command shows available distributions and lets you select a new one.
The current VM data will be preserved, and a new VM will be set up for
the selected distribution.`,
	RunE: runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) error {
	// Load current config
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}

	// Show current distro
	currentDistro := cfg.Distro
	if currentDistro == "" {
		currentDistro = "alpine"
	}

	currentProvider, _ := distro.Get(distro.ID(currentDistro))
	if currentProvider != nil {
		fmt.Printf("Current distro: %s %s\n", currentProvider.Name(), currentProvider.Version())
	} else {
		fmt.Printf("Current distro: %s\n", currentDistro)
	}
	fmt.Println()

	// List available distros
	fmt.Println("Available Linux distributions:")
	providers := distro.ListProviders()
	for i, p := range providers {
		marker := ""
		if string(p.ID()) == currentDistro {
			marker = " (current)"
		}
		fmt.Printf("  %d. %s %s%s\n", i+1, p.Name(), p.Version(), marker)
	}

	fmt.Print("\nSelect distro (or 'q' to quit): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "q" || input == "quit" || input == "" {
		fmt.Println("No changes made.")
		return nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(providers) {
		fmt.Println("Invalid selection.")
		return nil
	}

	selectedProvider := providers[choice-1]
	selectedID := string(selectedProvider.ID())

	// Check if same as current
	if selectedID == currentDistro {
		fmt.Println("Already using this distribution.")
		return nil
	}

	fmt.Printf("\nSwitching to %s %s...\n", selectedProvider.Name(), selectedProvider.Version())

	// Setup paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	cacheDir := filepath.Join(baseDir, "cache")
	dataDir := filepath.Join(baseDir, "data", "default")

	// Check if current VM is running
	if isVMRunningCheck(baseDir, "default") {
		fmt.Println("\nWarning: The current VM appears to be running.")
		fmt.Println("Please stop it first with 'vmterminal stop' or Ctrl+C.")
		return fmt.Errorf("VM is running")
	}

	// Ask about preserving data
	fmt.Println("\nNote: Switching distros will create a new disk image.")
	fmt.Println("Your current VM's disk will be preserved as a backup.")

	if !promptYesNoSwitch("Continue?", true) {
		fmt.Println("Cancelled.")
		return nil
	}

	// Backup current disk if exists
	currentDisk := filepath.Join(dataDir, "disk.img")
	if _, err := os.Stat(currentDisk); err == nil {
		backupDisk := filepath.Join(dataDir, fmt.Sprintf("disk-%s.img.bak", currentDistro))
		fmt.Printf("Backing up current disk to %s...\n", filepath.Base(backupDisk))
		if err := os.Rename(currentDisk, backupDisk); err != nil {
			fmt.Printf("Warning: Could not backup disk: %v\n", err)
		}
	}

	// Download new distro assets
	fmt.Printf("Downloading %s...\n", selectedProvider.Name())
	assets := vm.NewAssetManager(cacheDir, selectedProvider)
	assetPaths, err := assets.EnsureAssets()
	if err != nil {
		return fmt.Errorf("get asset paths: %w", err)
	}

	// Create new disk image
	fmt.Println("Creating new disk image...")
	images := vm.NewImageManager(dataDir)
	if _, err := images.EnsureDisk("disk", int64(cfg.DiskSizeMB)); err != nil {
		return fmt.Errorf("create disk: %w", err)
	}

	// Setup filesystem
	fmt.Println("\nDisk formatting requires sudo permissions.")
	if !promptYesNoSwitch("Give sudo permission to set up the disk?", true) {
		return fmt.Errorf("sudo permission required for setup")
	}

	rootfs := vm.NewRootfsManager(dataDir)
	reqs := selectedProvider.SetupRequirements()
	fsType := reqs.FSType
	if fsType == "" {
		fsType = "ext4"
	}

	fmt.Printf("Formatting disk with %s filesystem...\n", fsType)
	if err := rootfs.FormatDisk("disk", fsType); err != nil {
		return fmt.Errorf("format disk: %w", err)
	}

	fmt.Println("Extracting rootfs to disk...")
	if err := rootfs.ExtractRootfs("disk", assetPaths.Rootfs); err != nil {
		return fmt.Errorf("extract rootfs: %w", err)
	}

	// Update config
	cfg.Distro = selectedID
	if err := config.SaveState(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\nSwitched to %s %s.\n", selectedProvider.Name(), selectedProvider.Version())
	fmt.Println("Run 'vmterminal run' to start the new VM.")

	return nil
}

// isVMRunningCheck checks if a VM appears to be running.
func isVMRunningCheck(baseDir, vmName string) bool {
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

// promptYesNoSwitch asks a yes/no question with a default value.
func promptYesNoSwitch(question string, defaultYes bool) bool {
	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}
