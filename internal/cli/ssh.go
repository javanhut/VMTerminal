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

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Manage SSH access to VM",
	Long: `Manage SSH key pairs and connect to the VM via SSH.

SSH access requires networking support:
- macOS: Uses virtio-net with NAT networking
- Linux: Requires manual tap/bridge setup (virtio-net not available)

Examples:
  vmterminal ssh keygen    # Generate SSH key pair
  vmterminal ssh pubkey    # Print public key (for manual injection)
  vmterminal ssh connect   # Show SSH connection command`,
}

var sshKeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate SSH key pair for VM access",
	Long:  `Generate an ed25519 SSH key pair for VM access. Keys are stored in ~/.vmterminal/ssh/`,
	RunE:  runSSHKeygen,
}

var sshPubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Print public key for authorized_keys",
	Long:  `Print the SSH public key content suitable for adding to authorized_keys in the VM.`,
	RunE:  runSSHPubkey,
}

var sshConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Show SSH connection command",
	Long:  `Show the SSH command to connect to the VM. Platform-specific guidance included.`,
	RunE:  runSSHConnect,
}

func init() {
	sshCmd.AddCommand(sshKeygenCmd)
	sshCmd.AddCommand(sshPubkeyCmd)
	sshCmd.AddCommand(sshConnectCmd)
	rootCmd.AddCommand(sshCmd)
}

func runSSHKeygen(cmd *cobra.Command, args []string) error {
	// Get data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dataDir := filepath.Join(homeDir, ".vmterminal")

	manager := vm.NewSSHKeyManager(dataDir)

	// Check if keys already exist
	if manager.KeyPairExists() {
		privPath, _ := manager.PrivateKeyPath()
		fmt.Println("SSH key pair already exists:")
		fmt.Printf("  Private key: %s\n", privPath)
		fmt.Printf("  Public key: %s.pub\n", privPath)
		return nil
	}

	// Generate keys
	privPath, pubPath, err := manager.EnsureKeyPair()
	if err != nil {
		return fmt.Errorf("generate key pair: %w", err)
	}

	fmt.Println("SSH key pair generated:")
	fmt.Printf("  Private key: %s\n", privPath)
	fmt.Printf("  Public key: %s\n", pubPath)
	fmt.Println()
	fmt.Println("The public key will be automatically injected into new VMs.")
	fmt.Println("For existing VMs, use 'vmterminal ssh pubkey' to get the key.")

	return nil
}

func runSSHPubkey(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dataDir := filepath.Join(homeDir, ".vmterminal")

	manager := vm.NewSSHKeyManager(dataDir)

	content, err := manager.PublicKeyContent()
	if err != nil {
		return err
	}

	// Print just the key for easy copy-paste or piping
	fmt.Print(content)
	return nil
}

func runSSHConnect(cmd *cobra.Command, args []string) error {
	// Load config for SSH port
	cfg, err := config.LoadState()
	if err != nil {
		cfg = config.DefaultState()
	}

	// Get private key path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dataDir := filepath.Join(homeDir, ".vmterminal")
	manager := vm.NewSSHKeyManager(dataDir)

	privKeyPath, err := manager.PrivateKeyPath()
	if err != nil {
		fmt.Println("# No SSH key found. Generate one with:")
		fmt.Println("#   vmterminal ssh keygen")
		return nil
	}

	// Check platform capabilities
	driver, err := hypervisor.NewDriver()
	if err != nil {
		return fmt.Errorf("hypervisor not available: %w", err)
	}

	caps := driver.Capabilities()

	if caps.Networking {
		// macOS with virtio-net - networking works out of the box
		fmt.Println("# SSH connection command:")
		fmt.Printf("ssh -i %s -p %d root@localhost\n", privKeyPath, cfg.SSHHostPort)
		fmt.Println()
		fmt.Println("# Or to skip host key checking (for testing):")
		fmt.Printf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -p %d root@localhost\n", privKeyPath, cfg.SSHHostPort)
	} else {
		// Linux without virtio-net - needs manual networking setup
		printLinuxSSHGuide(privKeyPath)
	}

	return nil
}

func printLinuxSSHGuide(privKeyPath string) {
	fmt.Println("# SSH not directly available on Linux KVM (no virtio-net)")
	fmt.Println("#")
	fmt.Println("# The hype library lacks networking support.")
	fmt.Println("# To enable SSH access, set up tap networking:")
	fmt.Println("#")
	fmt.Printf("# On host (%s):\n", runtime.GOOS)
	fmt.Println("#   sudo ip tuntap add dev tap0 mode tap user $USER")
	fmt.Println("#   sudo ip addr add 192.168.100.1/24 dev tap0")
	fmt.Println("#   sudo ip link set tap0 up")
	fmt.Println("#")
	fmt.Println("# In VM (via console):")
	fmt.Println("#   ip addr add 192.168.100.2/24 dev eth0")
	fmt.Println("#   ip link set eth0 up")
	fmt.Println("#")
	fmt.Println("# Once networking configured:")
	fmt.Printf("ssh -i %s -p 22 root@192.168.100.2\n", privKeyPath)
	fmt.Println()
	fmt.Println("# Alternative: Use serial console for direct access")
	fmt.Println("# (no networking required)")
}
