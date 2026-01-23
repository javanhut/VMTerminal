package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/javanstorm/vmterminal/internal/vm"
	"github.com/spf13/cobra"
)

var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage VM instances",
	Long:  `Create, list, delete, and switch between multiple VM instances.`,
}

var vmCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new VM",
	Long:  `Create a new VM instance with the specified name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVMCreate,
}

var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	Long:  `List all VM instances, marking the active one with *.`,
	RunE:  runVMList,
}

var vmDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a VM",
	Long:  `Delete a VM instance and optionally its data.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVMDelete,
}

var vmUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set active VM",
	Long:  `Set the specified VM as the active (default) VM for other commands.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVMUse,
}

var vmShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show VM details",
	Long:  `Show details for a VM. If no name specified, shows the active VM.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runVMShow,
}

// Flags for vm create
var (
	vmCreateCPUs     int
	vmCreateMemory   int
	vmCreateDiskSize int
	vmCreateDistro   string
)

// Flag for vm delete
var vmDeleteData bool

func init() {
	// vm create flags
	vmCreateCmd.Flags().IntVarP(&vmCreateCPUs, "cpus", "c", runtime.NumCPU(), "Number of virtual CPUs")
	vmCreateCmd.Flags().IntVarP(&vmCreateMemory, "memory", "m", 2048, "Memory in MB")
	vmCreateCmd.Flags().IntVarP(&vmCreateDiskSize, "disk-size", "s", 10240, "Disk size in MB")
	vmCreateCmd.Flags().StringVarP(&vmCreateDistro, "distro", "d", "alpine", "Linux distribution")

	// vm delete flags
	vmDeleteCmd.Flags().BoolVar(&vmDeleteData, "data", false, "Also delete VM data (disk, state)")

	// Add subcommands
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmDeleteCmd)
	vmCmd.AddCommand(vmUseCmd)
	vmCmd.AddCommand(vmShowCmd)

	// Register vm command
	rootCmd.AddCommand(vmCmd)
}

func getRegistry() (*vm.Registry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	return vm.NewRegistry(baseDir), nil
}

func runVMCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid VM name: contains forbidden characters")
	}

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	entry := vm.VMEntry{
		Name:       name,
		Distro:     vmCreateDistro,
		CPUs:       vmCreateCPUs,
		MemoryMB:   vmCreateMemory,
		DiskSizeMB: vmCreateDiskSize,
	}

	if err := registry.CreateVM(entry); err != nil {
		return fmt.Errorf("create VM: %w", err)
	}

	fmt.Printf("Created VM '%s'\n", name)
	fmt.Printf("  Distro: %s\n", entry.Distro)
	fmt.Printf("  CPUs: %d\n", entry.CPUs)
	fmt.Printf("  Memory: %d MB\n", entry.MemoryMB)
	fmt.Printf("  Disk: %d MB\n", entry.DiskSizeMB)
	fmt.Println()
	fmt.Printf("To use this VM: vmterminal vm use %s\n", name)

	return nil
}

func runVMList(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	vms, err := registry.ListVMs()
	if err != nil {
		return fmt.Errorf("list VMs: %w", err)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found. Create one with: vmterminal vm create <name>")
		return nil
	}

	active, _ := registry.GetActive()

	fmt.Println("VMs:")
	for _, entry := range vms {
		marker := " "
		if entry.Name == active {
			marker = "*"
		}
		fmt.Printf("  %s %s (%s, %d CPUs, %d MB)\n",
			marker, entry.Name, entry.Distro, entry.CPUs, entry.MemoryMB)
	}

	return nil
}

func runVMDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	// Check VM exists
	_, err = registry.GetVM(name)
	if err != nil {
		return err
	}

	// Delete VM data if requested
	if vmDeleteData {
		if err := registry.DeleteVMData(name); err != nil {
			fmt.Printf("Warning: failed to delete VM data: %v\n", err)
		} else {
			fmt.Printf("Deleted VM data for '%s'\n", name)
		}
	}

	// Delete from registry
	if err := registry.DeleteVM(name); err != nil {
		return fmt.Errorf("delete VM: %w", err)
	}

	fmt.Printf("Deleted VM '%s'\n", name)

	if !vmDeleteData {
		fmt.Println("Note: VM data still exists. Use --data to delete it.")
	}

	return nil
}

func runVMUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if err := registry.SetActive(name); err != nil {
		return fmt.Errorf("set active: %w", err)
	}

	fmt.Printf("Active VM set to '%s'\n", name)
	return nil
}

func runVMShow(cmd *cobra.Command, args []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		name, err = registry.GetActive()
		if err != nil {
			return err
		}
		if name == "" {
			fmt.Println("No active VM. Specify a VM name or use: vmterminal vm use <name>")
			return nil
		}
	}

	entry, err := registry.GetVM(name)
	if err != nil {
		return err
	}

	active, _ := registry.GetActive()
	activeMarker := ""
	if entry.Name == active {
		activeMarker = " (active)"
	}

	fmt.Printf("VM: %s%s\n", entry.Name, activeMarker)
	fmt.Printf("  Distro: %s\n", entry.Distro)
	fmt.Printf("  CPUs: %d\n", entry.CPUs)
	fmt.Printf("  Memory: %d MB\n", entry.MemoryMB)
	fmt.Printf("  Disk Size: %d MB\n", entry.DiskSizeMB)
	fmt.Printf("  Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Data Dir: %s\n", registry.VMDataDir(entry.Name))

	return nil
}
