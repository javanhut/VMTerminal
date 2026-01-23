package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PersistentState holds VM state that survives restarts.
type PersistentState struct {
	// LastBoot is when the VM was last started.
	LastBoot time.Time `json:"last_boot,omitempty"`

	// LastShutdown is when the VM was last stopped.
	LastShutdown time.Time `json:"last_shutdown,omitempty"`

	// BootCount is the number of times the VM has booted.
	BootCount int `json:"boot_count"`

	// KernelVersion is the cached kernel version string.
	KernelVersion string `json:"kernel_version,omitempty"`

	// DiskSizeMB is the configured disk size.
	DiskSizeMB int64 `json:"disk_size_mb"`

	// CleanShutdown indicates if the last shutdown was clean.
	CleanShutdown bool `json:"clean_shutdown"`
}

// StateFile manages persistent state storage.
type StateFile struct {
	path string
}

// NewStateFile creates a state file manager.
func NewStateFile(dataDir string) *StateFile {
	return &StateFile{
		path: filepath.Join(dataDir, "state.json"),
	}
}

// Load reads the state from disk.
func (s *StateFile) Load() (*PersistentState, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &PersistentState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}

	return &state, nil
}

// Save writes the state to disk.
func (s *StateFile) Save(state *PersistentState) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Write atomically
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return os.Rename(tmpPath, s.path)
}

// RecordBoot updates state for a new boot.
func (s *StateFile) RecordBoot() error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.LastBoot = time.Now()
	state.BootCount++
	state.CleanShutdown = false

	return s.Save(state)
}

// RecordShutdown updates state for a shutdown.
func (s *StateFile) RecordShutdown(clean bool) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.LastShutdown = time.Now()
	state.CleanShutdown = clean

	return s.Save(state)
}

// Path returns the state file path.
func (s *StateFile) Path() string {
	return s.path
}
