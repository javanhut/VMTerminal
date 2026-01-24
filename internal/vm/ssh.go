package vm

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// SSHKeyManager handles SSH key generation and management for VM access.
type SSHKeyManager struct {
	dataDir string
}

// NewSSHKeyManager creates a new SSH key manager.
// Keys are stored in {dataDir}/ssh/ directory.
func NewSSHKeyManager(dataDir string) *SSHKeyManager {
	return &SSHKeyManager{dataDir: dataDir}
}

// sshDir returns the path to the SSH keys directory.
func (m *SSHKeyManager) sshDir() string {
	return filepath.Join(m.dataDir, "ssh")
}

// privateKeyPath returns the path to the private key file.
func (m *SSHKeyManager) privateKeyPath() string {
	return filepath.Join(m.sshDir(), "vmterminal")
}

// publicKeyPath returns the path to the public key file.
func (m *SSHKeyManager) publicKeyPath() string {
	return filepath.Join(m.sshDir(), "vmterminal.pub")
}

// EnsureKeyPair generates an ed25519 key pair if it doesn't exist.
// Returns paths to the private and public key files.
func (m *SSHKeyManager) EnsureKeyPair() (privateKeyPath, publicKeyPath string, err error) {
	privPath := m.privateKeyPath()
	pubPath := m.publicKeyPath()

	// Check if keys already exist
	if m.KeyPairExists() {
		return privPath, pubPath, nil
	}

	// Create SSH directory with secure permissions
	sshDir := m.sshDir()
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", "", fmt.Errorf("create ssh directory: %w", err)
	}

	// Generate ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Write private key in OpenSSH format
	if err := m.writePrivateKey(privPath, privKey); err != nil {
		return "", "", fmt.Errorf("write private key: %w", err)
	}

	// Write public key in OpenSSH format
	if err := m.writePublicKey(pubPath, pubKey); err != nil {
		// Clean up private key on failure
		os.Remove(privPath)
		return "", "", fmt.Errorf("write public key: %w", err)
	}

	return privPath, pubPath, nil
}

// KeyPairExists returns true if both private and public keys exist.
func (m *SSHKeyManager) KeyPairExists() bool {
	privPath := m.privateKeyPath()
	pubPath := m.publicKeyPath()

	_, privErr := os.Stat(privPath)
	_, pubErr := os.Stat(pubPath)

	return privErr == nil && pubErr == nil
}

// PrivateKeyPath returns the path to the private key, or error if not generated.
func (m *SSHKeyManager) PrivateKeyPath() (string, error) {
	path := m.privateKeyPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("SSH key not generated; run 'vmterminal ssh keygen' first")
		}
		return "", err
	}
	return path, nil
}

// PublicKeyContent returns the public key content suitable for authorized_keys.
func (m *SSHKeyManager) PublicKeyContent() (string, error) {
	path := m.publicKeyPath()
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("SSH key not generated; run 'vmterminal ssh keygen' first")
		}
		return "", err
	}
	return string(content), nil
}

// writePrivateKey writes an ed25519 private key in OpenSSH format.
func (m *SSHKeyManager) writePrivateKey(path string, privKey ed25519.PrivateKey) error {
	// Use x/crypto/ssh to marshal the private key in OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(privKey, "vmterminal key")
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}

	pemData := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(path, pemData, 0600); err != nil {
		return err
	}

	return nil
}

// writePublicKey writes an ed25519 public key in OpenSSH authorized_keys format.
func (m *SSHKeyManager) writePublicKey(path string, pubKey ed25519.PublicKey) error {
	// Convert to ssh.PublicKey and format for authorized_keys
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("convert public key: %w", err)
	}

	// Format: ssh-ed25519 <base64> <comment>
	authorizedKey := ssh.MarshalAuthorizedKey(sshPubKey)
	// Add comment
	keyLine := fmt.Sprintf("%svmterminal@vmterminal\n", string(authorizedKey[:len(authorizedKey)-1]))

	if err := os.WriteFile(path, []byte(keyLine), 0644); err != nil {
		return err
	}

	return nil
}

// InjectSSHKey adds the public key to /root/.ssh/authorized_keys in a mounted rootfs.
// mountPoint should be the path where the rootfs disk is mounted.
// This requires root privileges since the mount point is owned by root.
func (m *SSHKeyManager) InjectSSHKey(mountPoint string) error {
	pubKeyContent, err := m.PublicKeyContent()
	if err != nil {
		return err
	}

	// Create /root/.ssh directory with correct permissions
	sshDir := filepath.Join(mountPoint, "root", ".ssh")
	cmd := exec.Command("sudo", "mkdir", "-p", sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create .ssh directory: %w", err)
	}

	// Set permissions on .ssh directory
	cmd = exec.Command("sudo", "chmod", "700", sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chmod .ssh directory: %w", err)
	}

	// Write authorized_keys file
	authorizedKeys := filepath.Join(sshDir, "authorized_keys")
	// Use tee with sudo to write the file
	cmd = exec.Command("sudo", "tee", authorizedKeys)
	cmd.Stdin = stringReader(pubKeyContent)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write authorized_keys: %w", err)
	}

	// Set permissions on authorized_keys
	cmd = exec.Command("sudo", "chmod", "600", authorizedKeys)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chmod authorized_keys: %w", err)
	}

	// Set ownership (root:root)
	cmd = exec.Command("sudo", "chown", "-R", "root:root", sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chown .ssh directory: %w", err)
	}

	return nil
}

// stringReader returns an io.Reader for a string.
type stringReaderType struct {
	s string
	i int
}

func stringReader(s string) *stringReaderType {
	return &stringReaderType{s: s, i: 0}
}

func (r *stringReaderType) Read(b []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, os.ErrClosed
	}
	n = copy(b, r.s[r.i:])
	r.i += n
	return n, nil
}
