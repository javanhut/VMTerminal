package cli

import (
	"fmt"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/spf13/cobra"
)

var sshSetupCmd = &cobra.Command{
	Use:   "ssh-setup",
	Short: "Show SSH setup instructions",
	Long:  `Display instructions for setting up SSH access to the VM.`,
	RunE:  runSSHSetup,
}

func init() {
	rootCmd.AddCommand(sshSetupCmd)
}

func runSSHSetup(cmd *cobra.Command, args []string) error {
	cfg := config.Global
	if cfg == nil {
		if err := config.Load(); err != nil {
			cfg = config.DefaultConfig()
		} else {
			cfg = config.Global
		}
	}

	fmt.Println("SSH Setup for VMTerminal")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("SSH access allows you to connect to your VM without using the console.")
	fmt.Println()
	fmt.Println("STEP 1: Generate SSH Key (on host)")
	fmt.Println("-----------------------------------")
	fmt.Println("  vmterminal ssh-keygen")
	fmt.Println()
	fmt.Println("STEP 2: Start VM and Install OpenSSH (in VM)")
	fmt.Println("---------------------------------------------")
	fmt.Println("  # Start your VM")
	fmt.Println("  vmterminal run")
	fmt.Println()
	fmt.Println("  # Inside the VM, install and enable SSH")
	fmt.Println("  apk add openssh")
	fmt.Println("  rc-update add sshd")
	fmt.Println("  service sshd start")
	fmt.Println()
	fmt.Println("STEP 3: Configure Authentication (in VM)")
	fmt.Println("-----------------------------------------")
	fmt.Println("  # Create .ssh directory")
	fmt.Println("  mkdir -p ~/.ssh && chmod 700 ~/.ssh")
	fmt.Println()
	fmt.Println("  # Add your public key (copy from host)")
	fmt.Println("  # Run this on host to get your public key:")
	fmt.Println("  #   cat ~/.ssh/vmterminal_ed25519.pub")
	fmt.Println()
	fmt.Println("  # Then in VM:")
	fmt.Println("  echo 'YOUR_PUBLIC_KEY_HERE' >> ~/.ssh/authorized_keys")
	fmt.Println("  chmod 600 ~/.ssh/authorized_keys")
	fmt.Println()
	fmt.Println("STEP 4: Find VM IP Address (in VM)")
	fmt.Println("-----------------------------------")
	fmt.Println("  # Get the VM's IP address")
	fmt.Println("  ip addr show eth0 | grep 'inet '")
	fmt.Println()
	fmt.Println("  # The IP will be something like 192.168.64.X")
	fmt.Println()
	fmt.Println("STEP 5: Connect via SSH (on host)")
	fmt.Println("----------------------------------")
	fmt.Println("  # Using vmterminal (configure IP in config first):")
	fmt.Println("  vmterminal ssh")
	fmt.Println()
	fmt.Println("  # Or directly with ssh:")
	fmt.Printf("  ssh -i ~/.ssh/vmterminal_ed25519 %s@<VM_IP>\n", cfg.SSHUser)
	fmt.Println()
	fmt.Println("CONFIGURATION")
	fmt.Println("-------------")
	fmt.Println("Add to ~/.vmterminal/config.yaml:")
	fmt.Println()
	fmt.Printf("  ssh_user: %s\n", cfg.SSHUser)
	fmt.Printf("  ssh_port: %d\n", cfg.SSHPort)
	fmt.Println("  ssh_key_path: ~/.ssh/vmterminal_ed25519")
	fmt.Println("  vm_ip: <YOUR_VM_IP>")
	fmt.Println()
	fmt.Println("TROUBLESHOOTING")
	fmt.Println("---------------")
	fmt.Println("- If SSH times out: Check VM has network (ip addr)")
	fmt.Println("- If connection refused: Check sshd is running (service sshd status)")
	fmt.Println("- If permission denied: Check authorized_keys permissions")
	fmt.Println("- Linux hosts: Networking not yet supported, use console")
	fmt.Println()

	return nil
}
