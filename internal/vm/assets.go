package vm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/javanstorm/vmterminal/internal/distro"
)

// AssetManager handles kernel, initramfs, and rootfs downloads.
type AssetManager struct {
	cacheDir string
	provider distro.Provider
}

// NewAssetManager creates an asset manager with the given cache directory and distro provider.
func NewAssetManager(cacheDir string, provider distro.Provider) *AssetManager {
	return &AssetManager{
		cacheDir: cacheDir,
		provider: provider,
	}
}

// AssetPaths contains paths to downloaded assets.
type AssetPaths struct {
	Kernel    string
	Initramfs string
	Rootfs    string
}

// EnsureAssets downloads kernel, initramfs, and rootfs if not already cached.
// Returns paths to the assets.
func (m *AssetManager) EnsureAssets() (*AssetPaths, error) {
	arch := distro.CurrentArch()
	if arch == "" {
		return nil, fmt.Errorf("unsupported architecture")
	}

	if !m.provider.SupportsArch(arch) {
		return nil, fmt.Errorf("distro %s does not support architecture %s", m.provider.ID(), arch)
	}

	// Get distro-specific cache subdirectory
	cacheSubdir := filepath.Join(m.cacheDir, m.provider.CacheSubdir(arch))
	if err := os.MkdirAll(cacheSubdir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	// Get asset URLs from provider
	urls, err := m.provider.AssetURLs(arch)
	if err != nil {
		return nil, fmt.Errorf("get asset URLs: %w", err)
	}

	paths := &AssetPaths{}

	// Download kernel if URL is provided (some distros extract from rootfs)
	if urls.Kernel != "" {
		paths.Kernel = filepath.Join(cacheSubdir, "vmlinuz")
		if err := m.ensureFile(paths.Kernel, urls.Kernel); err != nil {
			return nil, fmt.Errorf("download kernel: %w", err)
		}
	}

	// Download initramfs if URL is provided
	if urls.Initrd != "" {
		paths.Initramfs = filepath.Join(cacheSubdir, "initramfs")
		if err := m.ensureFile(paths.Initramfs, urls.Initrd); err != nil {
			return nil, fmt.Errorf("download initramfs: %w", err)
		}
	}

	// Download rootfs if URL is provided
	if urls.Rootfs != "" {
		// Determine extension from URL
		ext := filepath.Ext(urls.Rootfs)
		paths.Rootfs = filepath.Join(cacheSubdir, "rootfs"+ext)
		if err := m.ensureFile(paths.Rootfs, urls.Rootfs); err != nil {
			return nil, fmt.Errorf("download rootfs: %w", err)
		}
	}

	return paths, nil
}

// CacheDir returns the cache directory path.
func (m *AssetManager) CacheDir() string {
	return m.cacheDir
}

// Provider returns the distro provider.
func (m *AssetManager) Provider() distro.Provider {
	return m.provider
}

// DistroID returns the distro ID.
func (m *AssetManager) DistroID() distro.ID {
	return m.provider.ID()
}

// BootConfig returns the boot configuration for the current architecture.
func (m *AssetManager) BootConfig() *distro.BootConfig {
	return m.provider.BootConfig(distro.CurrentArch())
}

// SetupRequirements returns the setup requirements for the distro.
func (m *AssetManager) SetupRequirements() *distro.SetupRequirements {
	return m.provider.SetupRequirements()
}

// GetAssetPaths returns paths for cached assets without downloading.
// Returns nil paths for assets that don't exist.
func (m *AssetManager) GetAssetPaths() (*AssetPaths, error) {
	arch := distro.CurrentArch()
	if arch == "" {
		return nil, fmt.Errorf("unsupported architecture")
	}

	cacheSubdir := filepath.Join(m.cacheDir, m.provider.CacheSubdir(arch))
	paths := &AssetPaths{}

	kernelPath := filepath.Join(cacheSubdir, "vmlinuz")
	if _, err := os.Stat(kernelPath); err == nil {
		paths.Kernel = kernelPath
	}

	initramfsPath := filepath.Join(cacheSubdir, "initramfs")
	if _, err := os.Stat(initramfsPath); err == nil {
		paths.Initramfs = initramfsPath
	}

	// Check for rootfs with various extensions
	for _, ext := range []string{".tar.gz", ".tar.xz", ".qcow2"} {
		rootfsPath := filepath.Join(cacheSubdir, "rootfs"+ext)
		if _, err := os.Stat(rootfsPath); err == nil {
			paths.Rootfs = rootfsPath
			break
		}
	}

	return paths, nil
}

// AssetsExist checks if all required assets are cached.
func (m *AssetManager) AssetsExist() (bool, error) {
	paths, err := m.GetAssetPaths()
	if err != nil {
		return false, err
	}

	urls, err := m.provider.AssetURLs(distro.CurrentArch())
	if err != nil {
		return false, err
	}

	// Check each asset that should exist
	if urls.Kernel != "" && paths.Kernel == "" {
		return false, nil
	}
	if urls.Initrd != "" && paths.Initramfs == "" {
		return false, nil
	}
	if urls.Rootfs != "" && paths.Rootfs == "" {
		return false, nil
	}

	return true, nil
}

func (m *AssetManager) ensureFile(path, url string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // Already exists
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (URL: %s)", resp.Status, url)
	}

	// Write to temp file first, then rename for atomicity
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}
