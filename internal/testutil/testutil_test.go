package testutil

import (
	"os"
	"testing"

	"github.com/javanstorm/vmterminal/internal/config"
)

func TestTestConfig(t *testing.T) {
	cfg := TestConfig(t)

	// Verify temp directories are set
	if cfg.CacheDir == "" {
		t.Error("CacheDir should not be empty")
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	// Verify directories exist
	if _, err := os.Stat(cfg.CacheDir); os.IsNotExist(err) {
		t.Errorf("CacheDir %s does not exist", cfg.CacheDir)
	}
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		t.Errorf("DataDir %s does not exist", cfg.DataDir)
	}

	// Verify sensible defaults
	if cfg.CPUs <= 0 {
		t.Errorf("CPUs should be positive, got %d", cfg.CPUs)
	}
	if cfg.MemoryMB <= 0 {
		t.Errorf("MemoryMB should be positive, got %d", cfg.MemoryMB)
	}
	if cfg.Provider == nil {
		t.Error("Provider should not be nil")
	}
}

func TestCreateTestDisk(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := tmpDir + "/test.raw"
	sizeMB := int64(10)

	CreateTestDisk(t, diskPath, sizeMB)

	// Verify file exists
	info, err := os.Stat(diskPath)
	if err != nil {
		t.Fatalf("disk file should exist: %v", err)
	}

	// Verify size (sparse file reports full size)
	expectedBytes := sizeMB * 1024 * 1024
	if info.Size() != expectedBytes {
		t.Errorf("disk size = %d, want %d", info.Size(), expectedBytes)
	}
}

func TestCreateTempState(t *testing.T) {
	state := &config.State{
		Distro:        "alpine",
		CPUs:          4,
		MemoryMB:      2048,
		DiskSizeMB:    10240,
		EnableNetwork: true,
	}

	path := CreateTempState(t, state)

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("state file should exist at %s", path)
	}

	// Verify content is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}
	if len(data) == 0 {
		t.Error("state file should not be empty")
	}
}
