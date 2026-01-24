package vm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotManagerCreateAndList(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	// Create a fake disk to snapshot
	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	if err := os.MkdirAll(diskDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	diskPath := filepath.Join(diskDir, "disk.raw")
	if err := os.WriteFile(diskPath, []byte("test disk content"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create snapshot
	err := mgr.CreateSnapshot(vmName, "snap1", "test snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	// List snapshots
	snapshots, err := mgr.ListSnapshots(vmName)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].Name != "snap1" {
		t.Errorf("wrong name: %s", snapshots[0].Name)
	}
	if snapshots[0].Checksum == "" {
		t.Error("checksum should be set")
	}
	if snapshots[0].DiskSize != 17 { // len("test disk content")
		t.Errorf("wrong disk size: %d", snapshots[0].DiskSize)
	}
}

func TestSnapshotManagerDuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// First snapshot should succeed
	err := mgr.CreateSnapshot(vmName, "snap1", "first")
	if err != nil {
		t.Fatalf("first CreateSnapshot: %v", err)
	}

	// Duplicate name should fail
	err = mgr.CreateSnapshot(vmName, "snap1", "duplicate")
	if err == nil {
		t.Error("expected error for duplicate snapshot name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("wrong error message: %v", err)
	}
}

func TestSnapshotManagerRestore(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")

	// Create original disk
	originalContent := []byte("original disk content")
	os.WriteFile(diskPath, originalContent, 0644)

	// Create snapshot
	err := mgr.CreateSnapshot(vmName, "snap1", "test")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	// Modify disk
	os.WriteFile(diskPath, []byte("modified content"), 0644)

	// Restore snapshot
	err = mgr.RestoreSnapshot(vmName, "snap1")
	if err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	// Verify disk content is restored
	restored, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(restored) != string(originalContent) {
		t.Errorf("content not restored: got %q, want %q", string(restored), string(originalContent))
	}
}

func TestSnapshotManagerRestoreChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create snapshot
	err := mgr.CreateSnapshot(vmName, "snap1", "test")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	// Corrupt the snapshot file
	snapPath := filepath.Join(tmpDir, "data", vmName, "snapshots", "snap1.raw.gz")
	snapData, _ := os.ReadFile(snapPath)
	// Modify a byte to corrupt the checksum
	if len(snapData) > 10 {
		snapData[10] ^= 0xFF
	}
	os.WriteFile(snapPath, snapData, 0644)

	// Restore should fail due to checksum mismatch
	err = mgr.RestoreSnapshot(vmName, "snap1")
	if err == nil {
		t.Error("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerDelete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create snapshot
	mgr.CreateSnapshot(vmName, "snap1", "test")

	// Delete snapshot
	err := mgr.DeleteSnapshot(vmName, "snap1")
	if err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}

	// Verify it's gone
	snapshots, _ := mgr.ListSnapshots(vmName)
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}

	// Verify file is deleted
	snapPath := filepath.Join(tmpDir, "data", vmName, "snapshots", "snap1.raw.gz")
	if _, err := os.Stat(snapPath); !os.IsNotExist(err) {
		t.Error("snapshot file should be deleted")
	}
}

func TestSnapshotManagerVerify(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create snapshot
	mgr.CreateSnapshot(vmName, "snap1", "test")

	// Verify should succeed
	err := mgr.VerifySnapshot(vmName, "snap1")
	if err != nil {
		t.Errorf("VerifySnapshot should pass: %v", err)
	}

	// Corrupt the file
	snapPath := filepath.Join(tmpDir, "data", vmName, "snapshots", "snap1.raw.gz")
	snapData, _ := os.ReadFile(snapPath)
	if len(snapData) > 10 {
		snapData[10] ^= 0xFF
	}
	os.WriteFile(snapPath, snapData, 0644)

	// Verify should fail
	err = mgr.VerifySnapshot(vmName, "snap1")
	if err == nil {
		t.Error("VerifySnapshot should fail on corrupt file")
	}
}

func TestSnapshotManagerLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	// No snapshots.json exists
	snapshots, err := mgr.ListSnapshots("nonexistent-vm")
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected empty list, got %d", len(snapshots))
	}
}

func TestSnapshotManagerAtomicCreate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create snapshot
	err := mgr.CreateSnapshot(vmName, "snap1", "test")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	// Verify no .tmp files remain
	snapshotsDir := filepath.Join(tmpDir, "data", vmName, "snapshots")
	tmpFiles, _ := filepath.Glob(filepath.Join(snapshotsDir, "*.tmp"))
	if len(tmpFiles) > 0 {
		t.Errorf("temp files should be cleaned up: %v", tmpFiles)
	}

	// Verify final file exists
	snapPath := filepath.Join(snapshotsDir, "snap1.raw.gz")
	if _, err := os.Stat(snapPath); err != nil {
		t.Errorf("snapshot file should exist: %v", err)
	}
}

func TestSnapshotManagerCleanupPartial(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	snapshotsDir := filepath.Join(diskDir, "snapshots")
	os.MkdirAll(snapshotsDir, 0755)

	// Create fake partial files
	tmpFile := filepath.Join(snapshotsDir, "snap1.raw.gz.tmp")
	restoringFile := filepath.Join(diskDir, "disk.raw.restoring")
	metaTmpFile := filepath.Join(diskDir, "snapshots.json.tmp")

	os.WriteFile(tmpFile, []byte("partial"), 0644)
	os.WriteFile(restoringFile, []byte("restoring"), 0644)
	os.WriteFile(metaTmpFile, []byte("meta"), 0644)

	// Verify files exist
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("tmp file not created")
	}

	// Cleanup
	err := mgr.CleanupPartial(vmName)
	if err != nil {
		t.Fatalf("CleanupPartial: %v", err)
	}

	// Verify all removed
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("tmp file should be removed")
	}
	if _, err := os.Stat(restoringFile); !os.IsNotExist(err) {
		t.Error("restoring file should be removed")
	}
	if _, err := os.Stat(metaTmpFile); !os.IsNotExist(err) {
		t.Error("meta tmp file should be removed")
	}
}

func TestSnapshotManagerHasPartialFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	snapshotsDir := filepath.Join(diskDir, "snapshots")
	os.MkdirAll(snapshotsDir, 0755)

	// Initially no partial files
	if mgr.HasPartialFiles(vmName) {
		t.Error("should not have partial files initially")
	}

	// Create a .tmp file
	tmpFile := filepath.Join(snapshotsDir, "snap1.raw.gz.tmp")
	os.WriteFile(tmpFile, []byte("partial"), 0644)

	// Should detect partial files
	if !mgr.HasPartialFiles(vmName) {
		t.Error("should detect .tmp file")
	}

	// Cleanup
	mgr.CleanupPartial(vmName)

	// Should no longer have partial files
	if mgr.HasPartialFiles(vmName) {
		t.Error("should not have partial files after cleanup")
	}

	// Test .restoring detection
	restoringFile := filepath.Join(diskDir, "disk.raw.restoring")
	os.WriteFile(restoringFile, []byte("restoring"), 0644)

	if !mgr.HasPartialFiles(vmName) {
		t.Error("should detect .restoring file")
	}
}

func TestSnapshotManagerGetSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create snapshot
	mgr.CreateSnapshot(vmName, "snap1", "test description")

	// Get existing snapshot
	snap, err := mgr.GetSnapshot(vmName, "snap1")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snap.Name != "snap1" {
		t.Errorf("wrong name: %s", snap.Name)
	}
	if snap.Description != "test description" {
		t.Errorf("wrong description: %s", snap.Description)
	}

	// Get non-existent snapshot
	_, err = mgr.GetSnapshot(vmName, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent snapshot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	before := time.Now().Add(-time.Second)
	mgr.CreateSnapshot(vmName, "snap1", "test")
	after := time.Now().Add(time.Second)

	snap, _ := mgr.GetSnapshot(vmName, "snap1")
	if snap.CreatedAt.Before(before) || snap.CreatedAt.After(after) {
		t.Errorf("timestamp out of range: %v (expected between %v and %v)",
			snap.CreatedAt, before, after)
	}
}

func TestSnapshotManagerSnapshotFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk content for size check"), 0644)

	mgr.CreateSnapshot(vmName, "snap1", "test")

	size, err := mgr.SnapshotFileSize(vmName, "snap1")
	if err != nil {
		t.Fatalf("SnapshotFileSize: %v", err)
	}
	if size == 0 {
		t.Error("snapshot file size should be > 0")
	}
}

func TestSnapshotManagerNoDisk(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	// Try to create snapshot without disk
	err := mgr.CreateSnapshot("no-disk-vm", "snap1", "test")
	if err == nil {
		t.Error("expected error when disk doesn't exist")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	err := mgr.DeleteSnapshot("test-vm", "nonexistent")
	if err == nil {
		t.Error("expected error deleting non-existent snapshot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerRestoreNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	err := mgr.RestoreSnapshot("test-vm", "nonexistent")
	if err == nil {
		t.Error("expected error restoring non-existent snapshot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerVerifyNoChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)

	// Manually create a snapshot entry without checksum (simulates old format)
	data := &SnapshotData{
		Snapshots: []SnapshotEntry{
			{
				Name:        "old-snap",
				VMName:      vmName,
				Description: "old snapshot",
				CreatedAt:   time.Now(),
				DiskSize:    100,
				Checksum:    "", // Empty checksum
			},
		},
	}
	mgr.Save(vmName, data)

	// Verify should fail with meaningful error
	err := mgr.VerifySnapshot(vmName, "old-snap")
	if err == nil {
		t.Error("expected error for snapshot without checksum")
	}
	if !strings.Contains(err.Error(), "no checksum") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestSnapshotManagerMultipleSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewSnapshotManager(tmpDir)

	vmName := "test-vm"
	diskDir := filepath.Join(tmpDir, "data", vmName)
	os.MkdirAll(diskDir, 0755)
	diskPath := filepath.Join(diskDir, "disk.raw")
	os.WriteFile(diskPath, []byte("test disk"), 0644)

	// Create multiple snapshots
	for i := 1; i <= 3; i++ {
		name := "snap" + string(rune('0'+i))
		err := mgr.CreateSnapshot(vmName, name, "test "+name)
		if err != nil {
			t.Fatalf("CreateSnapshot %s: %v", name, err)
		}
	}

	// Verify all exist
	snapshots, _ := mgr.ListSnapshots(vmName)
	if len(snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(snapshots))
	}

	// Delete middle one
	mgr.DeleteSnapshot(vmName, "snap2")

	snapshots, _ = mgr.ListSnapshots(vmName)
	if len(snapshots) != 2 {
		t.Errorf("expected 2 snapshots after delete, got %d", len(snapshots))
	}

	// Verify correct ones remain
	for _, snap := range snapshots {
		if snap.Name == "snap2" {
			t.Error("snap2 should be deleted")
		}
	}
}
