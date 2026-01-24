package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSetupStateNewDisk(t *testing.T) {
	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	// Check state for non-existent disk
	state, err := rm.CheckSetupState("nonexistent")
	if err != nil {
		t.Fatalf("CheckSetupState failed: %v", err)
	}

	if state.DiskExists {
		t.Error("DiskExists should be false for non-existent disk")
	}
	if state.DiskFormatted {
		t.Error("DiskFormatted should be false for non-existent disk")
	}
	if state.RootfsExtracted {
		t.Error("RootfsExtracted should be false for non-existent disk")
	}
}

func TestCheckSetupStateWithDiskFile(t *testing.T) {
	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	// Create an empty disk file
	diskPath := filepath.Join(dir, "test.raw")
	if err := os.WriteFile(diskPath, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("create disk file: %v", err)
	}

	// Check state - disk exists but not formatted
	state, err := rm.CheckSetupState("test")
	if err != nil {
		t.Fatalf("CheckSetupState failed: %v", err)
	}

	if !state.DiskExists {
		t.Error("DiskExists should be true for existing disk file")
	}
	// Without actual formatting (blkid won't find filesystem in empty file)
	// DiskFormatted should be false
	if state.DiskFormatted {
		t.Error("DiskFormatted should be false for empty disk file")
	}
}

func TestRootfsManagerDiskPath(t *testing.T) {
	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	tests := []struct {
		name string
		want string
	}{
		{"test", filepath.Join(dir, "test.raw")},
		{"alpine", filepath.Join(dir, "alpine.raw")},
		{"my-vm", filepath.Join(dir, "my-vm.raw")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := rm.DiskPath(tt.name)
			if path != tt.want {
				t.Errorf("DiskPath(%q) = %q, want %q", tt.name, path, tt.want)
			}
		})
	}
}

func TestFormatDisk(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	// Create a disk file first
	diskPath := filepath.Join(dir, "test.raw")
	f, err := os.Create(diskPath)
	if err != nil {
		t.Fatalf("create disk file: %v", err)
	}
	// Create a 100MB sparse file
	if err := f.Truncate(100 * 1024 * 1024); err != nil {
		f.Close()
		t.Fatalf("truncate disk file: %v", err)
	}
	f.Close()

	// Format with ext4
	if err := rm.FormatDisk("test", "ext4"); err != nil {
		t.Fatalf("FormatDisk failed: %v", err)
	}

	// Check state should now show formatted
	state, err := rm.CheckSetupState("test")
	if err != nil {
		t.Fatalf("CheckSetupState failed: %v", err)
	}

	if !state.DiskFormatted {
		t.Error("DiskFormatted should be true after format")
	}
	if state.FSType != "ext4" {
		t.Errorf("FSType = %q, want ext4", state.FSType)
	}
}

func TestFormatDiskNonExistent(t *testing.T) {
	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	// Format non-existent disk should fail
	err := rm.FormatDisk("nonexistent", "ext4")
	if err == nil {
		t.Error("FormatDisk should fail for non-existent disk")
	}
}

func TestFormatDiskUnsupportedFS(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	dir := t.TempDir()
	rm := NewRootfsManager(dir)

	// Create a disk file first
	diskPath := filepath.Join(dir, "test.raw")
	if err := os.WriteFile(diskPath, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("create disk file: %v", err)
	}

	// Format with unsupported filesystem should fail
	err := rm.FormatDisk("test", "ntfs")
	if err == nil {
		t.Error("FormatDisk should fail for unsupported filesystem")
	}
}

func TestExtractRootfs(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}
	// This test would require a formatted and mounted disk
	// Skip for now as it requires complex setup
	t.Skip("requires formatted disk and mount capability")
}

func TestSetupStateFields(t *testing.T) {
	// Test that SetupState has correct zero values
	state := &SetupState{}

	if state.DiskExists {
		t.Error("zero SetupState.DiskExists should be false")
	}
	if state.DiskFormatted {
		t.Error("zero SetupState.DiskFormatted should be false")
	}
	if state.RootfsExtracted {
		t.Error("zero SetupState.RootfsExtracted should be false")
	}
	if state.FSType != "" {
		t.Error("zero SetupState.FSType should be empty")
	}
}
