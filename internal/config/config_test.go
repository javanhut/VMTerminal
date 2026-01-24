package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultState(t *testing.T) {
	state := DefaultState()

	if state == nil {
		t.Fatal("DefaultState should not return nil")
	}

	// Check CPU count is positive (should be runtime.NumCPU())
	if state.CPUs <= 0 {
		t.Errorf("CPUs should be positive, got %d", state.CPUs)
	}
	if state.CPUs != runtime.NumCPU() {
		t.Errorf("CPUs should be %d (runtime.NumCPU()), got %d", runtime.NumCPU(), state.CPUs)
	}

	// Check memory default
	if state.MemoryMB != 2048 {
		t.Errorf("MemoryMB should be 2048, got %d", state.MemoryMB)
	}

	// Check disk size default (10GB)
	if state.DiskSizeMB != 10240 {
		t.Errorf("DiskSizeMB should be 10240, got %d", state.DiskSizeMB)
	}

	// Check network enabled by default
	if !state.EnableNetwork {
		t.Error("EnableNetwork should be true by default")
	}

	// Check default distro is Alpine
	if state.Distro != "alpine" {
		t.Errorf("Distro should be 'alpine', got %q", state.Distro)
	}

	// Check SSH host port default
	if state.SSHHostPort != 2222 {
		t.Errorf("SSHHostPort should be 2222, got %d", state.SSHHostPort)
	}

	// Check default terminal is false
	if state.IsDefaultTerminal {
		t.Error("IsDefaultTerminal should be false by default")
	}
}

func TestStateJSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		state *State
	}{
		{
			name: "minimal state",
			state: &State{
				Distro:   "alpine",
				CPUs:     2,
				MemoryMB: 1024,
			},
		},
		{
			name: "full state",
			state: &State{
				Distro:            "ubuntu",
				CPUs:              4,
				MemoryMB:          4096,
				DiskSizeMB:        20480,
				SharedDirs:        []string{"/home/user", "/tmp"},
				EnableNetwork:     true,
				MACAddress:        "00:11:22:33:44:55",
				SSHHostPort:       2222,
				IsDefaultTerminal: true,
			},
		},
		{
			name:  "default state",
			state: DefaultState(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			data, err := json.Marshal(tt.state)
			if err != nil {
				t.Fatalf("failed to marshal state: %v", err)
			}

			// Deserialize back
			var loaded State
			if err := json.Unmarshal(data, &loaded); err != nil {
				t.Fatalf("failed to unmarshal state: %v", err)
			}

			// Compare key fields
			if loaded.Distro != tt.state.Distro {
				t.Errorf("Distro mismatch: got %q, want %q", loaded.Distro, tt.state.Distro)
			}
			if loaded.CPUs != tt.state.CPUs {
				t.Errorf("CPUs mismatch: got %d, want %d", loaded.CPUs, tt.state.CPUs)
			}
			if loaded.MemoryMB != tt.state.MemoryMB {
				t.Errorf("MemoryMB mismatch: got %d, want %d", loaded.MemoryMB, tt.state.MemoryMB)
			}
			if loaded.DiskSizeMB != tt.state.DiskSizeMB {
				t.Errorf("DiskSizeMB mismatch: got %d, want %d", loaded.DiskSizeMB, tt.state.DiskSizeMB)
			}
			if loaded.EnableNetwork != tt.state.EnableNetwork {
				t.Errorf("EnableNetwork mismatch: got %v, want %v", loaded.EnableNetwork, tt.state.EnableNetwork)
			}
			if loaded.SSHHostPort != tt.state.SSHHostPort {
				t.Errorf("SSHHostPort mismatch: got %d, want %d", loaded.SSHHostPort, tt.state.SSHHostPort)
			}
			if loaded.IsDefaultTerminal != tt.state.IsDefaultTerminal {
				t.Errorf("IsDefaultTerminal mismatch: got %v, want %v", loaded.IsDefaultTerminal, tt.state.IsDefaultTerminal)
			}
		})
	}
}

func TestStateFileRoundTrip(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	original := &State{
		Distro:            "arch",
		CPUs:              8,
		MemoryMB:          8192,
		DiskSizeMB:        51200,
		SharedDirs:        []string{"/home/test"},
		EnableNetwork:     true,
		MACAddress:        "aa:bb:cc:dd:ee:ff",
		SSHHostPort:       2223,
		IsDefaultTerminal: false,
	}

	// Write state to file
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}
	if err := os.WriteFile(statePath, data, 0600); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Read state back
	readData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var loaded State
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}

	// Verify round-trip
	if loaded.Distro != original.Distro {
		t.Errorf("Distro mismatch after round-trip: got %q, want %q", loaded.Distro, original.Distro)
	}
	if loaded.CPUs != original.CPUs {
		t.Errorf("CPUs mismatch after round-trip: got %d, want %d", loaded.CPUs, original.CPUs)
	}
	if loaded.MemoryMB != original.MemoryMB {
		t.Errorf("MemoryMB mismatch after round-trip: got %d, want %d", loaded.MemoryMB, original.MemoryMB)
	}
	if loaded.MACAddress != original.MACAddress {
		t.Errorf("MACAddress mismatch after round-trip: got %q, want %q", loaded.MACAddress, original.MACAddress)
	}
}

func TestGetPaths(t *testing.T) {
	paths, err := GetPaths()
	if err != nil {
		t.Fatalf("GetPaths failed: %v", err)
	}

	if paths == nil {
		t.Fatal("GetPaths should not return nil")
	}

	// DataDir should be set
	if paths.DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	// ConfigDir should be set
	if paths.ConfigDir == "" {
		t.Error("ConfigDir should not be empty")
	}

	// ConfigFile should be set
	if paths.ConfigFile == "" {
		t.Error("ConfigFile should not be empty")
	}

	// DataDir should contain .vmterminal
	if !filepath.IsAbs(paths.DataDir) {
		t.Error("DataDir should be absolute path")
	}
}

func TestDefaultConfig(t *testing.T) {
	// Test legacy DefaultConfig function
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	if cfg.VMName != "vmterminal" {
		t.Errorf("VMName should be 'vmterminal', got %q", cfg.VMName)
	}

	if cfg.Distro != "alpine" {
		t.Errorf("Distro should be 'alpine', got %q", cfg.Distro)
	}

	if cfg.CPUs <= 0 {
		t.Errorf("CPUs should be positive, got %d", cfg.CPUs)
	}

	if cfg.MemoryMB != 2048 {
		t.Errorf("MemoryMB should be 2048, got %d", cfg.MemoryMB)
	}
}
