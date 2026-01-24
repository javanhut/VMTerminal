package vm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Check if we need to extract kernel from rootfs
	locator := m.provider.KernelLocator()

	if locator != nil {
		// Download rootfs first (needed for extraction)
		if urls.Rootfs != "" {
			ext := filepath.Ext(urls.Rootfs)
			paths.Rootfs = filepath.Join(cacheSubdir, "rootfs"+ext)
			if err := m.ensureFile(paths.Rootfs, urls.Rootfs); err != nil {
				return nil, fmt.Errorf("download rootfs: %w", err)
			}
		}

		// Check if kernel/initrd already extracted
		kernelPath := filepath.Join(cacheSubdir, "vmlinuz")
		initrdPath := filepath.Join(cacheSubdir, "initramfs")

		_, kernelErr := os.Stat(kernelPath)
		_, initrdErr := os.Stat(initrdPath)

		if os.IsNotExist(kernelErr) || os.IsNotExist(initrdErr) {
			// Extract kernel/initrd from rootfs
			fmt.Printf("Extracting kernel and initrd from %s...\n", filepath.Base(paths.Rootfs))
			extractor := NewKernelExtractor(cacheSubdir)
			kernel, initrd, err := extractor.ExtractKernel(paths.Rootfs, locator)
			if err != nil {
				return nil, fmt.Errorf("extract kernel: %w", err)
			}
			paths.Kernel = kernel
			paths.Initramfs = initrd
		} else {
			paths.Kernel = kernelPath
			paths.Initramfs = initrdPath
		}
	} else {
		// Direct download (Alpine-style or iso: URL scheme)
		// Download kernel if URL is provided
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
			ext := filepath.Ext(urls.Rootfs)
			paths.Rootfs = filepath.Join(cacheSubdir, "rootfs"+ext)
			if err := m.ensureFile(paths.Rootfs, urls.Rootfs); err != nil {
				return nil, fmt.Errorf("download rootfs: %w", err)
			}
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
	for _, ext := range []string{".tar.gz", ".tar.xz", ".tar.zst", ".qcow2"} {
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

	// Check if we need kernel extraction
	locator := m.provider.KernelLocator()

	if locator != nil {
		// For distros with kernel extraction, check extracted files
		if paths.Kernel == "" || paths.Initramfs == "" {
			return false, nil
		}
		if urls.Rootfs != "" && paths.Rootfs == "" {
			return false, nil
		}
	} else {
		// For direct download distros, check URLs vs paths
		if urls.Kernel != "" && paths.Kernel == "" {
			return false, nil
		}
		if urls.Initrd != "" && paths.Initramfs == "" {
			return false, nil
		}
		if urls.Rootfs != "" && paths.Rootfs == "" {
			return false, nil
		}
	}

	return true, nil
}

func (m *AssetManager) ensureFile(path, url string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // Already exists
	}

	// Handle iso: URL scheme for extracting files from ISOs
	// Format: iso:<iso-url>#<path-in-iso>
	if strings.HasPrefix(url, "iso:") {
		return m.ensureFileFromISO(path, url)
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

// ensureFileFromISO extracts a file from an ISO image.
// URL format: iso:<iso-url>#<path-in-iso>
func (m *AssetManager) ensureFileFromISO(destPath, isoURL string) error {
	// Parse the ISO URL and path
	// Format: iso:https://example.com/file.iso#/path/in/iso
	url := strings.TrimPrefix(isoURL, "iso:")
	parts := strings.SplitN(url, "#", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid iso URL format: %s (expected iso:<url>#<path>)", isoURL)
	}
	isoDownloadURL := parts[0]
	pathInISO := parts[1]

	// Ensure ISO is downloaded to cache
	isoPath := filepath.Join(m.cacheDir, "iso", filepath.Base(isoDownloadURL))
	if err := os.MkdirAll(filepath.Dir(isoPath), 0755); err != nil {
		return fmt.Errorf("create iso cache dir: %w", err)
	}

	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		fmt.Printf("Downloading ISO: %s\n", filepath.Base(isoDownloadURL))
		if err := m.downloadFile(isoPath, isoDownloadURL); err != nil {
			return fmt.Errorf("download ISO: %w", err)
		}
	}

	// Extract the file from ISO using bsdtar (preferred) or 7z
	fmt.Printf("Extracting %s from ISO...\n", filepath.Base(pathInISO))

	// Try bsdtar first (available on most Linux systems)
	if _, err := exec.LookPath("bsdtar"); err == nil {
		return m.extractWithBsdtar(isoPath, pathInISO, destPath)
	}

	// Try 7z as fallback
	if _, err := exec.LookPath("7z"); err == nil {
		return m.extractWith7z(isoPath, pathInISO, destPath)
	}

	return fmt.Errorf("cannot extract from ISO: install bsdtar (libarchive) or 7z (p7zip)")
}

// extractWithBsdtar extracts a single file from an ISO using bsdtar.
func (m *AssetManager) extractWithBsdtar(isoPath, pathInISO, destPath string) error {
	// bsdtar can read ISOs directly
	// Extract to stdout and write to destination
	// Path in ISO typically starts with / which bsdtar expects without leading /
	isoInternalPath := strings.TrimPrefix(pathInISO, "/")

	tmpPath := destPath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer outFile.Close()

	cmd := exec.Command("bsdtar", "-xOf", isoPath, isoInternalPath)
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("bsdtar extract: %w", err)
	}

	outFile.Close()
	return os.Rename(tmpPath, destPath)
}

// extractWith7z extracts a single file from an ISO using 7z.
func (m *AssetManager) extractWith7z(isoPath, pathInISO, destPath string) error {
	// 7z extracts to current directory, so we need a temp dir
	tmpDir, err := os.MkdirTemp("", "vmterminal-iso-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 7z uses paths without leading /
	isoInternalPath := strings.TrimPrefix(pathInISO, "/")

	cmd := exec.Command("7z", "x", "-o"+tmpDir, isoPath, isoInternalPath)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("7z extract: %w", err)
	}

	// Move extracted file to destination
	extractedPath := filepath.Join(tmpDir, isoInternalPath)
	return os.Rename(extractedPath, destPath)
}

// downloadFile downloads a URL to a local path.
func (m *AssetManager) downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s (URL: %s)", resp.Status, url)
	}

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
