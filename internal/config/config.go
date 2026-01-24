// Package config provides configuration management for VMTerminal.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// State holds all VMTerminal configuration state.
// This is stored in an internal JSON file, not user-editable.
// Configuration is changed via 'vmterminal config' interactive editor.
type State struct {
	// Distro is the Linux distribution to use.
	Distro string `json:"distro"`

	// CPUs is the number of virtual CPUs allocated to the VM.
	CPUs int `json:"cpus"`

	// MemoryMB is the amount of RAM in megabytes allocated to the VM.
	MemoryMB int `json:"memory_mb"`

	// DiskSizeMB is the disk image size in megabytes.
	DiskSizeMB int `json:"disk_size_mb"`

	// SharedDirs are host directories mounted inside the VM.
	SharedDirs []string `json:"shared_dirs"`

	// EnableNetwork enables VM networking (NAT mode on macOS).
	EnableNetwork bool `json:"enable_network"`

	// MACAddress is an optional custom MAC address (empty = auto-generate).
	MACAddress string `json:"mac_address,omitempty"`

	// SSHHostPort is the host port for SSH port forwarding (0 = disabled).
	SSHHostPort int `json:"ssh_host_port"`

	// IsDefaultTerminal indicates if VM is set as default terminal.
	IsDefaultTerminal bool `json:"is_default_terminal"`
}

// DefaultState returns a State with sensible defaults.
func DefaultState() *State {
	home, _ := os.UserHomeDir()
	sharedDirs := []string{}
	if home != "" {
		sharedDirs = append(sharedDirs, home)
	}

	return &State{
		Distro:            "alpine",
		CPUs:              runtime.NumCPU(),
		MemoryMB:          2048,
		DiskSizeMB:        10240, // 10GB
		SharedDirs:        sharedDirs,
		EnableNetwork:     true,
		MACAddress:        "",
		SSHHostPort:       2222,
		IsDefaultTerminal: false,
	}
}

// stateFilePath returns the path to the state file.
func stateFilePath() (string, error) {
	paths, err := GetPaths()
	if err != nil {
		return "", err
	}
	return filepath.Join(paths.DataDir, "state.json"), nil
}

// LoadState reads the state from the internal state file.
func LoadState() (*State, error) {
	statePath, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, err
	}

	state := &State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}

	return state, nil
}

// SaveState writes the state to the internal state file.
func SaveState(state *State) error {
	statePath, err := stateFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0600)
}

// Config holds all VMTerminal configuration (legacy support).
// Deprecated: Use State instead.
type Config struct {
	VMName        string   `json:"vm_name"`
	Distro        string   `json:"distro"`
	CPUs          int      `json:"cpus"`
	MemoryMB      int      `json:"memory_mb"`
	DiskSizeMB    int      `json:"disk_size_mb"`
	DiskPath      string   `json:"disk_path"`
	SharedDirs    []string `json:"shared_dirs"`
	EnableNetwork bool     `json:"enable_network"`
	MACAddress    string   `json:"mac_address"`
	SSHUser       string   `json:"ssh_user"`
	SSHPort       int      `json:"ssh_port"`
	SSHKeyPath    string   `json:"ssh_key_path"`
	SSHHostPort   int      `json:"ssh_host_port"`
	VMIP          string   `json:"vm_ip"`
}

// DefaultConfig returns a Config with sensible defaults.
// Deprecated: Use DefaultState instead.
func DefaultConfig() *Config {
	paths, err := GetPaths()
	if err != nil {
		paths = &Paths{
			DataDir: "/tmp/vmterminal",
		}
	}

	home, _ := os.UserHomeDir()
	sharedDirs := []string{}
	if home != "" {
		sharedDirs = append(sharedDirs, home)
	}

	return &Config{
		VMName:        "vmterminal",
		Distro:        "alpine",
		CPUs:          runtime.NumCPU(),
		MemoryMB:      2048,
		DiskSizeMB:    10240,
		DiskPath:      filepath.Join(paths.DataDir, "disk.img"),
		SharedDirs:    sharedDirs,
		EnableNetwork: true,
		MACAddress:    "",
		SSHUser:       "root",
		SSHPort:       22,
		SSHKeyPath:    "",
		SSHHostPort:   2222,
		VMIP:          "",
	}
}

// Global holds the loaded configuration (legacy support).
// Deprecated: Use LoadState instead.
var Global *Config

// Load reads configuration from file (legacy support).
// Deprecated: Use LoadState instead.
func Load() error {
	state, err := LoadState()
	if err != nil {
		// Use defaults if state doesn't exist
		Global = DefaultConfig()
		return nil
	}

	// Convert state to legacy config
	paths, _ := GetPaths()
	Global = &Config{
		VMName:        "default",
		Distro:        state.Distro,
		CPUs:          state.CPUs,
		MemoryMB:      state.MemoryMB,
		DiskSizeMB:    state.DiskSizeMB,
		DiskPath:      filepath.Join(paths.DataDir, "data", "default", "disk.img"),
		SharedDirs:    state.SharedDirs,
		EnableNetwork: state.EnableNetwork,
		MACAddress:    state.MACAddress,
		SSHUser:       "root",
		SSHPort:       22,
		SSHKeyPath:    "",
		SSHHostPort:   state.SSHHostPort,
		VMIP:          "",
	}

	return nil
}
