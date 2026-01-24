package cli

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactive config editor",
	Long: `Open an interactive configuration editor.

This allows you to modify VM settings such as CPUs, memory, disk size,
shared directories, and network settings.

Changes are saved to an internal state file and take effect on the next
VM start. Run 'vmterminal reload' or restart the VM to apply changes.`,
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Load current config
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		// Display current configuration
		fmt.Println()
		fmt.Println("VMTerminal Configuration")
		fmt.Println("========================")
		fmt.Println()

		// Get distro display name
		distroDisplay := cfg.Distro
		if provider, err := distro.Get(distro.ID(cfg.Distro)); err == nil {
			distroDisplay = fmt.Sprintf("%s %s", provider.Name(), provider.Version())
		}
		fmt.Printf("Current Distro: %s\n", distroDisplay)
		fmt.Printf("Architecture: %s\n", runtime.GOARCH)
		fmt.Println()

		fmt.Printf("1. CPUs: %d\n", cfg.CPUs)
		fmt.Printf("2. Memory: %d MB\n", cfg.MemoryMB)
		fmt.Printf("3. Disk Size: %d MB\n", cfg.DiskSizeMB)
		fmt.Printf("4. Shared Directories: %s\n", formatSharedDirs(cfg.SharedDirs))
		fmt.Printf("5. Network: %s\n", formatBool(cfg.EnableNetwork))
		fmt.Printf("6. SSH Host Port: %d\n", cfg.SSHHostPort)
		fmt.Println()

		fmt.Print("Enter number to change (or 'q' to quit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "q" || input == "quit" {
			// Save and exit
			if err := config.SaveState(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Println("Configuration saved. Run 'vmterminal reload' to apply changes.")
			return nil
		}

		switch input {
		case "1":
			cfg.CPUs = editInt(reader, "Number of CPUs", cfg.CPUs, 1, 64)
		case "2":
			cfg.MemoryMB = editInt(reader, "Memory (MB)", cfg.MemoryMB, 256, 65536)
		case "3":
			cfg.DiskSizeMB = editInt(reader, "Disk Size (MB)", cfg.DiskSizeMB, 1024, 1024*1024)
		case "4":
			cfg.SharedDirs = editSharedDirs(reader, cfg.SharedDirs)
		case "5":
			cfg.EnableNetwork = editBool(reader, "Enable Network", cfg.EnableNetwork)
		case "6":
			cfg.SSHHostPort = editInt(reader, "SSH Host Port", cfg.SSHHostPort, 0, 65535)
		default:
			fmt.Println("Invalid selection.")
		}
	}
}

// formatSharedDirs formats the shared directories for display.
func formatSharedDirs(dirs []string) string {
	if len(dirs) == 0 {
		return "(none)"
	}
	if len(dirs) == 1 {
		return dirs[0]
	}
	return fmt.Sprintf("%s (+%d more)", dirs[0], len(dirs)-1)
}

// formatBool formats a boolean for display.
func formatBool(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

// editInt prompts for an integer value with validation.
func editInt(reader *bufio.Reader, name string, current, min, max int) int {
	fmt.Printf("%s [%d]: ", name, current)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Invalid number, keeping current value.")
		return current
	}

	if value < min || value > max {
		fmt.Printf("Value must be between %d and %d, keeping current value.\n", min, max)
		return current
	}

	fmt.Printf("Updated %s to %d.\n", name, value)
	return value
}

// editBool prompts for a boolean value.
func editBool(reader *bufio.Reader, name string, current bool) bool {
	currentStr := "n"
	if current {
		currentStr = "y"
	}

	fmt.Printf("%s (y/n) [%s]: ", name, currentStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return current
	}

	value := input == "y" || input == "yes" || input == "true" || input == "1"
	fmt.Printf("Updated %s to %s.\n", name, formatBool(value))
	return value
}

// editSharedDirs allows editing the shared directories.
func editSharedDirs(reader *bufio.Reader, current []string) []string {
	fmt.Println()
	fmt.Println("Shared Directories:")
	if len(current) == 0 {
		fmt.Println("  (none)")
	} else {
		for i, dir := range current {
			fmt.Printf("  %d. %s\n", i+1, dir)
		}
	}
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  a <path> - Add a directory")
	fmt.Println("  d <num>  - Delete a directory by number")
	fmt.Println("  c        - Clear all directories")
	fmt.Println("  q        - Done editing")
	fmt.Println()

	dirs := make([]string, len(current))
	copy(dirs, current)

	for {
		fmt.Print("Shared dirs> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "q" || input == "" {
			fmt.Printf("Updated shared directories (%d total).\n", len(dirs))
			return dirs
		}

		parts := strings.SplitN(input, " ", 2)
		cmd := parts[0]

		switch cmd {
		case "a", "add":
			if len(parts) < 2 {
				fmt.Println("Usage: a <path>")
				continue
			}
			path := strings.TrimSpace(parts[1])
			// Expand ~ to home directory
			if strings.HasPrefix(path, "~") {
				home, _ := os.UserHomeDir()
				path = home + path[1:]
			}
			// Check if path exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("Warning: Path does not exist: %s\n", path)
			}
			dirs = append(dirs, path)
			fmt.Printf("Added: %s\n", path)

		case "d", "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: d <number>")
				continue
			}
			num, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil || num < 1 || num > len(dirs) {
				fmt.Println("Invalid number.")
				continue
			}
			removed := dirs[num-1]
			dirs = append(dirs[:num-1], dirs[num:]...)
			fmt.Printf("Removed: %s\n", removed)

		case "c", "clear":
			dirs = []string{}
			fmt.Println("Cleared all directories.")

		default:
			fmt.Println("Unknown command. Use 'a', 'd', 'c', or 'q'.")
		}
	}
}
