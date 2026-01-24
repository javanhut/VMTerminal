package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDisk(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

	// Create new disk
	path, err := im.EnsureDisk("test", 100) // 100MB
	if err != nil {
		t.Fatalf("EnsureDisk failed: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("disk file not created: %v", err)
	}

	// Sparse file - logical size should match requested size
	expectedSize := int64(100 * 1024 * 1024)
	if info.Size() != expectedSize {
		t.Errorf("disk size = %d, want %d", info.Size(), expectedSize)
	}

	// Second call returns same path without error
	path2, err := im.EnsureDisk("test", 100)
	if err != nil {
		t.Fatalf("EnsureDisk second call failed: %v", err)
	}
	if path != path2 {
		t.Error("EnsureDisk should return same path for existing disk")
	}

	// Verify disk exists in expected location
	expectedPath := filepath.Join(dir, "test.raw")
	if path != expectedPath {
		t.Errorf("disk path = %q, want %q", path, expectedPath)
	}
}

func TestEnsureDiskDifferentSizes(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

	tests := []struct {
		name   string
		sizeMB int64
	}{
		{"small", 10},
		{"medium", 100},
		{"large", 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := im.EnsureDisk(tt.name, tt.sizeMB)
			if err != nil {
				t.Fatalf("EnsureDisk(%d) failed: %v", tt.sizeMB, err)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("disk file not created: %v", err)
			}

			expectedSize := tt.sizeMB * 1024 * 1024
			if info.Size() != expectedSize {
				t.Errorf("disk size = %d, want %d", info.Size(), expectedSize)
			}
		})
	}
}

func TestDiskExists(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

	// Should not exist initially
	if im.DiskExists("nonexistent") {
		t.Error("DiskExists should return false for non-existent disk")
	}

	// Create disk
	_, err := im.EnsureDisk("test", 10)
	if err != nil {
		t.Fatalf("EnsureDisk failed: %v", err)
	}

	// Should exist now
	if !im.DiskExists("test") {
		t.Error("DiskExists should return true for existing disk")
	}
}

func TestDiskPath(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

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
			path := im.DiskPath(tt.name)
			if path != tt.want {
				t.Errorf("DiskPath(%q) = %q, want %q", tt.name, path, tt.want)
			}
		})
	}
}

func TestDeleteDisk(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

	// Create disk first
	_, err := im.EnsureDisk("test", 10)
	if err != nil {
		t.Fatalf("EnsureDisk failed: %v", err)
	}

	// Verify it exists
	if !im.DiskExists("test") {
		t.Fatal("disk should exist after creation")
	}

	// Delete disk
	if err := im.DeleteDisk("test"); err != nil {
		t.Fatalf("DeleteDisk failed: %v", err)
	}

	// Verify it's gone
	if im.DiskExists("test") {
		t.Error("disk should not exist after deletion")
	}
}

func TestDeleteDiskNonExistent(t *testing.T) {
	dir := t.TempDir()
	im := NewImageManager(dir)

	// Delete non-existent disk should not error
	if err := im.DeleteDisk("nonexistent"); err != nil {
		t.Errorf("DeleteDisk should not error for non-existent disk: %v", err)
	}
}

func TestImageManagerCreatesDirIfNeeded(t *testing.T) {
	// Use a nested directory that doesn't exist
	dir := filepath.Join(t.TempDir(), "nested", "data")
	im := NewImageManager(dir)

	// EnsureDisk should create the directory
	_, err := im.EnsureDisk("test", 10)
	if err != nil {
		t.Fatalf("EnsureDisk failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("path should be a directory")
	}
}
