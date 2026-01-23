package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start and run the VM",
	Long:  `Start the Linux VM, downloading assets if needed. The VM runs in the foreground until stopped.`,
	RunE:  runRun,
}

var runDistro string
var runVMName string

func init() {
	runCmd.Flags().IntP("cpus", "c", 0, "Number of virtual CPUs (default from VM config)")
	runCmd.Flags().IntP("memory", "m", 0, "Memory in MB (default from VM config)")
	runCmd.Flags().StringVarP(&runDistro, "distro", "d", "", "Linux distribution to use (default from VM config)")
	runCmd.Flags().StringVar(&runVMName, "vm", "", "VM to run (default: active VM)")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Config is already loaded by PersistentPreRunE in root.go
	cfg := config.Global
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Setup paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")

	// Get VM from registry
	registry := vm.NewRegistry(baseDir)
	var vmEntry *vm.VMEntry

	if runVMName != "" {
		// Use specified VM
		vmEntry, err = registry.GetVM(runVMName)
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

	// Use VM config as defaults, override from flags if specified
	cpus := vmEntry.CPUs
	memory := vmEntry.MemoryMB
	diskSizeMB := vmEntry.DiskSizeMB
	if flagCPUs, _ := cmd.Flags().GetInt("cpus"); flagCPUs > 0 {
		cpus = flagCPUs
	}
	if flagMemory, _ := cmd.Flags().GetInt("memory"); flagMemory > 0 {
		memory = flagMemory
	}

	// Resolve distro
	distroID := distro.ID(vmEntry.Distro)
	if runDistro != "" {
		distroID = distro.ID(runDistro)
	}
	if distroID == "" {
		distroID = distro.DefaultID()
	}

	provider, err := distro.Get(distroID)
	if err != nil {
		return fmt.Errorf("get distro: %w", err)
	}

	// Build shared dirs map from config
	sharedDirs := make(map[string]string)
	for i, dir := range cfg.SharedDirs {
		tag := fmt.Sprintf("share%d", i)
		sharedDirs[tag] = dir
	}

	managerCfg := vm.ManagerConfig{
		CacheDir:      filepath.Join(baseDir, "cache"),
		DataDir:       registry.VMDataDir(vmEntry.Name),
		CPUs:          cpus,
		MemoryMB:      memory,
		DiskSizeMB:    int64(diskSizeMB),
		DiskName:      "disk",
		SharedDirs:    sharedDirs,
		EnableNetwork: cfg.EnableNetwork,
		MACAddress:    cfg.MACAddress,
		SSHHostPort:   cfg.SSHHostPort,
		Provider:      provider,
	}

	mgr, err := vm.NewManager(managerCfg)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	fmt.Printf("Hypervisor: %s (%s)\n", mgr.DriverInfo().Name, mgr.DriverInfo().Arch)
	fmt.Printf("Distro: %s %s\n", provider.Name(), provider.Version())

	// Show shared directories
	if len(sharedDirs) > 0 {
		fmt.Println("Shared directories (mount in VM with: mount -t virtiofs <tag> <mountpoint>):")
		for tag, path := range sharedDirs {
			fmt.Printf("  %s -> %s\n", tag, path)
		}
	}

	// Check if setup is needed
	rootfs := vm.NewRootfsManager(managerCfg.DataDir)
	state, err := rootfs.CheckSetupState("disk")
	if err != nil {
		return fmt.Errorf("check setup state: %w", err)
	}

	if !state.RootfsExtracted {
		fmt.Println()
		fmt.Printf("VM '%s' disk is not set up. Please run setup first:\n", vmEntry.Name)
		if runVMName != "" {
			fmt.Printf("  sudo vmterminal setup --vm %s\n", vmEntry.Name)
		} else {
			fmt.Println("  sudo vmterminal setup")
		}
		fmt.Println()
		fmt.Println("This will format the disk and extract the rootfs.")
		return fmt.Errorf("setup required")
	}

	fmt.Println("Preparing VM (downloading assets if needed)...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare VM: %w", err)
	}

	fmt.Println("Starting VM...")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}

	// Get console I/O handles
	vmIn, vmOut, err := mgr.Console()
	if err != nil {
		return fmt.Errorf("get console: %w", err)
	}

	fmt.Println("Attaching to console (Ctrl+C to stop VM)...")

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Attach in background, stop on signal
	attachDone := make(chan error, 1)
	go func() {
		attachDone <- AttachToConsole(ctx, vmIn, vmOut)
	}()

	// Wait for either signal or attach completion
	select {
	case <-sigCh:
		fmt.Println("\nStopping VM...")
		cancel() // Cancel context to stop attach
		if err := mgr.Stop(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Stop error: %v\n", err)
		}
	case err := <-attachDone:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Attach error: %v\n", err)
		}
	}

	// Wait for VM to exit
	if err := mgr.Wait(); err != nil {
		return fmt.Errorf("VM exited with error: %w", err)
	}

	fmt.Println("VM stopped")
	return nil
}
