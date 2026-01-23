package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var sshKeygenCmd = &cobra.Command{
	Use:   "ssh-keygen",
	Short: "Generate SSH keys for VM access",
	Long:  `Generate an SSH key pair for passwordless access to the VM.`,
	RunE:  runSSHKeygen,
}

var (
	keyType    string
	keyComment string
	forceKey   bool
)

func init() {
	sshKeygenCmd.Flags().StringVarP(&keyType, "type", "t", "ed25519", "Key type (ed25519, rsa)")
	sshKeygenCmd.Flags().StringVarP(&keyComment, "comment", "C", "vmterminal", "Key comment")
	sshKeygenCmd.Flags().BoolVarP(&forceKey, "force", "f", false, "Overwrite existing key")
	rootCmd.AddCommand(sshKeygenCmd)
}

func runSSHKeygen(cmd *cobra.Command, args []string) error {
	// Determine key path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	keyPath := filepath.Join(homeDir, ".ssh", "vmterminal_"+keyType)
	pubKeyPath := keyPath + ".pub"

	// Check if key exists
	if _, err := os.Stat(keyPath); err == nil && !forceKey {
		fmt.Printf("Key already exists: %s\n", keyPath)
		fmt.Println("Use --force to overwrite.")
		return nil
	}

	// Ensure .ssh directory exists
	sshDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create .ssh dir: %w", err)
	}

	// Generate key using ssh-keygen
	sshKeygenBin, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return fmt.Errorf("ssh-keygen not found in PATH: %w", err)
	}

	genArgs := []string{
		"-t", keyType,
		"-f", keyPath,
		"-N", "", // Empty passphrase
		"-C", keyComment,
	}

	genCmd := exec.Command(sshKeygenBin, genArgs...)
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr

	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	fmt.Println()
	fmt.Println("SSH key generated successfully!")
	fmt.Printf("  Private key: %s\n", keyPath)
	fmt.Printf("  Public key:  %s\n", pubKeyPath)
	fmt.Println()

	// Read public key for display
	pubKey, err := os.ReadFile(pubKeyPath)
	if err == nil && len(pubKey) > 0 {
		// Trim trailing newline for cleaner output
		pubKeyStr := string(pubKey)
		if pubKeyStr[len(pubKeyStr)-1] == '\n' {
			pubKeyStr = pubKeyStr[:len(pubKeyStr)-1]
		}

		fmt.Println("To configure SSH in your VM, run these commands inside the VM:")
		fmt.Println()
		fmt.Println("  # Install OpenSSH server (if not installed)")
		fmt.Println("  apk add openssh")
		fmt.Println()
		fmt.Println("  # Enable and start SSH daemon")
		fmt.Println("  rc-update add sshd")
		fmt.Println("  service sshd start")
		fmt.Println()
		fmt.Println("  # Add your public key for passwordless login")
		fmt.Println("  mkdir -p ~/.ssh")
		fmt.Println("  chmod 700 ~/.ssh")
		fmt.Printf("  echo '%s' >> ~/.ssh/authorized_keys\n", pubKeyStr)
		fmt.Println("  chmod 600 ~/.ssh/authorized_keys")
		fmt.Println()
		fmt.Println("Then update your config to use this key:")
		fmt.Printf("  ssh_key_path: %s\n", keyPath)
	}

	return nil
}
