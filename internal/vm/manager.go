package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/javanstorm/vmterminal/pkg/hypervisor"
)

// State represents the VM lifecycle state.
type State int

const (
	StateNew State = iota
	StateReady     // Assets downloaded, disk created
	StateRunning   // VM is running
	StateStopping  // Shutdown in progress
	StateStopped   // Clean shutdown complete
	StateError     // Error state
)

func (s State) String() string {
	switch s {
	case StateNew:
		return "new"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// ManagerConfig holds configuration for the VM manager.
type ManagerConfig struct {
	// CacheDir is where kernel/initramfs are cached.
	CacheDir string

	// DataDir is where disk images are stored.
	DataDir string

	// CPUs is the number of virtual CPUs.
	CPUs int

	// MemoryMB is the amount of memory in megabytes.
	MemoryMB int

	// DiskSizeMB is the disk image size in megabytes.
	DiskSizeMB int64

	// DiskName is the name for the disk image (without extension).
	DiskName string

	// SharedDirs maps mount tags to host paths for filesystem sharing.
	SharedDirs map[string]string

	// EnableNetwork enables VM networking.
	EnableNetwork bool

	// MACAddress is optional custom MAC (empty = auto-generate).
	MACAddress string

	// SSHHostPort is the host port for SSH port forwarding (0 = disabled).
	SSHHostPort int

	// Provider is the distribution provider.
	Provider distro.Provider
}

// Manager orchestrates VM lifecycle with asset and disk management.
type Manager struct {
	cfg       ManagerConfig
	assets    *AssetManager
	images    *ImageManager
	driver    hypervisor.Driver
	stateFile *StateFile
	mu        sync.RWMutex
	state     State
	errCh     chan error
	lastErr   error
}

// NewManager creates a new VM manager.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	driver, err := hypervisor.NewDriver()
	if err != nil {
		return nil, fmt.Errorf("create hypervisor driver: %w", err)
	}

	// Apply defaults
	if cfg.CPUs == 0 {
		cfg.CPUs = 1
	}
	if cfg.MemoryMB == 0 {
		cfg.MemoryMB = 512
	}
	if cfg.DiskSizeMB == 0 {
		cfg.DiskSizeMB = DefaultDiskSizeMB
	}
	if cfg.DiskName == "" {
		cfg.DiskName = "root"
	}

	// Use default provider if not specified
	if cfg.Provider == nil {
		var err error
		cfg.Provider, err = distro.GetDefault()
		if err != nil {
			return nil, fmt.Errorf("get default distro: %w", err)
		}
	}

	return &Manager{
		cfg:       cfg,
		assets:    NewAssetManager(cfg.CacheDir, cfg.Provider),
		images:    NewImageManager(cfg.DataDir),
		driver:    driver,
		stateFile: NewStateFile(cfg.DataDir),
		state:     StateNew,
	}, nil
}

// Prepare downloads assets and creates disk image if needed.
// Uses optimized warm path when assets and disk already exist.
func (m *Manager) Prepare(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateNew && m.state != StateStopped {
		return fmt.Errorf("cannot prepare: invalid state %s", m.state)
	}

	// Check for warm path - assets and disk already exist
	if m.isWarmPath() {
		return m.warmPrepare(ctx)
	}

	return m.coldPrepare(ctx)
}

// isWarmPath returns true if all assets and disk already exist (no downloads needed).
func (m *Manager) isWarmPath() bool {
	// Check if assets exist
	exist, _ := m.assets.AssetsExist()
	if !exist {
		return false
	}

	// Check if disk exists - depends on distro setup requirements
	setupReqs := m.assets.SetupRequirements()
	if setupReqs != nil && !setupReqs.NeedsExtraction {
		// For qcow2-based distros, check if rootfs.raw exists
		paths, err := m.assets.GetAssetPaths()
		if err != nil {
			return false
		}
		// Rootfs path should be the raw converted image
		if paths.Rootfs == "" {
			return false
		}
		if _, err := os.Stat(paths.Rootfs); err != nil {
			return false
		}
		return true
	}

	// For extraction-based distros, check if disk exists
	return m.images.DiskExists(m.cfg.DiskName)
}

// warmPrepare is the optimized path when assets and disk already exist.
// Skips EnsureAssets/EnsureDisk overhead and goes directly to VM creation.
func (m *Manager) warmPrepare(ctx context.Context) error {
	// Get cached paths directly - we know they exist
	assetPaths, err := m.assets.GetAssetPaths()
	if err != nil {
		// Fallback to cold path if GetAssetPaths fails
		return m.coldPrepare(ctx)
	}

	// Determine disk path based on distro setup requirements
	var diskPath string
	setupReqs := m.assets.SetupRequirements()
	if setupReqs != nil && !setupReqs.NeedsExtraction && assetPaths.Rootfs != "" {
		// For qcow2-based distros, use the converted raw image
		diskPath = assetPaths.Rootfs
	} else {
		diskPath = m.images.DiskPath(m.cfg.DiskName)
	}

	bootConfig := m.assets.BootConfig()

	// Configure and create VM
	vmCfg := &hypervisor.VMConfig{
		CPUs:          m.cfg.CPUs,
		MemoryMB:      m.cfg.MemoryMB,
		Kernel:        assetPaths.Kernel,
		Initrd:        assetPaths.Initramfs,
		Cmdline:       bootConfig.Cmdline,
		DiskPath:      diskPath,
		SharedDirs:    m.cfg.SharedDirs,
		EnableNetwork: m.cfg.EnableNetwork,
		MACAddress:    m.cfg.MACAddress,
	}

	// Add SSH port forwarding if configured
	if m.cfg.SSHHostPort > 0 {
		vmCfg.PortForwards = map[int]int{
			m.cfg.SSHHostPort: 22,
		}
	}

	// Skip Validate on warm path - config hasn't changed since last successful run
	if err := m.driver.Create(ctx, vmCfg); err != nil {
		m.state = StateError
		m.lastErr = err
		return fmt.Errorf("create VM: %w", err)
	}

	m.state = StateReady
	return nil
}

// coldPrepare is the full path that ensures assets and disk exist.
func (m *Manager) coldPrepare(ctx context.Context) error {
	// Download kernel/initramfs if needed
	assetPaths, err := m.assets.EnsureAssets()
	if err != nil {
		m.state = StateError
		m.lastErr = err
		return fmt.Errorf("ensure assets: %w", err)
	}

	// Determine disk path based on distro setup requirements
	var diskPath string
	setupReqs := m.assets.SetupRequirements()

	if setupReqs != nil && !setupReqs.NeedsExtraction && assetPaths.Rootfs != "" {
		// For distros like Ubuntu where rootfs is the complete disk image,
		// use the converted raw image directly
		diskPath = assetPaths.Rootfs
	} else {
		// Create disk image if needed (for distros like Alpine that need extraction)
		diskPath, err = m.images.EnsureDisk(m.cfg.DiskName, m.cfg.DiskSizeMB)
		if err != nil {
			m.state = StateError
			m.lastErr = err
			return fmt.Errorf("ensure disk: %w", err)
		}
	}

	// Get boot config from provider
	bootConfig := m.assets.BootConfig()

	// Configure and create VM
	vmCfg := &hypervisor.VMConfig{
		CPUs:          m.cfg.CPUs,
		MemoryMB:      m.cfg.MemoryMB,
		Kernel:        assetPaths.Kernel,
		Initrd:        assetPaths.Initramfs,
		Cmdline:       bootConfig.Cmdline,
		DiskPath:      diskPath,
		SharedDirs:    m.cfg.SharedDirs,
		EnableNetwork: m.cfg.EnableNetwork,
		MACAddress:    m.cfg.MACAddress,
	}

	// Add SSH port forwarding if configured
	if m.cfg.SSHHostPort > 0 {
		vmCfg.PortForwards = map[int]int{
			m.cfg.SSHHostPort: 22,
		}
	}

	if err := m.driver.Validate(ctx, vmCfg); err != nil {
		m.state = StateError
		m.lastErr = err
		return fmt.Errorf("validate config: %w", err)
	}

	if err := m.driver.Create(ctx, vmCfg); err != nil {
		m.state = StateError
		m.lastErr = err
		return fmt.Errorf("create VM: %w", err)
	}

	m.state = StateReady
	return nil
}

// Start boots the VM.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateReady {
		return fmt.Errorf("cannot start: invalid state %s", m.state)
	}

	errCh, err := m.driver.Start(ctx)
	if err != nil {
		m.state = StateError
		m.lastErr = err
		return fmt.Errorf("start VM: %w", err)
	}

	m.errCh = errCh
	m.state = StateRunning

	// Record boot in persistent state
	if err := m.stateFile.RecordBoot(); err != nil {
		// Log but don't fail - state tracking is non-critical
		fmt.Fprintf(os.Stderr, "Warning: failed to record boot: %v\n", err)
	}

	// Monitor VM in background
	go m.monitorVM()

	return nil
}

// Stop gracefully shuts down the VM.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if m.state != StateRunning {
		m.mu.Unlock()
		return fmt.Errorf("cannot stop: invalid state %s", m.state)
	}
	m.state = StateStopping
	m.mu.Unlock()

	if err := m.driver.Stop(ctx); err != nil {
		m.mu.Lock()
		m.state = StateError
		m.lastErr = err
		m.mu.Unlock()
		return fmt.Errorf("stop VM: %w", err)
	}

	return nil
}

// Kill forcefully terminates the VM.
func (m *Manager) Kill(ctx context.Context) error {
	m.mu.Lock()
	if m.state != StateRunning && m.state != StateStopping {
		m.mu.Unlock()
		return fmt.Errorf("cannot kill: invalid state %s", m.state)
	}
	m.mu.Unlock()

	if err := m.driver.Kill(ctx); err != nil {
		m.mu.Lock()
		m.state = StateError
		m.lastErr = err
		m.mu.Unlock()
		return fmt.Errorf("kill VM: %w", err)
	}

	return nil
}

// State returns the current VM state.
func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// LastError returns the last error that occurred.
func (m *Manager) LastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErr
}

// Wait blocks until the VM stops.
func (m *Manager) Wait() error {
	m.mu.RLock()
	errCh := m.errCh
	m.mu.RUnlock()

	if errCh == nil {
		return fmt.Errorf("VM not started")
	}

	return <-errCh
}

// DriverInfo returns hypervisor driver information.
func (m *Manager) DriverInfo() hypervisor.Info {
	return m.driver.Info()
}

func (m *Manager) monitorVM() {
	err := <-m.errCh

	// Record shutdown
	clean := err == nil
	if stateErr := m.stateFile.RecordShutdown(clean); stateErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to record shutdown: %v\n", stateErr)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		m.state = StateError
		m.lastErr = err
	} else {
		m.state = StateStopped
	}
}

// PersistentState returns the current persistent state.
func (m *Manager) PersistentState() (*PersistentState, error) {
	return m.stateFile.Load()
}

// Console returns VM console I/O handles. Only valid when VM is running.
func (m *Manager) Console() (io.Writer, io.Reader, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != StateRunning {
		return nil, nil, fmt.Errorf("VM not running")
	}
	return m.driver.Console()
}

// Provider returns the distro provider.
func (m *Manager) Provider() distro.Provider {
	return m.cfg.Provider
}
