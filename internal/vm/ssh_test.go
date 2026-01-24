package vm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHKeyManagerEnsureKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Generate key pair
	privPath, pubPath, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() error = %v", err)
	}

	// Verify paths are in expected location
	expectedPrivPath := filepath.Join(tmpDir, "ssh", "vmterminal")
	expectedPubPath := filepath.Join(tmpDir, "ssh", "vmterminal.pub")

	if privPath != expectedPrivPath {
		t.Errorf("Private key path = %q, want %q", privPath, expectedPrivPath)
	}
	if pubPath != expectedPubPath {
		t.Errorf("Public key path = %q, want %q", pubPath, expectedPubPath)
	}

	// Verify files exist
	if _, err := os.Stat(privPath); err != nil {
		t.Errorf("Private key file not created: %v", err)
	}
	if _, err := os.Stat(pubPath); err != nil {
		t.Errorf("Public key file not created: %v", err)
	}

	// Verify private key permissions (should be 0600)
	info, _ := os.Stat(privPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Private key permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSSHKeyManagerIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// First generation
	privPath1, pubPath1, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("First EnsureKeyPair() error = %v", err)
	}

	// Read original content
	origPriv, _ := os.ReadFile(privPath1)
	origPub, _ := os.ReadFile(pubPath1)

	// Second call should be idempotent
	privPath2, pubPath2, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("Second EnsureKeyPair() error = %v", err)
	}

	// Paths should be the same
	if privPath1 != privPath2 || pubPath1 != pubPath2 {
		t.Error("EnsureKeyPair() not idempotent: paths differ")
	}

	// Content should be the same (keys not regenerated)
	newPriv, _ := os.ReadFile(privPath2)
	newPub, _ := os.ReadFile(pubPath2)

	if string(origPriv) != string(newPriv) {
		t.Error("EnsureKeyPair() regenerated private key")
	}
	if string(origPub) != string(newPub) {
		t.Error("EnsureKeyPair() regenerated public key")
	}
}

func TestSSHKeyManagerKeyPairExists(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Before generation
	if manager.KeyPairExists() {
		t.Error("KeyPairExists() = true before generation")
	}

	// Generate keys
	_, _, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() error = %v", err)
	}

	// After generation
	if !manager.KeyPairExists() {
		t.Error("KeyPairExists() = false after generation")
	}
}

func TestSSHKeyManagerPrivateKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Before generation should error
	_, err := manager.PrivateKeyPath()
	if err == nil {
		t.Error("PrivateKeyPath() should error before key generation")
	}
	if !strings.Contains(err.Error(), "not generated") {
		t.Errorf("PrivateKeyPath() error = %q, should mention 'not generated'", err)
	}

	// Generate keys
	_, _, _ = manager.EnsureKeyPair()

	// After generation should succeed
	path, err := manager.PrivateKeyPath()
	if err != nil {
		t.Errorf("PrivateKeyPath() error = %v after generation", err)
	}
	if path == "" {
		t.Error("PrivateKeyPath() returned empty path")
	}
}

func TestSSHKeyManagerPublicKeyContent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Before generation should error
	_, err := manager.PublicKeyContent()
	if err == nil {
		t.Error("PublicKeyContent() should error before key generation")
	}

	// Generate keys
	_, _, _ = manager.EnsureKeyPair()

	// After generation should succeed
	content, err := manager.PublicKeyContent()
	if err != nil {
		t.Errorf("PublicKeyContent() error = %v after generation", err)
	}

	// Verify it's a valid OpenSSH ed25519 public key
	if !strings.HasPrefix(content, "ssh-ed25519 ") {
		t.Errorf("Public key content doesn't start with 'ssh-ed25519 ': %q", content[:min(len(content), 50)])
	}

	// Verify it has the vmterminal comment
	if !strings.Contains(content, "vmterminal@vmterminal") {
		t.Error("Public key missing vmterminal comment")
	}
}

func TestSSHKeyManagerPrivateKeyFormat(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	privPath, _, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() error = %v", err)
	}

	// Read private key content
	content, err := os.ReadFile(privPath)
	if err != nil {
		t.Fatalf("Read private key error = %v", err)
	}

	// Verify OpenSSH format
	if !strings.HasPrefix(string(content), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("Private key not in OpenSSH format")
	}
	if !strings.Contains(string(content), "-----END OPENSSH PRIVATE KEY-----") {
		t.Error("Private key missing END marker")
	}
}

func TestSSHKeyManagerInjectSSHKey(t *testing.T) {
	// Skip if not running as root (needed for sudo operations)
	if os.Getuid() != 0 {
		t.Skip("Skipping SSH key injection test: requires root privileges")
	}

	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Generate keys first
	_, _, err := manager.EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() error = %v", err)
	}

	// Create a mock mount point
	mountPoint := filepath.Join(tmpDir, "rootfs")
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		t.Fatalf("Create mount point: %v", err)
	}

	// Inject SSH key
	if err := manager.InjectSSHKey(mountPoint); err != nil {
		t.Fatalf("InjectSSHKey() error = %v", err)
	}

	// Verify authorized_keys was created
	authKeysPath := filepath.Join(mountPoint, "root", ".ssh", "authorized_keys")
	content, err := os.ReadFile(authKeysPath)
	if err != nil {
		t.Fatalf("Read authorized_keys: %v", err)
	}

	if !strings.HasPrefix(string(content), "ssh-ed25519 ") {
		t.Error("authorized_keys doesn't contain valid SSH key")
	}
}

func TestSSHKeyManagerInjectWithoutKey(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewSSHKeyManager(tmpDir)

	// Try to inject without generating keys first
	err := manager.InjectSSHKey(filepath.Join(tmpDir, "rootfs"))
	if err == nil {
		t.Error("InjectSSHKey() should error without generated keys")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
