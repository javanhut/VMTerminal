package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the VM disk with rootfs",
	Long: `Downloads distro assets and sets up the VM disk.

This command:
1. Downloads kernel, initramfs, and rootfs for the selected distro
2. Creates a disk image if it doesn't exist
3. Formats the disk with the appropriate filesystem
4. Extracts the rootfs to the disk

The format and extract steps require root privileges (sudo).
After setup, you can run the VM without sudo.`,
	RunE: runSetup,
}

var (
	setupDistro string
	setupForce  bool
	setupVMName string
)

func init() {
	setupCmd.Flags().StringVarP(&setupDistro, "distro", "d", "", "Linux distribution to set up (default from VM config)")
	setupCmd.Flags().BoolVarP(&setupForce, "force", "f", false, "Force setup even if already done")
	setupCmd.Flags().StringVar(&setupVMName, "vm", "", "VM to set up (default: active VM)")
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Load config
	if err := config.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg := config.Global

	// Setup paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	cacheDir := filepath.Join(baseDir, "cache")

	// Get VM from registry
	registry := vm.NewRegistry(baseDir)
	var vmEntry *vm.VMEntry

	if setupVMName != "" {
		// Use specified VM
		vmEntry, err = registry.GetVM(setupVMName)
		if err != nil {
			return fmt.Errorf("get VM: %w", err)
		}
	} else {
		// Use active VM (create default if none exist)
		vmEntry, err = registry.GetActiveOrDefault(cfg.Distro, cfg.CPUs, cfg.MemoryMB, cfg.DiskSizeMB)
		if err != nil {
			return fmt.Errorf("get active VM: %w", err)
		}
	}

	dataDir := registry.VMDataDir(vmEntry.Name)

	// Resolve distro: flag > VM config > default
	distroID := distro.ID(vmEntry.Distro)
	if setupDistro != "" {
		distroID = distro.ID(setupDistro)
	}
	if distroID == "" {
		distroID = distro.DefaultID()
	}

	provider, err := distro.Get(distroID)
	if err != nil {
		return fmt.Errorf("get distro: %w", err)
	}

	fmt.Printf("Setting up VM '%s' with %s %s\n", vmEntry.Name, provider.Name(), provider.Version())

	// Create directories
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Create managers
	assets := vm.NewAssetManager(cacheDir, provider)
	images := vm.NewImageManager(dataDir)
	rootfs := vm.NewRootfsManager(dataDir)

	// Check current setup state
	state, err := rootfs.CheckSetupState("disk")
	if err != nil {
		return fmt.Errorf("check setup state: %w", err)
	}

	if state.RootfsExtracted && !setupForce {
		fmt.Println("Disk is already set up. Use --force to redo setup.")
		return nil
	}

	// Download assets
	fmt.Println("Downloading assets...")
	assetPaths, err := assets.EnsureAssets()
	if err != nil {
		return fmt.Errorf("ensure assets: %w", err)
	}

	if assetPaths.Kernel != "" {
		fmt.Printf("  Kernel: %s\n", assetPaths.Kernel)
	}
	if assetPaths.Initramfs != "" {
		fmt.Printf("  Initramfs: %s\n", assetPaths.Initramfs)
	}
	if assetPaths.Rootfs != "" {
		fmt.Printf("  Rootfs: %s\n", assetPaths.Rootfs)
	}

	// Create disk image if needed
	diskSizeMB := int64(vmEntry.DiskSizeMB)
	if diskSizeMB == 0 {
		diskSizeMB = vm.DefaultDiskSizeMB
	}

	fmt.Printf("Creating disk image (%d MB)...\n", diskSizeMB)
	diskPath, err := images.EnsureDisk("disk", diskSizeMB)
	if err != nil {
		return fmt.Errorf("create disk: %w", err)
	}
	fmt.Printf("  Disk: %s\n", diskPath)

	// Get setup requirements from provider
	setupReqs := provider.SetupRequirements()

	// Format disk
	if setupReqs.NeedsFormatting {
		fmt.Printf("Formatting disk with %s...\n", setupReqs.FSType)
		if err := rootfs.FormatDisk("disk", setupReqs.FSType); err != nil {
			return fmt.Errorf("format disk: %w", err)
		}
	}

	// Extract rootfs
	if setupReqs.NeedsExtraction && assetPaths.Rootfs != "" {
		fmt.Println("Extracting rootfs (this may take a while)...")
		if err := rootfs.ExtractRootfs("disk", assetPaths.Rootfs); err != nil {
			return fmt.Errorf("extract rootfs: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("Setup complete! You can now run the VM with:")
	if setupVMName != "" {
		fmt.Printf("  vmterminal run --vm %s\n", vmEntry.Name)
	} else {
		fmt.Println("  vmterminal run")
	}

	return nil
}
