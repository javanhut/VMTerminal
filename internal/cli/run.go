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
	"syscall"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/terminal"
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
//   - console_attach:  <50ms   (terminal raw mode setup)
//   - TOTAL:           <3000ms
//
// Cold path (first run) adds: asset downloads, disk creation, rootfs extraction.
// Run with VMT_TIMING=1 to see actual breakdown.

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start VM (handles all setup interactively if needed)",
	Long: `Start the Linux VM. If the VM is not set up, this command will
interactively guide you through the setup process:

1. Detect system architecture and OS
2. Check for optional dependencies (FuseFS)
3. Download the Linux distribution if needed
4. Set up filesystem (may require sudo)
5. Optionally set VM as default terminal
6. Start VM and attach to console`,
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

	// Print system information
	printSystemInfo()

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
		if err := interactiveSetup(cfg, provider, baseDir, dataDir, cacheDir); err != nil {
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

	fmt.Printf("\nDistro: %s %s\n", provider.Name(), provider.Version())

	// Show shared directories
	if len(sharedDirs) > 0 {
		fmt.Println("Shared directories (mount with: mount -t virtiofs <tag> <mountpoint>):")
		for tag, path := range sharedDirs {
			fmt.Printf("  %s -> %s\n", tag, path)
		}
	}

	fmt.Println("\nPreparing VM...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare VM: %w", err)
	}
	if timer != nil {
		timer.Mark("vm_prepare")
	}

	fmt.Println("Starting VM...")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start VM: %w", err)
	}
	if timer != nil {
		timer.Mark("vm_start")
	}

	// Get console I/O handles
	vmIn, vmOut, err := mgr.Console()
	if err != nil {
		return fmt.Errorf("get console: %w", err)
	}

	fmt.Println("Attaching to console (Ctrl+C to stop VM)...")
	fmt.Println()

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Attach in background, stop on signal
	attachDone := make(chan error, 1)
	go func() {
		attachDone <- attachToConsole(ctx, vmIn, vmOut)
	}()

	// Print timing report if enabled (before blocking on console)
	if timer != nil {
		timer.Mark("console_attach")
		timer.Report(os.Stderr)
	}

	// Wait for either signal or attach completion
	select {
	case <-sigCh:
		fmt.Println("\nStopping VM...")
		cancel()
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
		fmt.Printf("Warning: configured distro %q not found\n", cfg.Distro)
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
func interactiveSetup(cfg *config.State, provider distro.Provider, baseDir, dataDir, cacheDir string) error {
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

	// Create disk image
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

		reqs := provider.SetupRequirements()
		fsType := reqs.FSType
		if fsType == "" {
			fsType = "ext4"
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

	fmt.Printf("Installed %s.\n", provider.Name())

	// Prompt to set as default terminal
	if promptYesNo("Set as default for Terminal?", true) {
		fmt.Println("Setting this in system shell...")
		if err := setAsDefaultShell(baseDir); err != nil {
			fmt.Printf("Warning: could not set as default: %v\n", err)
			fmt.Println("You can manually add vmterminal to your shell's profile.")
		} else {
			fmt.Println("Finished!")
			fmt.Println("Restart Terminal or run 'vmterminal reload'.")
		}
	}

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

// setAsDefaultShell configures vmterminal as the default terminal shell.
func setAsDefaultShell(baseDir string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Determine which shell config file to modify
	shell := os.Getenv("SHELL")
	var configFile string
	var configLine string

	binPath := "/usr/local/bin/vmterminal"
	// Check if binary exists, otherwise use current executable
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if exePath, err := os.Executable(); err == nil {
			binPath = exePath
		}
	}

	configLine = fmt.Sprintf("\n# VMTerminal - Launch Linux VM on terminal start\nif [ -z \"$VMTERMINAL_SKIP\" ] && [ -x %s ]; then\n  exec %s\nfi\n", binPath, binPath)

	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(homeDir, ".zshrc")
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(homeDir, ".bashrc")
	default:
		configFile = filepath.Join(homeDir, ".profile")
	}

	// Read existing config
	content, _ := os.ReadFile(configFile)

	// Check if already configured
	if strings.Contains(string(content), "VMTERMINAL_SKIP") {
		return nil // Already configured
	}

	// Append configuration
	f, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(configLine); err != nil {
		return err
	}

	return nil
}

// attachToConsole attaches the current terminal to VM console I/O.
func attachToConsole(ctx context.Context, vmIn interface{}, vmOut interface{}) error {
	// Type assert the interfaces to io.Writer and io.Reader
	writer, ok := vmIn.(interface{ Write([]byte) (int, error) })
	if !ok {
		return fmt.Errorf("vmIn does not implement Write")
	}
	reader, ok := vmOut.(interface{ Read([]byte) (int, error) })
	if !ok {
		return fmt.Errorf("vmOut does not implement Read")
	}

	console := terminal.Current()

	// Create a context that we can cancel on escape sequence
	attachCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-attachCtx.Done():
		}
	}()

	// Use the terminal package's Attach for bidirectional I/O
	return console.Attach(attachCtx, writer, reader)
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
