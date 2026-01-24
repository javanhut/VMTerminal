package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/javanstorm/vmterminal/pkg/hypervisor"
	"github.com/spf13/cobra"
)

var (
	mountTag   string
	mountCheck bool
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Show mount commands for shared directories",
	Long: `Display shell commands to mount shared directories inside the VM.

The output can be copy-pasted into the VM shell or piped to execute.
Shared directories use virtio-fs (macOS only).

Examples:
  vmterminal mount              # Show mount script for all shares
  vmterminal mount --tag home   # Show command for single share
  vmterminal mount --check      # Verify platform capabilities`,
	RunE: runMount,
}

func init() {
	mountCmd.Flags().StringVar(&mountTag, "tag", "", "Show mount command for specific share tag only")
	mountCmd.Flags().BoolVar(&mountCheck, "check", false, "Check platform capabilities for shared directories")
	rootCmd.AddCommand(mountCmd)
}

func runMount(cmd *cobra.Command, args []string) error {
	// Check capabilities first if requested
	if mountCheck {
		return runMountCheck()
	}

	// Load config
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}

	// Convert SharedDirs slice to map (tag derived from basename)
	shares := make(map[string]string)
	for _, dir := range cfg.SharedDirs {
		tag := filepath.Base(dir)
		// Handle home directory specially
		if dir == os.Getenv("HOME") || tag == filepath.Base(os.Getenv("HOME")) {
			tag = "home"
		}
		shares[tag] = dir
	}

	if len(shares) == 0 {
		fmt.Println("# No shared directories configured")
		fmt.Println("# Use 'vmterminal config' to add shared directories")
		return nil
	}

	helper := vm.NewMountHelper(shares)

	// Check capabilities and warn if needed
	driver, err := hypervisor.NewDriver()
	if err == nil {
		caps := driver.Capabilities()
		if !caps.SharedDirs {
			printLinuxWarning()
			return nil
		}
	}

	if mountTag != "" {
		// Single share
		if _, ok := shares[mountTag]; !ok {
			return fmt.Errorf("unknown mount tag %q, available: %v", mountTag, helper.Tags())
		}
		mountpoint := "/mnt/host/" + mountTag
		fmt.Println(helper.GenerateMountCommand(mountTag, mountpoint))
	} else {
		// All shares
		fmt.Println("# Mount shared directories in your VM:")
		fmt.Println("# Copy and paste the following into your VM shell")
		fmt.Println()

		// Print simple one-liner commands for easy copy-paste
		for _, tag := range helper.Tags() {
			mountpoint := "/mnt/host/" + tag
			fmt.Println(helper.GenerateMountCommand(tag, mountpoint))
		}

		fmt.Println()
		fmt.Println("# Or generate a full script with error handling:")
		fmt.Println("# vmterminal mount > mount-shares.sh && chmod +x mount-shares.sh")
	}

	return nil
}

func runMountCheck() error {
	driver, err := hypervisor.NewDriver()
	if err != nil {
		return fmt.Errorf("hypervisor not available: %w", err)
	}

	info := driver.Info()
	caps := driver.Capabilities()

	fmt.Println("Platform Capabilities")
	fmt.Println("=====================")
	fmt.Printf("  Hypervisor: %s v%s (%s)\n", info.Name, info.Version, info.Arch)
	fmt.Printf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	fmt.Println("Feature Support:")
	fmt.Printf("  Shared Directories: %s\n", capabilityStatus(caps.SharedDirs))
	fmt.Printf("  Networking: %s\n", capabilityStatus(caps.Networking))
	fmt.Printf("  Snapshots: %s\n", capabilityStatus(caps.Snapshots))

	if !caps.SharedDirs {
		fmt.Println()
		printLinuxWarning()
	}

	return nil
}

func capabilityStatus(supported bool) string {
	if supported {
		return "supported"
	}
	return "not available"
}

func printLinuxWarning() {
	fmt.Println("# Shared directories not available on Linux KVM")
	fmt.Println("#")
	fmt.Println("# The hype library lacks virtio-fs support.")
	fmt.Println("# Workarounds:")
	fmt.Println("#   1. Use SSH to transfer files (see 'vmterminal ssh')")
	fmt.Println("#   2. Mount host directories via NFS/SSHFS from guest")
	fmt.Println("#   3. Use a different hypervisor with virtio-fs support")
}
