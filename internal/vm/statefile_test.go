package vm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateFileLoadNewState(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	// Initial state should have zero values
	state, err := sf.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if state == nil {
		t.Fatal("Load should return non-nil state for new file")
	}
	if state.BootCount != 0 {
		t.Errorf("initial boot count = %d, want 0", state.BootCount)
	}
	if !state.LastBoot.IsZero() {
		t.Error("initial LastBoot should be zero")
	}
	if state.CleanShutdown {
		t.Error("initial CleanShutdown should be false")
	}
}

func TestStateFilePersistence(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	// Record boot
	if err := sf.RecordBoot(); err != nil {
		t.Fatalf("RecordBoot failed: %v", err)
	}

	// Load and verify
	state, err := sf.Load()
	if err != nil {
		t.Fatalf("Load after boot failed: %v", err)
	}
	if state.BootCount != 1 {
		t.Errorf("boot count = %d, want 1", state.BootCount)
	}
	if state.LastBoot.IsZero() {
		t.Error("last boot time should be set")
	}
	// Boot marks CleanShutdown as false
	if state.CleanShutdown {
		t.Error("CleanShutdown should be false after boot")
	}

	// Record clean shutdown
	if err := sf.RecordShutdown(true); err != nil {
		t.Fatalf("RecordShutdown failed: %v", err)
	}

	state, err = sf.Load()
	if err != nil {
		t.Fatalf("Load after shutdown failed: %v", err)
	}
	if !state.CleanShutdown {
		t.Error("last shutdown should be marked clean")
	}
	if state.LastShutdown.IsZero() {
		t.Error("last shutdown time should be set")
	}
}

func TestStateFileMultipleBoots(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	for i := 1; i <= 5; i++ {
		if err := sf.RecordBoot(); err != nil {
			t.Fatalf("RecordBoot %d failed: %v", i, err)
		}

		state, err := sf.Load()
		if err != nil {
			t.Fatalf("Load after boot %d failed: %v", i, err)
		}
		if state.BootCount != i {
			t.Errorf("after boot %d: boot count = %d, want %d", i, state.BootCount, i)
		}
	}
}

func TestStateFileDirtyShutdown(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	// Boot
	if err := sf.RecordBoot(); err != nil {
		t.Fatalf("RecordBoot failed: %v", err)
	}

	// Dirty shutdown
	if err := sf.RecordShutdown(false); err != nil {
		t.Fatalf("RecordShutdown failed: %v", err)
	}

	state, err := sf.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if state.CleanShutdown {
		t.Error("shutdown should be marked as dirty")
	}
}

func TestStateFilePath(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	expected := filepath.Join(dir, "state.json")
	if sf.Path() != expected {
		t.Errorf("Path() = %q, want %q", sf.Path(), expected)
	}
}

func TestStateFileAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	// Record boot to create file
	if err := sf.RecordBoot(); err != nil {
		t.Fatalf("RecordBoot failed: %v", err)
	}

	// Verify the file exists (not the temp file)
	statePath := sf.Path()
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}

	// Verify temp file doesn't exist (atomic write cleanup)
	tmpPath := statePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful write")
	}
}

func TestStateFileTimestamps(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	before := time.Now()

	if err := sf.RecordBoot(); err != nil {
		t.Fatalf("RecordBoot failed: %v", err)
	}

	after := time.Now()

	state, err := sf.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// LastBoot should be between before and after
	if state.LastBoot.Before(before) || state.LastBoot.After(after) {
		t.Errorf("LastBoot %v not between %v and %v", state.LastBoot, before, after)
	}

	// Record shutdown
	before = time.Now()
	if err := sf.RecordShutdown(true); err != nil {
		t.Fatalf("RecordShutdown failed: %v", err)
	}
	after = time.Now()

	state, err = sf.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// LastShutdown should be between before and after
	if state.LastShutdown.Before(before) || state.LastShutdown.After(after) {
		t.Errorf("LastShutdown %v not between %v and %v", state.LastShutdown, before, after)
	}
}

func TestStateFileSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	sf := NewStateFile(dir)

	// Create state with all fields
	original := &PersistentState{
		LastBoot:      time.Now().Add(-time.Hour),
		LastShutdown:  time.Now().Add(-30 * time.Minute),
		BootCount:     42,
		KernelVersion: "6.1.0-vmterminal",
		DiskSizeMB:    10240,
		CleanShutdown: true,
	}

	if err := sf.Save(original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := sf.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Compare fields
	if loaded.BootCount != original.BootCount {
		t.Errorf("BootCount = %d, want %d", loaded.BootCount, original.BootCount)
	}
	if loaded.KernelVersion != original.KernelVersion {
		t.Errorf("KernelVersion = %q, want %q", loaded.KernelVersion, original.KernelVersion)
	}
	if loaded.DiskSizeMB != original.DiskSizeMB {
		t.Errorf("DiskSizeMB = %d, want %d", loaded.DiskSizeMB, original.DiskSizeMB)
	}
	if loaded.CleanShutdown != original.CleanShutdown {
		t.Errorf("CleanShutdown = %v, want %v", loaded.CleanShutdown, original.CleanShutdown)
	}
	// Time comparison with truncation for JSON round-trip
	if loaded.LastBoot.Unix() != original.LastBoot.Unix() {
		t.Errorf("LastBoot = %v, want %v", loaded.LastBoot, original.LastBoot)
	}
	if loaded.LastShutdown.Unix() != original.LastShutdown.Unix() {
		t.Errorf("LastShutdown = %v, want %v", loaded.LastShutdown, original.LastShutdown)
	}
}

func TestStateFileCreatesDirIfNeeded(t *testing.T) {
	// Use nested directory that doesn't exist
	dir := filepath.Join(t.TempDir(), "nested", "state")
	sf := NewStateFile(dir)

	// Save should create the directory
	state := &PersistentState{BootCount: 1}
	if err := sf.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
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
