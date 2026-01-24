package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestQuietMode(t *testing.T) {
	// Save original state
	origQuiet := quietMode
	defer func() { quietMode = origQuiet }()

	// Default is not quiet
	if quietMode {
		t.Error("quietMode should be false by default")
	}

	SetQuietMode(true)
	if !quietMode {
		t.Error("SetQuietMode(true) should enable quiet mode")
	}

	SetQuietMode(false)
	if quietMode {
		t.Error("SetQuietMode(false) should disable quiet mode")
	}
}

func TestWritePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	vmName := "test-vm"
	dataDir := filepath.Join(tmpDir, "data", vmName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	err := writePIDFile(tmpDir, vmName)
	if err != nil {
		t.Fatalf("writePIDFile: %v", err)
	}

	pidFile := filepath.Join(tmpDir, "data", vmName, "vm.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		t.Fatalf("parse PID: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("PID mismatch: got %d, want %d", pid, os.Getpid())
	}

	cleanupPIDFile(tmpDir, vmName)
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after cleanup")
	}
}

func TestIsVMRunning(t *testing.T) {
	tmpDir := t.TempDir()
	vmName := "test-vm"
	dataDir := filepath.Join(tmpDir, "data", vmName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// No PID file - not running
	running, _ := isVMRunning(tmpDir, vmName)
	if running {
		t.Error("VM should not be running without PID file")
	}

	// Write PID file with current process (which is running)
	err := writePIDFile(tmpDir, vmName)
	if err != nil {
		t.Fatalf("writePIDFile: %v", err)
	}

	running, pid := isVMRunning(tmpDir, vmName)
	if !running {
		t.Error("VM should be detected as running")
	}
	if pid != os.Getpid() {
		t.Errorf("wrong PID: got %d, want %d", pid, os.Getpid())
	}

	// Clean up
	cleanupPIDFile(tmpDir, vmName)
}

func TestIsVMRunningStalePID(t *testing.T) {
	tmpDir := t.TempDir()
	vmName := "test-vm"
	dataDir := filepath.Join(tmpDir, "data", vmName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a PID file with a non-existent process (very high PID)
	pidFile := filepath.Join(tmpDir, "data", vmName, "vm.pid")
	// Use a PID that's unlikely to exist (max PID is usually 32768 or 4194304)
	os.WriteFile(pidFile, []byte("999999999"), 0644)

	// Should not be detected as running (process doesn't exist)
	running, _ := isVMRunning(tmpDir, vmName)
	if running {
		t.Error("stale PID file should not be detected as running")
	}
}

func TestIsVMRunningInvalidPID(t *testing.T) {
	tmpDir := t.TempDir()
	vmName := "test-vm"
	dataDir := filepath.Join(tmpDir, "data", vmName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a PID file with invalid content
	pidFile := filepath.Join(tmpDir, "data", vmName, "vm.pid")
	os.WriteFile(pidFile, []byte("not-a-number"), 0644)

	// Should not be detected as running (can't parse PID)
	running, _ := isVMRunning(tmpDir, vmName)
	if running {
		t.Error("invalid PID file should not be detected as running")
	}
}
