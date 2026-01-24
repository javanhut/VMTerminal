package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/javanstorm/vmterminal/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current VM state, distro, config",
	Long:  `Display information about the VM including distro, running state, architecture, host OS, and configuration.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}

	// Get paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	dataDir := filepath.Join(baseDir, "data", "default")
	cacheDir := filepath.Join(baseDir, "cache")

	fmt.Println("VMTerminal Status")
	fmt.Println("=================")
	fmt.Println()

	// System Information
	fmt.Println("System:")
	fmt.Printf("  Architecture: %s\n", formatArch(runtime.GOARCH))
	fmt.Printf("  Host OS: %s\n", formatOS(runtime.GOOS))

	// Get hypervisor info
	driver, err := hypervisor.NewDriver()
	if err != nil {
		fmt.Printf("  Hypervisor: unavailable (%v)\n", err)
	} else {
		info := driver.Info()
		fmt.Printf("  Hypervisor: %s v%s (%s)\n", info.Name, info.Version, info.Arch)
	}
	fmt.Println()

	// Distro Information
	fmt.Println("Distro:")
	distroID := distro.ID(cfg.Distro)
	if distroID == "" {
		distroID = distro.DefaultID()
	}

	provider, err := distro.Get(distroID)
	if err != nil {
		fmt.Printf("  Current: %s (unknown)\n", cfg.Distro)
	} else {
		fmt.Printf("  Current: %s %s\n", provider.Name(), provider.Version())
		fmt.Printf("  Supported architectures: %v\n", provider.SupportedArchs())
	}
	fmt.Printf("  Available: %v\n", distro.List())
	fmt.Println()

	// VM State
	fmt.Println("VM State:")
	isRunning := isVMRunningCheck(baseDir, "default")
	if isRunning {
		fmt.Println("  Status: RUNNING")
	} else {
		fmt.Println("  Status: stopped")
	}

	// Check disk and setup state
	images := vm.NewImageManager(dataDir)
	rootfs := vm.NewRootfsManager(dataDir)

	if images.DiskExists("disk") {
		diskPath := images.DiskPath("disk")
		info, err := os.Stat(diskPath)
		if err == nil {
			fmt.Printf("  Disk: %.2f MB (allocated)\n", float64(info.Size())/(1024*1024))

			state, err := rootfs.CheckSetupState("disk")
			if err != nil {
				fmt.Printf("  Setup: error checking (%v)\n", err)
			} else if state.RootfsExtracted {
				fmt.Printf("  Setup: complete (%s)\n", state.FSType)
			} else if state.DiskFormatted {
				fmt.Println("  Setup: formatted, rootfs not extracted")
			} else {
				fmt.Println("  Setup: not done")
			}
		}
	} else {
		fmt.Println("  Disk: not created")
		fmt.Println("  Setup: not done (run 'vmterminal run' to set up)")
	}

	// Boot history
	stateFile := vm.NewStateFile(dataDir)
	vmState, err := stateFile.Load()
	if err == nil && vmState.BootCount > 0 {
		fmt.Printf("  Boot count: %d\n", vmState.BootCount)
		if !vmState.LastBoot.IsZero() {
			fmt.Printf("  Last boot: %s\n", vmState.LastBoot.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Println()

	// Configuration
	fmt.Println("Configuration:")
	fmt.Printf("  CPUs: %d\n", cfg.CPUs)
	fmt.Printf("  Memory: %d MB\n", cfg.MemoryMB)
	fmt.Printf("  Disk Size: %d MB\n", cfg.DiskSizeMB)
	fmt.Printf("  Network: %s\n", formatEnabled(cfg.EnableNetwork))
	fmt.Printf("  SSH Port: %d\n", cfg.SSHHostPort)
	if len(cfg.SharedDirs) > 0 {
		fmt.Printf("  Shared Dirs: %s\n", cfg.SharedDirs[0])
		for _, dir := range cfg.SharedDirs[1:] {
			fmt.Printf("               %s\n", dir)
		}
	} else {
		fmt.Println("  Shared Dirs: (none)")
	}
	fmt.Printf("  Default Terminal: %s\n", formatEnabled(cfg.IsDefaultTerminal))
	fmt.Println()

	// Assets
	if provider != nil {
		fmt.Println("Assets:")
		assets := vm.NewAssetManager(cacheDir, provider)
		assetPaths, err := assets.GetAssetPaths()
		if err != nil {
			fmt.Println("  Status: not downloaded")
		} else {
			hasKernel := assetPaths.Kernel != ""
			hasInitramfs := assetPaths.Initramfs != ""
			hasRootfs := assetPaths.Rootfs != ""

			if hasKernel && hasInitramfs && hasRootfs {
				fmt.Println("  Status: downloaded")
				fmt.Printf("  Kernel: %s\n", filepath.Base(assetPaths.Kernel))
				fmt.Printf("  Initramfs: %s\n", filepath.Base(assetPaths.Initramfs))
				fmt.Printf("  Rootfs: %s\n", filepath.Base(assetPaths.Rootfs))
			} else {
				fmt.Println("  Status: partially downloaded")
			}
		}
	}

	return nil
}

// formatArch returns a human-readable architecture name.
func formatArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "Arm64"
	default:
		return arch
	}
}

// formatOS returns a human-readable OS name.
func formatOS(os string) string {
	switch os {
	case "darwin":
		return "Darwin (macOS)"
	case "linux":
		return "Linux"
	default:
		return os
	}
}

// formatEnabled returns "enabled" or "disabled".
func formatEnabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}
