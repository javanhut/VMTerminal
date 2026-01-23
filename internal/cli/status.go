package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/javanstorm/vmterminal/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VM status and information",
	Long:  `Display information about the VM including hypervisor, distro, boot history, and disk status.`,
	RunE:  runStatus,
}

var statusVMName string

func init() {
	statusCmd.Flags().StringVar(&statusVMName, "vm", "", "VM to show status for (default: active VM)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load config to get distro setting
	if err := config.Load(); err != nil {
		// Continue even if config fails - we can still show some info
		fmt.Printf("Warning: failed to load config: %v\n\n", err)
	}
	cfg := config.Global
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Get paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	cacheDir := filepath.Join(baseDir, "cache")

	// Get VM from registry
	registry := vm.NewRegistry(baseDir)
	var vmEntry *vm.VMEntry

	if statusVMName != "" {
		// Use specified VM
		vmEntry, err = registry.GetVM(statusVMName)
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

	// Show VM info
	active, _ := registry.GetActive()
	activeMarker := ""
	if vmEntry.Name == active {
		activeMarker = " (active)"
	}
	fmt.Printf("VM: %s%s\n", vmEntry.Name, activeMarker)
	fmt.Println()

	// Get hypervisor info
	driver, err := hypervisor.NewDriver()
	if err != nil {
		fmt.Printf("Hypervisor: unavailable (%v)\n", err)
	} else {
		info := driver.Info()
		fmt.Printf("Hypervisor: %s v%s (%s)\n", info.Name, info.Version, info.Arch)
	}

	fmt.Println()

	// Get and display distro info
	distroID := distro.ID(vmEntry.Distro)
	if distroID == "" {
		distroID = distro.DefaultID()
	}

	provider, err := distro.Get(distroID)
	if err != nil {
		fmt.Printf("Distro: unknown (%v)\n", err)
	} else {
		fmt.Printf("Distro: %s %s\n", provider.Name(), provider.Version())
		fmt.Printf("  Supported architectures: %v\n", provider.SupportedArchs())
	}

	// List available distros
	fmt.Printf("  Available distros: %v\n", distro.List())

	fmt.Println()

	// Check assets
	if provider != nil {
		assets := vm.NewAssetManager(cacheDir, provider)
		assetPaths, err := assets.GetAssetPaths()
		if err != nil {
			fmt.Printf("Assets: error checking (%v)\n", err)
		} else {
			hasKernel := assetPaths.Kernel != ""
			hasInitramfs := assetPaths.Initramfs != ""
			hasRootfs := assetPaths.Rootfs != ""

			if !hasKernel && !hasInitramfs && !hasRootfs {
				fmt.Printf("Assets: not downloaded\n")
			} else {
				fmt.Printf("Assets:\n")
				if hasKernel {
					fmt.Printf("  Kernel: %s\n", assetPaths.Kernel)
				} else {
					fmt.Printf("  Kernel: not downloaded\n")
				}
				if hasInitramfs {
					fmt.Printf("  Initramfs: %s\n", assetPaths.Initramfs)
				} else {
					fmt.Printf("  Initramfs: not downloaded\n")
				}
				if hasRootfs {
					fmt.Printf("  Rootfs: %s\n", assetPaths.Rootfs)
				} else {
					fmt.Printf("  Rootfs: not downloaded\n")
				}
			}
		}
	}

	fmt.Println()

	// Check disk and setup state
	images := vm.NewImageManager(dataDir)
	rootfs := vm.NewRootfsManager(dataDir)

	if images.DiskExists("disk") {
		diskPath := images.DiskPath("disk")
		info, err := os.Stat(diskPath)
		if err == nil {
			fmt.Printf("Disk:\n")
			fmt.Printf("  Path: %s\n", diskPath)
			fmt.Printf("  Size: %.2f MB (allocated)\n", float64(info.Size())/(1024*1024))

			// Check setup state
			state, err := rootfs.CheckSetupState("disk")
			if err != nil {
				fmt.Printf("  Setup: error checking (%v)\n", err)
			} else if state.RootfsExtracted {
				fmt.Printf("  Setup: complete (%s filesystem)\n", state.FSType)
			} else if state.DiskFormatted {
				fmt.Printf("  Setup: formatted but rootfs not extracted\n")
			} else {
				fmt.Printf("  Setup: not done (run: sudo vmterminal setup)\n")
			}
		}
	} else {
		fmt.Printf("Disk: not created\n")
	}

	fmt.Println()

	// Check state
	stateFile := vm.NewStateFile(dataDir)
	state, err := stateFile.Load()
	if err != nil {
		fmt.Printf("State: error loading (%v)\n", err)
	} else if state.BootCount == 0 {
		fmt.Printf("State: never booted\n")
	} else {
		fmt.Printf("State:\n")
		fmt.Printf("  Boot count: %d\n", state.BootCount)
		if !state.LastBoot.IsZero() {
			fmt.Printf("  Last boot: %s\n", state.LastBoot.Format("2006-01-02 15:04:05"))
		}
		if !state.LastShutdown.IsZero() {
			fmt.Printf("  Last shutdown: %s\n", state.LastShutdown.Format("2006-01-02 15:04:05"))
			if state.CleanShutdown {
				fmt.Printf("  Shutdown type: clean\n")
			} else {
				fmt.Printf("  Shutdown type: unclean\n")
			}
		}
	}

	return nil
}
