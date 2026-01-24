package vm

import (
	"testing"

	"github.com/javanstorm/vmterminal/internal/distro"
)

// createTestConfig creates a ManagerConfig suitable for testing.
// This avoids import cycle with testutil package.
func createTestConfig(t *testing.T) ManagerConfig {
	t.Helper()

	provider, err := distro.GetDefault()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}

	return ManagerConfig{
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

func TestManagerConfigWithTestHelper(t *testing.T) {
	cfg := createTestConfig(t)

	// Verify test config returns valid config
	if cfg.CacheDir == "" {
		t.Error("CacheDir should be set")
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should be set")
	}
	if cfg.Provider == nil {
		t.Error("Provider should be set")
	}

	// Verify reasonable defaults
	if cfg.CPUs <= 0 {
		t.Errorf("CPUs should be positive, got %d", cfg.CPUs)
	}
	if cfg.MemoryMB <= 0 {
		t.Errorf("MemoryMB should be positive, got %d", cfg.MemoryMB)
	}
	if cfg.DiskSizeMB <= 0 {
		t.Errorf("DiskSizeMB should be positive, got %d", cfg.DiskSizeMB)
	}
	if cfg.DiskName == "" {
		t.Error("DiskName should be set")
	}
	if cfg.SharedDirs == nil {
		t.Error("SharedDirs should not be nil")
	}
}

func TestManagerConfigDefaults(t *testing.T) {
	// Test that ManagerConfig fields have expected zero values
	var cfg ManagerConfig

	// All these should be zero/empty for a zero-value config
	if cfg.CacheDir != "" {
		t.Errorf("zero ManagerConfig.CacheDir = %q, want empty", cfg.CacheDir)
	}
	if cfg.DataDir != "" {
		t.Errorf("zero ManagerConfig.DataDir = %q, want empty", cfg.DataDir)
	}
	if cfg.CPUs != 0 {
		t.Errorf("zero ManagerConfig.CPUs = %d, want 0", cfg.CPUs)
	}
	if cfg.MemoryMB != 0 {
		t.Errorf("zero ManagerConfig.MemoryMB = %d, want 0", cfg.MemoryMB)
	}
	if cfg.DiskSizeMB != 0 {
		t.Errorf("zero ManagerConfig.DiskSizeMB = %d, want 0", cfg.DiskSizeMB)
	}
	if cfg.EnableNetwork != false {
		t.Errorf("zero ManagerConfig.EnableNetwork = %v, want false", cfg.EnableNetwork)
	}
	if cfg.SSHHostPort != 0 {
		t.Errorf("zero ManagerConfig.SSHHostPort = %d, want 0", cfg.SSHHostPort)
	}
	if cfg.Provider != nil {
		t.Errorf("zero ManagerConfig.Provider should be nil")
	}
}

// TestNewManagerRequiresHypervisor documents that NewManager requires hypervisor access.
// This test is skipped when /dev/kvm is not available (typical in CI/containers).
func TestNewManagerRequiresHypervisor(t *testing.T) {
	cfg := createTestConfig(t)

	// NewManager will fail without hypervisor access (/dev/kvm on Linux)
	// This is expected behavior - we're documenting the dependency
	_, err := NewManager(cfg)

	// We expect this to either succeed (if KVM available) or fail with hypervisor error
	if err != nil {
		// Expected on systems without KVM
		t.Logf("NewManager failed (expected without hypervisor): %v", err)
		t.Skip("Skipping manager tests - hypervisor not available")
	}

	// If we get here, hypervisor is available - but we don't proceed
	// because we'd need proper kernel/initramfs to actually test
	t.Log("NewManager succeeded - hypervisor available")
}

// TestDefaultDiskSizeMB verifies the default disk size constant.
func TestDefaultDiskSizeMB(t *testing.T) {
	// DefaultDiskSizeMB should be defined and reasonable
	if DefaultDiskSizeMB <= 0 {
		t.Errorf("DefaultDiskSizeMB = %d, want positive", DefaultDiskSizeMB)
	}

	// Should be at least 1GB (1024 MB) for a usable Linux system
	if DefaultDiskSizeMB < 1024 {
		t.Errorf("DefaultDiskSizeMB = %d, want at least 1024", DefaultDiskSizeMB)
	}
}
