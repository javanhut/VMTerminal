package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VMEntry represents a single VM configuration in the registry.
type VMEntry struct {
	Name       string    `json:"name"`
	Distro     string    `json:"distro"`
	CPUs       int       `json:"cpus"`
	MemoryMB   int       `json:"memory_mb"`
	DiskSizeMB int       `json:"disk_size_mb"`
	CreatedAt  time.Time `json:"created_at"`
}

// RegistryData holds the registry file contents.
type RegistryData struct {
	VMs []VMEntry `json:"vms"`
}

// Registry manages multiple VM configurations.
type Registry struct {
	baseDir      string
	registryPath string
	activePath   string
}

// NewRegistry creates a new registry instance.
func NewRegistry(baseDir string) *Registry {
	return &Registry{
		baseDir:      baseDir,
		registryPath: filepath.Join(baseDir, "vms.json"),
		activePath:   filepath.Join(baseDir, "active"),
	}
}

// Load reads the registry from disk.
func (r *Registry) Load() (*RegistryData, error) {
	data, err := os.ReadFile(r.registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &RegistryData{VMs: []VMEntry{}}, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var reg RegistryData
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	return &reg, nil
}

// Save writes the registry to disk.
func (r *Registry) Save(reg *RegistryData) error {
	// Ensure base directory exists
	if err := os.MkdirAll(r.baseDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	if err := os.WriteFile(r.registryPath, data, 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

// CreateVM adds a new VM to the registry.
func (r *Registry) CreateVM(entry VMEntry) error {
	reg, err := r.Load()
	if err != nil {
		return err
	}

	// Check for duplicate name
	for _, vm := range reg.VMs {
		if vm.Name == entry.Name {
			return fmt.Errorf("VM '%s' already exists", entry.Name)
		}
	}

	// Set creation time
	entry.CreatedAt = time.Now()

	reg.VMs = append(reg.VMs, entry)

	if err := r.Save(reg); err != nil {
		return err
	}

	// Create VM data directory
	vmDataDir := r.VMDataDir(entry.Name)
	if err := os.MkdirAll(vmDataDir, 0755); err != nil {
		return fmt.Errorf("create VM data directory: %w", err)
	}

	return nil
}

// GetVM returns a VM entry by name.
func (r *Registry) GetVM(name string) (*VMEntry, error) {
	reg, err := r.Load()
	if err != nil {
		return nil, err
	}

	for _, vm := range reg.VMs {
		if vm.Name == name {
			return &vm, nil
		}
	}

	return nil, fmt.Errorf("VM '%s' not found", name)
}

// ListVMs returns all VM entries.
func (r *Registry) ListVMs() ([]VMEntry, error) {
	reg, err := r.Load()
	if err != nil {
		return nil, err
	}

	return reg.VMs, nil
}

// DeleteVM removes a VM from the registry.
func (r *Registry) DeleteVM(name string) error {
	reg, err := r.Load()
	if err != nil {
		return err
	}

	found := false
	newVMs := make([]VMEntry, 0, len(reg.VMs))
	for _, vm := range reg.VMs {
		if vm.Name == name {
			found = true
		} else {
			newVMs = append(newVMs, vm)
		}
	}

	if !found {
		return fmt.Errorf("VM '%s' not found", name)
	}

	reg.VMs = newVMs

	// Clear active if this was the active VM
	active, _ := r.GetActive()
	if active == name {
		r.ClearActive()
	}

	return r.Save(reg)
}

// SetActive sets the active VM.
func (r *Registry) SetActive(name string) error {
	// Verify VM exists
	_, err := r.GetVM(name)
	if err != nil {
		return err
	}

	// Ensure base directory exists
	if err := os.MkdirAll(r.baseDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(r.activePath, []byte(name), 0644); err != nil {
		return fmt.Errorf("write active file: %w", err)
	}

	return nil
}

// GetActive returns the name of the active VM.
func (r *Registry) GetActive() (string, error) {
	data, err := os.ReadFile(r.activePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read active file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ClearActive removes the active VM setting.
func (r *Registry) ClearActive() error {
	if err := os.Remove(r.activePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove active file: %w", err)
	}
	return nil
}

// VMDataDir returns the data directory for a specific VM.
func (r *Registry) VMDataDir(name string) string {
	return filepath.Join(r.baseDir, "data", name)
}

// EnsureDefault creates a default VM if the registry is empty.
func (r *Registry) EnsureDefault(defaultDistro string, defaultCPUs, defaultMemoryMB, defaultDiskSizeMB int) error {
	reg, err := r.Load()
	if err != nil {
		return err
	}

	// If registry has VMs, nothing to do
	if len(reg.VMs) > 0 {
		return nil
	}

	// Create default VM
	entry := VMEntry{
		Name:       "default",
		Distro:     defaultDistro,
		CPUs:       defaultCPUs,
		MemoryMB:   defaultMemoryMB,
		DiskSizeMB: defaultDiskSizeMB,
	}

	if err := r.CreateVM(entry); err != nil {
		return fmt.Errorf("create default VM: %w", err)
	}

	// Set as active
	if err := r.SetActive("default"); err != nil {
		return fmt.Errorf("set default active: %w", err)
	}

	return nil
}

// GetActiveOrDefault returns the active VM, creating a default if needed.
func (r *Registry) GetActiveOrDefault(defaultDistro string, defaultCPUs, defaultMemoryMB, defaultDiskSizeMB int) (*VMEntry, error) {
	// Ensure default VM exists
	if err := r.EnsureDefault(defaultDistro, defaultCPUs, defaultMemoryMB, defaultDiskSizeMB); err != nil {
		return nil, err
	}

	// Get active VM name
	active, err := r.GetActive()
	if err != nil {
		return nil, err
	}

	// If no active, use first VM (should be "default")
	if active == "" {
		vms, err := r.ListVMs()
		if err != nil {
			return nil, err
		}
		if len(vms) == 0 {
			return nil, fmt.Errorf("no VMs available")
		}
		active = vms[0].Name
		r.SetActive(active)
	}

	return r.GetVM(active)
}

// DeleteVMData removes the VM's data directory.
func (r *Registry) DeleteVMData(name string) error {
	vmDataDir := r.VMDataDir(name)
	if err := os.RemoveAll(vmDataDir); err != nil {
		return fmt.Errorf("remove VM data: %w", err)
	}
	return nil
}
