// Package distro provides distribution-specific configuration for VM boot.
package distro

import (
	"fmt"
	"runtime"
)

// ID identifies a Linux distribution.
type ID string

const (
	Alpine    ID = "alpine"
	Ubuntu    ID = "ubuntu"
	ArchLinux ID = "arch"
	Debian    ID = "debian"
	Rocky     ID = "rocky"
	OpenSUSE  ID = "opensuse"
)

// AllDistros returns all supported distribution IDs.
func AllDistros() []ID {
	return []ID{Alpine, Ubuntu, ArchLinux, Debian, Rocky, OpenSUSE}
}

// Arch represents a CPU architecture.
type Arch string

const (
	ArchAMD64 Arch = "amd64"
	ArchARM64 Arch = "arm64"
)

// CurrentArch returns the current system architecture.
func CurrentArch() Arch {
	switch runtime.GOARCH {
	case "amd64":
		return ArchAMD64
	case "arm64":
		return ArchARM64
	default:
		return ""
	}
}

// AssetURLs contains download URLs for distro assets.
type AssetURLs struct {
	Kernel  string // URL for kernel (vmlinuz)
	Initrd  string // URL for initial ramdisk
	Rootfs  string // URL for root filesystem tarball
}

// BootConfig contains kernel boot configuration.
type BootConfig struct {
	Cmdline       string // Kernel command line
	RootDevice    string // Root device (e.g., /dev/vda)
	RootFSType    string // Root filesystem type (e.g., ext4)
	ConsoleDevice string // Console device (e.g., hvc0)
	ExtraModules  string // Additional kernel modules to load
}

// SetupRequirements describes what's needed to set up the rootfs.
type SetupRequirements struct {
	NeedsFormatting bool   // Whether disk needs formatting
	FSType          string // Filesystem type to format with
	NeedsExtraction bool   // Whether rootfs tarball needs extraction
}

// KernelLocator defines how to find kernel/initrd within an archive.
type KernelLocator struct {
	// KernelPatterns - glob patterns to find kernel (tried in order)
	// e.g., ["boot/vmlinuz-*-generic", "boot/vmlinuz-*", "vmlinuz"]
	KernelPatterns []string

	// InitrdPatterns - glob patterns to find initrd (tried in order)
	// e.g., ["boot/initrd.img-*-generic", "boot/initramfs-*.img", "initrd"]
	InitrdPatterns []string

	// ArchiveType - "tarball", "qcow2", or "iso"
	ArchiveType string
}

// Provider defines the interface for distribution-specific configuration.
type Provider interface {
	// ID returns the unique identifier for this distribution.
	ID() ID

	// Name returns the human-readable name.
	Name() string

	// Version returns the distribution version.
	Version() string

	// SupportedArchs returns the architectures this distro supports.
	SupportedArchs() []Arch

	// SupportsArch checks if the given architecture is supported.
	SupportsArch(arch Arch) bool

	// AssetURLs returns download URLs for the given architecture.
	AssetURLs(arch Arch) (*AssetURLs, error)

	// BootConfig returns the kernel boot configuration.
	BootConfig(arch Arch) *BootConfig

	// SetupRequirements returns what's needed to set up the rootfs.
	SetupRequirements() *SetupRequirements

	// CacheSubdir returns the subdirectory name for caching assets.
	// Format: {distro}/{version}/{arch}
	CacheSubdir(arch Arch) string

	// KernelLocator returns patterns for finding kernel in rootfs.
	// Returns nil if kernel is provided via direct URL.
	KernelLocator() *KernelLocator
}

// BaseProvider implements common Provider functionality.
type BaseProvider struct {
	id       ID
	name     string
	version  string
	archs    []Arch
}

// ID returns the distribution identifier.
func (p *BaseProvider) ID() ID {
	return p.id
}

// Name returns the human-readable name.
func (p *BaseProvider) Name() string {
	return p.name
}

// Version returns the distribution version.
func (p *BaseProvider) Version() string {
	return p.version
}

// SupportedArchs returns the supported architectures.
func (p *BaseProvider) SupportedArchs() []Arch {
	return p.archs
}

// SupportsArch checks if the given architecture is supported.
func (p *BaseProvider) SupportsArch(arch Arch) bool {
	for _, a := range p.archs {
		if a == arch {
			return true
		}
	}
	return false
}

// CacheSubdir returns the cache subdirectory.
func (p *BaseProvider) CacheSubdir(arch Arch) string {
	return fmt.Sprintf("%s/%s/%s", p.id, p.version, arch)
}

// KernelLocator returns nil by default (kernel provided via direct URL).
// Override in providers that need kernel extraction from rootfs.
func (p *BaseProvider) KernelLocator() *KernelLocator {
	return nil
}

// ErrUnsupportedArch is returned when an architecture is not supported.
type ErrUnsupportedArch struct {
	Distro ID
	Arch   Arch
}

func (e *ErrUnsupportedArch) Error() string {
	return fmt.Sprintf("architecture %s not supported by %s", e.Arch, e.Distro)
}
