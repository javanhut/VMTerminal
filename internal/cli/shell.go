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

// IsLoginShell returns true if invoked as a login shell.
// Login shells are invoked with argv[0] prefixed with '-' by convention.
func IsLoginShell() bool {
	if len(os.Args) == 0 {
		return false
	}
	arg0 := os.Args[0]
	if len(arg0) == 0 {
		return false
	}
	return arg0[0] == '-'
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start VM and attach to shell",
	Long:  `Start the Linux VM and attach to its console as a shell session.`,
	RunE:  runShell,
}

var shellVMName string

func init() {
	shellCmd.Flags().StringVar(&shellVMName, "vm", "", "VM to use (default: active VM)")
}

func runShell(cmd *cobra.Command, args []string) error {
	// Load config
	cfg := config.Global
	if cfg == nil {
		// Try loading config if not already loaded (for direct shell invocation)
		if err := config.Load(); err != nil {
			// Use defaults if config fails to load
			cfg = config.DefaultConfig()
		} else {
			cfg = config.Global
		}
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

	if shellVMName != "" {
		// Use specified VM
		vmEntry, err = registry.GetVM(shellVMName)
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

	// Use VM config as defaults
	cpus := vmEntry.CPUs
	memory := vmEntry.MemoryMB
	diskSizeMB := vmEntry.DiskSizeMB

	// Resolve distro
	distroID := distro.ID(vmEntry.Distro)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare VM: %w", err)
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}

	// Get console I/O handles
	vmIn, vmOut, err := mgr.Console()
	if err != nil {
		return fmt.Errorf("get console: %w", err)
	}

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Attach in background
	attachDone := make(chan error, 1)
	go func() {
		attachDone <- AttachToConsole(ctx, vmIn, vmOut)
	}()

	// Wait for either signal or attach completion
	select {
	case <-sigCh:
		cancel() // Cancel context to stop attach
		if err := mgr.Stop(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Stop error: %v\n", err)
		}
	case err := <-attachDone:
		if err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "Attach error: %v\n", err)
		}
		// Console closed, stop VM
		if err := mgr.Stop(ctx); err != nil {
			// Ignore stop errors on clean exit
		}
	}

	// Wait for VM to exit
	mgr.Wait()

	return nil
}

// RunShellMode runs vmterminal as a login shell.
// This is called from main.go when detected as login shell.
func RunShellMode() {
	if err := runShell(nil, nil); err != nil {
		fmt.Fprintf(os.Stderr, "vmterminal: %v\n", err)
		os.Exit(1)
	}
}
