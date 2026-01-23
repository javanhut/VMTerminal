// Package config provides configuration management for VMTerminal.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths holds platform-specific directory paths for VMTerminal.
type Paths struct {
	// ConfigDir is the directory for configuration files.
	// macOS: ~/Library/Application Support/VMTerminal
	// Linux: ~/.config/vmterminal (or XDG_CONFIG_HOME)
	ConfigDir string

	// DataDir is the directory for VM disk images and state.
	// All platforms: ~/.vmterminal
	DataDir string

	// ConfigFile is the path to the main config file.
	ConfigFile string
}

// GetPaths returns platform-aware paths for VMTerminal.
func GetPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	p := &Paths{}

	// Data directory is always ~/.vmterminal
	p.DataDir = filepath.Join(home, ".vmterminal")

	// Config directory is platform-specific
	switch runtime.GOOS {
	case "darwin":
		p.ConfigDir = filepath.Join(home, "Library", "Application Support", "VMTerminal")
	default: // Linux and others
		// Respect XDG_CONFIG_HOME if set
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			p.ConfigDir = filepath.Join(xdgConfig, "vmterminal")
		} else {
			p.ConfigDir = filepath.Join(home, ".config", "vmterminal")
		}
	}

	// Config file lives in data directory for simplicity
	p.ConfigFile = filepath.Join(p.DataDir, "config.yaml")

	return p, nil
}

// EnsureDirectories creates the config and data directories if they don't exist.
func (p *Paths) EnsureDirectories() error {
	if err := os.MkdirAll(p.ConfigDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(p.DataDir, 0755); err != nil {
		return err
	}
	return nil
}
