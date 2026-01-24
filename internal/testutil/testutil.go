// Package testutil provides common test helpers for VMTerminal tests.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/javanstorm/vmterminal/internal/config"
	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/internal/vm"
)

// TestConfig returns a ManagerConfig suitable for testing.
// Uses t.TempDir() for CacheDir and DataDir, ensuring automatic cleanup.
func TestConfig(t *testing.T) vm.ManagerConfig {
	t.Helper()

	provider, err := distro.GetDefault()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	return vm.ManagerConfig{
		CacheDir:      t.TempDir(),
		DataDir:       t.TempDir(),
		CPUs:          2,
		MemoryMB:      512,
		DiskSizeMB:    1024,
		DiskName:      "test-disk",
		SharedDirs:    make(map[string]string),
		EnableNetwork: false,
		MACAddress:    "",
		SSHHostPort:   0,
		Provider:      provider,
	}
}

// CreateTestDisk creates a sparse disk file at the given path with the specified size.
// The file is created as a sparse file, so it doesn't actually allocate all the space.
func CreateTestDisk(t *testing.T, path string, sizeMB int64) {
	t.Helper()

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test disk at %s: %v", path, err)
	}
	defer f.Close()

	// Create sparse file by truncating to desired size
	sizeBytes := sizeMB * 1024 * 1024
	if err := f.Truncate(sizeBytes); err != nil {
		t.Fatalf("failed to truncate test disk to %d bytes: %v", sizeBytes, err)
	}
}

// CreateTempState writes the given State to a temporary file and returns the path.
// The file is created in a temporary directory that is automatically cleaned up.
func CreateTempState(t *testing.T, state *config.State) string {
	t.Helper()

	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	return statePath
}
