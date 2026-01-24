package vm

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// SnapshotEntry represents a single VM snapshot.
type SnapshotEntry struct {
	Name        string    `json:"name"`
	VMName      string    `json:"vm_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	DiskSize    int64     `json:"disk_size"` // Original uncompressed size in bytes
	Checksum    string    `json:"checksum"`  // SHA256 of compressed file
}

// SnapshotData holds all snapshots for a VM.
type SnapshotData struct {
	Snapshots []SnapshotEntry `json:"snapshots"`
}

// SnapshotManager handles VM disk snapshots.
type SnapshotManager struct {
	baseDir string // ~/.vmterminal
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(baseDir string) *SnapshotManager {
	return &SnapshotManager{baseDir: baseDir}
}

// snapshotsDir returns the snapshots directory for a VM.
func (m *SnapshotManager) snapshotsDir(vmName string) string {
	return filepath.Join(m.baseDir, "data", vmName, "snapshots")
}

// snapshotsFile returns the snapshots metadata file path for a VM.
func (m *SnapshotManager) snapshotsFile(vmName string) string {
	return filepath.Join(m.baseDir, "data", vmName, "snapshots.json")
}

// diskPath returns the disk image path for a VM.
func (m *SnapshotManager) diskPath(vmName string) string {
	return filepath.Join(m.baseDir, "data", vmName, "disk.raw")
}

// snapshotPath returns the path to a specific snapshot file.
func (m *SnapshotManager) snapshotPath(vmName, snapshotName string) string {
	return filepath.Join(m.snapshotsDir(vmName), snapshotName+".raw.gz")
}

// Load reads the snapshot metadata from disk.
func (m *SnapshotManager) Load(vmName string) (*SnapshotData, error) {
	data, err := os.ReadFile(m.snapshotsFile(vmName))
	if err != nil {
		if os.IsNotExist(err) {
			return &SnapshotData{Snapshots: []SnapshotEntry{}}, nil
		}
		return nil, fmt.Errorf("read snapshots: %w", err)
	}

	var snapshots SnapshotData
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("parse snapshots: %w", err)
	}

	return &snapshots, nil
}

// Save writes the snapshot metadata to disk atomically.
// Uses temp file + rename to prevent corruption on interrupted writes.
func (m *SnapshotManager) Save(vmName string, data *SnapshotData) error {
	// Ensure data directory exists
	dataDir := filepath.Join(m.baseDir, "data", vmName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshots: %w", err)
	}

	// Write atomically: temp file + rename
	finalPath := m.snapshotsFile(vmName)
	tmpPath := finalPath + ".tmp"

	if err := os.WriteFile(tmpPath, jsonData, 0644); err != nil {
		return fmt.Errorf("write snapshots temp: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on rename failure
		return fmt.Errorf("rename snapshots: %w", err)
	}

	return nil
}

// CreateSnapshot creates a new snapshot by compressing the VM disk.
// Uses atomic temp file + rename to prevent corruption on interrupted writes.
func (m *SnapshotManager) CreateSnapshot(vmName, snapshotName, description string) error {
	// Clean up any previous partial operations
	m.CleanupPartial(vmName)

	// Check if disk exists
	diskPath := m.diskPath(vmName)
	diskInfo, err := os.Stat(diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("VM disk not found: %s", diskPath)
		}
		return fmt.Errorf("stat disk: %w", err)
	}

	// Load existing snapshots
	data, err := m.Load(vmName)
	if err != nil {
		return err
	}

	// Check for duplicate name
	for _, snap := range data.Snapshots {
		if snap.Name == snapshotName {
			return fmt.Errorf("snapshot '%s' already exists", snapshotName)
		}
	}

	// Create snapshots directory
	snapshotsDir := m.snapshotsDir(vmName)
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return fmt.Errorf("create snapshots dir: %w", err)
	}

	// Open source disk
	srcFile, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("open disk: %w", err)
	}
	defer srcFile.Close()

	// Create compressed snapshot file using temp file for atomic operation
	snapPath := m.snapshotPath(vmName, snapshotName)
	tmpPath := snapPath + ".tmp"
	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create snapshot file: %w", err)
	}
	defer dstFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// Copy disk to gzip
	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("compress disk: %w", err)
	}

	// Ensure gzip is flushed
	if err := gzWriter.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalize compression: %w", err)
	}

	// Close destination file before rename
	if err := dstFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close snapshot file: %w", err)
	}

	// Atomic rename: temp file -> final path
	if err := os.Rename(tmpPath, snapPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalize snapshot: %w", err)
	}

	// Compute checksum of the final snapshot file
	checksum, err := m.computeChecksum(snapPath)
	if err != nil {
		os.Remove(snapPath)
		return fmt.Errorf("compute checksum: %w", err)
	}

	// Create snapshot entry with checksum
	entry := SnapshotEntry{
		Name:        snapshotName,
		VMName:      vmName,
		Description: description,
		CreatedAt:   time.Now(),
		DiskSize:    diskInfo.Size(),
		Checksum:    checksum,
	}

	data.Snapshots = append(data.Snapshots, entry)

	// Save metadata
	if err := m.Save(vmName, data); err != nil {
		os.Remove(snapPath)
		return err
	}

	return nil
}

// ListSnapshots returns all snapshots for a VM.
func (m *SnapshotManager) ListSnapshots(vmName string) ([]SnapshotEntry, error) {
	data, err := m.Load(vmName)
	if err != nil {
		return nil, err
	}
	return data.Snapshots, nil
}

// GetSnapshot returns a specific snapshot.
func (m *SnapshotManager) GetSnapshot(vmName, snapshotName string) (*SnapshotEntry, error) {
	data, err := m.Load(vmName)
	if err != nil {
		return nil, err
	}

	for _, snap := range data.Snapshots {
		if snap.Name == snapshotName {
			return &snap, nil
		}
	}

	return nil, fmt.Errorf("snapshot '%s' not found", snapshotName)
}

// RestoreSnapshot restores a VM disk from a snapshot.
// WARNING: This overwrites the current disk! VM must be stopped.
// Verifies checksum before restoration to detect corruption.
func (m *SnapshotManager) RestoreSnapshot(vmName, snapshotName string) error {
	// Clean up any previous partial operations
	m.CleanupPartial(vmName)

	// Verify snapshot exists
	snap, err := m.GetSnapshot(vmName, snapshotName)
	if err != nil {
		return err
	}

	snapPath := m.snapshotPath(vmName, snapshotName)
	diskPath := m.diskPath(vmName)

	// Verify checksum before restore (if checksum exists)
	if snap.Checksum != "" {
		checksum, err := m.computeChecksum(snapPath)
		if err != nil {
			return fmt.Errorf("verify checksum: %w", err)
		}
		if checksum != snap.Checksum {
			return fmt.Errorf("snapshot corrupted: checksum mismatch")
		}
	}

	// Open compressed snapshot
	srcFile, err := os.Open(snapPath)
	if err != nil {
		return fmt.Errorf("open snapshot: %w", err)
	}
	defer srcFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gzReader.Close()

	// Create temporary file for restored disk
	tmpPath := diskPath + ".restoring"
	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp disk: %w", err)
	}
	defer dstFile.Close()

	// Decompress to disk
	if _, err := io.Copy(dstFile, gzReader); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("decompress snapshot: %w", err)
	}

	// Close files before rename
	dstFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, diskPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace disk: %w", err)
	}

	return nil
}

// DeleteSnapshot removes a snapshot.
func (m *SnapshotManager) DeleteSnapshot(vmName, snapshotName string) error {
	data, err := m.Load(vmName)
	if err != nil {
		return err
	}

	// Find and remove snapshot entry
	found := false
	newSnapshots := make([]SnapshotEntry, 0, len(data.Snapshots))
	for _, snap := range data.Snapshots {
		if snap.Name == snapshotName {
			found = true
		} else {
			newSnapshots = append(newSnapshots, snap)
		}
	}

	if !found {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	// Delete snapshot file
	snapPath := m.snapshotPath(vmName, snapshotName)
	if err := os.Remove(snapPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete snapshot file: %w", err)
	}

	// Update metadata
	data.Snapshots = newSnapshots
	return m.Save(vmName, data)
}

// SnapshotFileSize returns the compressed size of a snapshot file.
func (m *SnapshotManager) SnapshotFileSize(vmName, snapshotName string) (int64, error) {
	snapPath := m.snapshotPath(vmName, snapshotName)
	info, err := os.Stat(snapPath)
	if err != nil {
		return 0, fmt.Errorf("stat snapshot: %w", err)
	}
	return info.Size(), nil
}

// VerifySnapshot verifies the integrity of a snapshot by checking its checksum.
// Returns nil if the snapshot is valid, or an error describing the issue.
func (m *SnapshotManager) VerifySnapshot(vmName, snapshotName string) error {
	snap, err := m.GetSnapshot(vmName, snapshotName)
	if err != nil {
		return err
	}

	if snap.Checksum == "" {
		return fmt.Errorf("snapshot has no checksum (created before checksum support)")
	}

	snapPath := m.snapshotPath(vmName, snapshotName)
	checksum, err := m.computeChecksum(snapPath)
	if err != nil {
		return fmt.Errorf("compute checksum: %w", err)
	}

	if checksum != snap.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", snap.Checksum, checksum)
	}

	return nil
}

// computeChecksum calculates the SHA256 checksum of a file.
func (m *SnapshotManager) computeChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CleanupPartial removes any partial/interrupted snapshot files.
// Call on startup or before operations to ensure clean state.
func (m *SnapshotManager) CleanupPartial(vmName string) error {
	snapshotsDir := m.snapshotsDir(vmName)

	// Clean up .tmp files (interrupted creates)
	tmpFiles, _ := filepath.Glob(filepath.Join(snapshotsDir, "*.tmp"))
	for _, f := range tmpFiles {
		os.Remove(f)
	}

	// Clean up .restoring files (interrupted restores)
	dataDir := filepath.Join(m.baseDir, "data", vmName)
	restoringFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.restoring"))
	for _, f := range restoringFiles {
		os.Remove(f)
	}

	// Clean up orphaned metadata temp files
	metaTmp := m.snapshotsFile(vmName) + ".tmp"
	os.Remove(metaTmp)

	return nil
}

// HasPartialFiles returns true if there are leftover temp files from interrupted operations.
func (m *SnapshotManager) HasPartialFiles(vmName string) bool {
	snapshotsDir := m.snapshotsDir(vmName)
	dataDir := filepath.Join(m.baseDir, "data", vmName)

	tmpFiles, _ := filepath.Glob(filepath.Join(snapshotsDir, "*.tmp"))
	if len(tmpFiles) > 0 {
		return true
	}

	restoringFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.restoring"))
	if len(restoringFiles) > 0 {
		return true
	}

	metaTmp := m.snapshotsFile(vmName) + ".tmp"
	if _, err := os.Stat(metaTmp); err == nil {
		return true
	}

	return false
}
