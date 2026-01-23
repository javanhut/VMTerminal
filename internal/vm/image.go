package vm

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultDiskSizeMB is the default disk image size (10GB sparse).
	DefaultDiskSizeMB = 10 * 1024
)

// ImageManager handles disk image creation and management.
type ImageManager struct {
	dataDir string
}

// NewImageManager creates an image manager with the given data directory.
func NewImageManager(dataDir string) *ImageManager {
	return &ImageManager{dataDir: dataDir}
}

// EnsureDisk creates a disk image if it doesn't exist.
// Returns the path to the disk image.
func (m *ImageManager) EnsureDisk(name string, sizeMB int64) (string, error) {
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	path := filepath.Join(m.dataDir, name+".raw")

	if _, err := os.Stat(path); err == nil {
		return path, nil // Already exists
	}

	if err := m.createSparseImage(path, sizeMB); err != nil {
		return "", fmt.Errorf("create disk image: %w", err)
	}

	return path, nil
}

// DiskPath returns the path to a named disk image.
func (m *ImageManager) DiskPath(name string) string {
	return filepath.Join(m.dataDir, name+".raw")
}

// DiskExists checks if a disk image exists.
func (m *ImageManager) DiskExists(name string) bool {
	_, err := os.Stat(m.DiskPath(name))
	return err == nil
}

// DeleteDisk removes a disk image.
func (m *ImageManager) DeleteDisk(name string) error {
	path := m.DiskPath(name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete disk: %w", err)
	}
	return nil
}

func (m *ImageManager) createSparseImage(path string, sizeMB int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Truncate creates a sparse file on Linux/macOS
	return f.Truncate(sizeMB * 1024 * 1024)
}
