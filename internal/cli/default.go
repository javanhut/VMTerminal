package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/spf13/cobra"
)

var defaultCmd = &cobra.Command{
	Use:   "default",
	Short: "Switch terminal default between VM and host",
	Long: `Manage whether vmterminal starts automatically when you open a terminal.

Use --host to switch back to the host OS shell as the default.
Use --vm to set the VM as the default terminal.`,
	RunE: runDefault,
}

var (
	defaultHost bool
	defaultVM   bool
)

func init() {
	defaultCmd.Flags().BoolVar(&defaultHost, "host", false, "Switch terminal default back to host OS")
	defaultCmd.Flags().BoolVar(&defaultVM, "vm", false, "Set VM as default terminal")
}

func runDefault(cmd *cobra.Command, args []string) error {
	if !defaultHost && !defaultVM {
		// Show current status
		cfg, err := config.LoadState()
		if err != nil {
			cfg = config.DefaultState()
		}

		if cfg.IsDefaultTerminal {
			fmt.Println("Current default: VM (vmterminal starts automatically)")
		} else {
			fmt.Println("Current default: Host OS shell")
		}
		fmt.Println()
		fmt.Println("Use --host to switch to host OS, or --vm to set VM as default.")
		return nil
	}

	if defaultHost && defaultVM {
		return fmt.Errorf("cannot specify both --host and --vm")
	}

	if defaultHost {
		return switchToHost()
	}

	return switchToVM()
}

// switchToHost removes vmterminal from shell startup.
func switchToHost() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	shell := os.Getenv("SHELL")
	var configFile string

	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(homeDir, ".zshrc")
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(homeDir, ".bashrc")
	default:
		configFile = filepath.Join(homeDir, ".profile")
	}

	// Read existing config
	content, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Shell configuration file not found.")
			fmt.Println("VMTerminal is not set as default.")
			return nil
		}
		return err
	}

	// Check if vmterminal config exists
	if !strings.Contains(string(content), "VMTERMINAL_SKIP") {
		fmt.Println("VMTerminal is not currently set as default terminal.")
		return nil
	}

	// Remove vmterminal configuration block
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inBlock := false

	for _, line := range lines {
		if strings.Contains(line, "# VMTerminal - Launch Linux VM") {
			inBlock = true
			continue
		}
		if inBlock && strings.HasPrefix(strings.TrimSpace(line), "fi") {
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		newLines = append(newLines, line)
	}

	// Write back
	newContent := strings.Join(newLines, "\n")
	// Remove any trailing multiple newlines
	for strings.HasSuffix(newContent, "\n\n") {
		newContent = strings.TrimSuffix(newContent, "\n")
	}

	if err := os.WriteFile(configFile, []byte(newContent), 0644); err != nil {
		return err
	}

	// Update state
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}
	cfg.IsDefaultTerminal = false
	if err := config.SaveState(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	fmt.Println("Switched terminal default to host OS.")
	fmt.Println("Restart your terminal or run:")
	fmt.Printf("  source %s\n", configFile)
	fmt.Println()
	fmt.Println("You can still run 'vmterminal' to start the VM manually.")

	return nil
}

// switchToVM sets vmterminal as the default shell.
func switchToVM() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(homeDir, ".vmterminal")

	// Check if VM is set up
	dataDir := filepath.Join(baseDir, "data", "default")
	diskPath := filepath.Join(dataDir, "disk.img")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		fmt.Println("VM is not set up yet.")
		fmt.Println("Run 'vmterminal run' first to set up the VM.")
		return nil
	}

	if err := setAsDefaultShell(baseDir); err != nil {
		return fmt.Errorf("set as default: %w", err)
	}

	// Update state
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}
	cfg.IsDefaultTerminal = true
	if err := config.SaveState(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	}

	shell := os.Getenv("SHELL")
	var configFile string
	switch {
	case strings.Contains(shell, "zsh"):
		configFile = filepath.Join(homeDir, ".zshrc")
	case strings.Contains(shell, "bash"):
		configFile = filepath.Join(homeDir, ".bashrc")
	default:
		configFile = filepath.Join(homeDir, ".profile")
	}

	fmt.Println("VM set as default terminal.")
	fmt.Println("Restart your terminal or run:")
	fmt.Printf("  source %s\n", configFile)
	fmt.Println()
	fmt.Println("To skip vmterminal temporarily, set VMTERMINAL_SKIP=1 before opening terminal.")

	return nil
}

// promptYesNoDefault asks a yes/no question with a default value for default command.
func promptYesNoDefault(question string, defaultYes bool) bool {
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
