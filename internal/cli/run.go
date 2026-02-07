package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/gui"
	"github.com/javanstorm/vmterminal/internal/timing"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/javanstorm/vmterminal/pkg/hypervisor"
	"github.com/spf13/cobra"
)

// Warm startup timing targets (VMT_TIMING=1):
// When assets are cached and disk exists (warm path):
//   - config_load:     <50ms   (load JSON config from disk)
//   - distro_resolve:  <10ms   (registry lookup)
//   - manager_create:  <100ms  (driver init, apply defaults)
//   - vm_prepare:      <200ms  (parallel asset checks, skip downloads)
//   - vm_start:        <2000ms (hypervisor-dependent, kernel boot)
//   - gui_launch:      <100ms  (open GUI terminal window)
//   - TOTAL:           <3000ms
//
// Cold path (first run) adds: asset downloads, disk creation, rootfs extraction.
// Run with VMT_TIMING=1 to see actual breakdown.

// quietMode suppresses verbose output when running as login shell.
var quietMode bool

// SetQuietMode enables or disables quiet mode (minimal output).
func SetQuietMode(quiet bool) {
	quietMode = quiet
}

// printIfNotQuiet prints only when not in quiet mode.
func printIfNotQuiet(format string, args ...interface{}) {
	if !quietMode {
		fmt.Printf(format, args...)
	}
}

// printlnIfNotQuiet prints a line only when not in quiet mode.
func printlnIfNotQuiet(args ...interface{}) {
	if !quietMode {
		fmt.Println(args...)
	}
}

// isVMRunning checks if a VM is already running.
func isVMRunning(baseDir, vmName string) (bool, int) {
	pidFile := filepath.Join(baseDir, "data", vmName, "vm.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return false, 0
	}
	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, 0
	}
	return true, pid
}

// writePIDFile creates a PID file for the current process.
func writePIDFile(baseDir, vmName string) error {
	pidFile := filepath.Join(baseDir, "data", vmName, "vm.pid")
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

// cleanupPIDFile removes the PID file.
func cleanupPIDFile(baseDir, vmName string) {
	pidFile := filepath.Join(baseDir, "data", vmName, "vm.pid")
	os.Remove(pidFile)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start VM (handles all setup interactively if needed)",
	Long: `Start the Linux VM. If the VM is not set up, this command will
interactively guide you through the setup process:

1. Detect system architecture and OS
2. Check for optional dependencies (FuseFS)
3. Download the Linux distribution if needed
4. Set up filesystem (may require sudo)
5. Start VM and open GUI terminal window`,
	RunE: runRun,
}

var runDistro string

func init() {
	runCmd.Flags().StringVarP(&runDistro, "distro", "d", "", "Linux distribution to use")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Initialize timing if VMT_TIMING=1
	var timer *timing.Timer
	if os.Getenv("VMT_TIMING") == "1" {
		timer = timing.New()
	}

	// Load or create config
	cfg, err := config.LoadState()
	if err != nil {
		// First run - create default config
		cfg = config.DefaultState()
	}
	if timer != nil {
		timer.Mark("config_load")
	}

	// Override distro from flag if specified
	if runDistro != "" {
		cfg.Distro = runDistro
	}

	// Print system information (skip in quiet mode)
	if !quietMode {
		printSystemInfo()
	}

	// Setup paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("create base dir: %w", err)
	}

	// Check if VM is already running
	running, pid := isVMRunning(baseDir, "default")
	if running {
		fmt.Printf("VM is already running (PID %d).\n", pid)
		fmt.Println("You can:")
		fmt.Println("  - Run 'vmterminal stop' to stop the VM")
		fmt.Println("  - Run 'vmterminal status' to see VM state")
		return nil
	}

	// Get or prompt for distro
	distroID, err := resolveDistro(cfg)
	if err != nil {
		return err
	}
	cfg.Distro = string(distroID)

	provider, err := distro.Get(distroID)
	if err != nil {
		return fmt.Errorf("get distro: %w", err)
	}
	if timer != nil {
		timer.Mark("distro_resolve")
	}

	// Setup data directory for VM
	dataDir := filepath.Join(baseDir, "data", "default")
	cacheDir := filepath.Join(baseDir, "cache")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Check setup state
	rootfs := vm.NewRootfsManager(dataDir)
	state, err := rootfs.CheckSetupState("disk")
	if err != nil {
		// Disk might not exist yet, that's OK
		state = &vm.SetupState{}
	}

	// If not set up, run interactive setup
	if !state.RootfsExtracted {
		fmt.Println()
		if err := interactiveSetup(cfg, provider, dataDir, cacheDir); err != nil {
			return err
		}
	}

	// Early capability check - warn about unsupported features
	driver, err := hypervisor.NewDriver()
	if err != nil {
		return fmt.Errorf("create driver: %w", err)
	}
	caps := driver.Capabilities()

	warnings := config.ValidateConfig(cfg, caps)
	if len(warnings) > 0 {
		fmt.Fprint(os.Stderr, config.FormatValidationErrors(warnings))
		// Continue anyway - warnings are informational
	}

	// Build shared dirs map from config
	sharedDirs := make(map[string]string)
	for i, dir := range cfg.SharedDirs {
		tag := fmt.Sprintf("share%d", i)
		sharedDirs[tag] = dir
	}

	// Create VM manager
	managerCfg := vm.ManagerConfig{
		CacheDir:      cacheDir,
		DataDir:       dataDir,
		CPUs:          cfg.CPUs,
		MemoryMB:      cfg.MemoryMB,
		DiskSizeMB:    int64(cfg.DiskSizeMB),
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
	if timer != nil {
		timer.Mark("manager_create")
	}

	// Save config state
	if err := config.SaveState(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	printIfNotQuiet("\nDistro: %s %s\n", provider.Name(), provider.Version())

	// Show shared directories
	if len(sharedDirs) > 0 && !quietMode {
		fmt.Println("Shared directories (mount with: mount -t virtiofs <tag> <mountpoint>):")
		for tag, path := range sharedDirs {
			fmt.Printf("  %s -> %s\n", tag, path)
		}
	}

	printlnIfNotQuiet("\nPreparing VM...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare VM: %w", err)
	}
	if timer != nil {
		timer.Mark("vm_prepare")
	}

	printlnIfNotQuiet("Starting VM...")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}
	if timer != nil {
		timer.Mark("vm_start")
	}

	// Write PID file for other processes to detect running VM
	if err := writePIDFile(baseDir, "default"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write PID file: %v\n", err)
	}
	defer cleanupPIDFile(baseDir, "default")

	// Record boot in state
	stateFile := vm.NewStateFile(dataDir)
	if err := stateFile.RecordBoot(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not record boot: %v\n", err)
	}

	// Track whether we had a clean shutdown
	cleanShutdown := false
	defer func() {
		if err := stateFile.RecordShutdown(cleanShutdown); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not record shutdown: %v\n", err)
		}
	}()

	// Get console I/O handles
	vmIn, vmOut, err := mgr.Console()
	if err != nil {
		return fmt.Errorf("get console: %w", err)
	}

	// Print timing report if enabled (before blocking on GUI)
	if timer != nil {
		timer.Mark("gui_launch")
		timer.Report(os.Stderr)
	}

	// Setup signal handler for graceful shutdown from outside
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// shutdownOnce ensures we only run the shutdown sequence once,
	// whether triggered by signal or GUI window close.
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			cancel()
			mgr.CloseConsole()
			if stopErr := mgr.Stop(ctx); stopErr != nil {
				fmt.Fprintf(os.Stderr, "Stop error: %v\n", stopErr)
			}
			cleanShutdown = true
		})
	}

	// Handle external signals in background
	go func() {
		<-sigCh
		shutdown()
	}()

	// Build window title
	windowTitle := fmt.Sprintf("VMTerminal - %s %s", provider.Name(), provider.Version())

	printlnIfNotQuiet("Opening GUI terminal...")

	// Launch GUI terminal window (blocks until window is closed)
	gui.RunTerminal(vmIn, vmOut, windowTitle, shutdown)

	// Ensure shutdown runs even if window closed without triggering onClose
	shutdown()

	// Wait for VM to exit
	if err := mgr.Wait(); err != nil {
		return fmt.Errorf("VM exited with error: %w", err)
	}

	return nil
}

// printSystemInfo displays system architecture and OS information.
func printSystemInfo() {
	arch := runtime.GOARCH
	hostOS := runtime.GOOS

	// Map Go arch names to more readable versions
	archDisplay := arch
	switch arch {
	case "amd64":
		archDisplay = "x86_64"
	case "arm64":
		archDisplay = "Arm64"
	}

	osDisplay := hostOS
	switch hostOS {
	case "darwin":
		osDisplay = "Darwin (macOS)"
	case "linux":
		osDisplay = "Linux"
	}

	fmt.Printf("System Architecture: %s\n", archDisplay)
	fmt.Printf("Host OS: %s\n", osDisplay)

	// Try to get system name
	if hostOS == "darwin" {
		if out, err := exec.Command("sysctl", "-n", "hw.model").Output(); err == nil {
			model := strings.TrimSpace(string(out))
			fmt.Printf("System: %s\n", model)
		}
	} else if hostOS == "linux" {
		// Try to read from /sys/devices/virtual/dmi/id/product_name
		if data, err := os.ReadFile("/sys/devices/virtual/dmi/id/product_name"); err == nil {
			model := strings.TrimSpace(string(data))
			if model != "" {
				fmt.Printf("System: %s\n", model)
			}
		}
	}

	fmt.Println("Making Linux Kernel structure on top of", archDisplay, "Architecture")
}

// resolveDistro determines which distro to use, prompting if needed.
func resolveDistro(cfg *config.State) (distro.ID, error) {
	// If distro is already set, use it
	if cfg.Distro != "" {
		id := distro.ID(cfg.Distro)
		if distro.IsRegistered(id) {
			return id, nil
		}
		if !quietMode {
			fmt.Printf("Warning: configured distro %q not found\n", cfg.Distro)
		}
	}

	// In quiet mode, use default without prompting
	if quietMode {
		return distro.DefaultID(), nil
	}

	// For first run, prompt user to select distro
	fmt.Println("\nAvailable Linux distributions:")
	providers := distro.ListProviders()
	for i, p := range providers {
		marker := ""
		if p.ID() == distro.Alpine {
			marker = " (default)"
		}
		fmt.Printf("  %d. %s %s%s\n", i+1, p.Name(), p.Version(), marker)
	}

	fmt.Print("\nSelect distro [1]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return distro.DefaultID(), nil
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(providers) {
		return distro.DefaultID(), nil
	}

	return providers[choice-1].ID(), nil
}

// interactiveSetup guides the user through initial VM setup.
func interactiveSetup(cfg *config.State, provider distro.Provider, dataDir, cacheDir string) error {
	fmt.Println("Creating File Structure...")

	// Check for FuseFS (optional)
	if err := checkFuseFS(); err != nil {
		fmt.Println("FuseFS not installed.")
		if promptYesNo("Install Fuse?", false) {
			fmt.Println("Please install FuseFS manually for your system.")
			fmt.Println("Continuing with default filesystem...")
		} else {
			fmt.Println("Using default filesystem to mount Linux Kernel.")
		}
	} else {
		fmt.Println("FuseFS available.")
	}

	// Download distro assets
	fmt.Printf("Downloading %s...\n", provider.Name())

	assets := vm.NewAssetManager(cacheDir, provider)
	assetPaths, err := assets.EnsureAssets()
	if err != nil {
		return fmt.Errorf("get asset paths: %w", err)
	}

	// Check setup requirements - some distros (like Ubuntu) use qcow2 directly
	reqs := provider.SetupRequirements()

	if reqs != nil && !reqs.NeedsExtraction {
		// For qcow2-based distros (Ubuntu, Debian, etc.), the converted rootfs.raw
		// IS the disk image - no need to create a separate disk or extract
		fmt.Printf("Using %s cloud image as disk.\n", provider.Name())
	} else {
		// For tarball-based distros (Alpine, Arch), create and populate a disk
		images := vm.NewImageManager(dataDir)
		if !images.DiskExists("disk") {
			fmt.Println("Creating disk image...")
			if _, err := images.EnsureDisk("disk", int64(cfg.DiskSizeMB)); err != nil {
				return fmt.Errorf("create disk: %w", err)
			}
		}

		// Setup filesystem (requires sudo)
		rootfs := vm.NewRootfsManager(dataDir)
		state, _ := rootfs.CheckSetupState("disk")

		if !state.DiskFormatted {
			fmt.Println("\nDisk formatting requires sudo permissions.")
			if !promptYesNo("Give sudo permission to run file building?", true) {
				return fmt.Errorf("sudo permission required for setup")
			}

			fsType := "ext4"
			if reqs != nil && reqs.FSType != "" {
				fsType = reqs.FSType
			}

			fmt.Printf("Formatting disk with %s filesystem...\n", fsType)
			if err := rootfs.FormatDisk("disk", fsType); err != nil {
				return fmt.Errorf("format disk: %w", err)
			}
		}

		if !state.RootfsExtracted {
			fmt.Println("Extracting rootfs to disk...")
			if err := rootfs.ExtractRootfs("disk", assetPaths.Rootfs); err != nil {
				return fmt.Errorf("extract rootfs: %w", err)
			}
		}
	}

	fmt.Printf("Installed %s.\n", provider.Name())

	fmt.Printf("\nWelcome to %s.\n", provider.Name())
	return nil
}

// checkFuseFS checks if FuseFS is available on the system.
func checkFuseFS() error {
	switch runtime.GOOS {
	case "darwin":
		// Check for macFUSE
		if _, err := os.Stat("/Library/Filesystems/macfuse.fs"); err == nil {
			return nil
		}
		if _, err := os.Stat("/usr/local/lib/libfuse.dylib"); err == nil {
			return nil
		}
		return fmt.Errorf("macFUSE not found")
	case "linux":
		// Check for FUSE
		if _, err := os.Stat("/dev/fuse"); err == nil {
			return nil
		}
		return fmt.Errorf("FUSE not found")
	default:
		return fmt.Errorf("unsupported OS")
	}
}

// promptYesNo asks a yes/no question with a default value.
func promptYesNo(question string, defaultYes bool) bool {
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

// getHypervisorInfo returns information about the available hypervisor.
func getHypervisorInfo() (string, string, error) {
	driver, err := hypervisor.NewDriver()
	if err != nil {
		return "", "", err
	}
	info := driver.Info()
	return info.Name, info.Version, nil
}
