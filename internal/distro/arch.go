package distro

import "fmt"

const (
	archVersion = "latest"
	archBaseURL = "https://geo.mirror.pkgbuild.com/images"
)

// ArchProvider implements Provider for Arch Linux.
type ArchProvider struct {
	BaseProvider
}

// NewArchProvider creates a new Arch Linux provider.
func NewArchProvider() *ArchProvider {
	return &ArchProvider{
		BaseProvider: BaseProvider{
			id:      ArchLinux,
			name:    "Arch Linux",
			version: archVersion,
			// Arch Linux only officially supports x86_64
			archs: []Arch{ArchAMD64},
		},
	}
}

// AssetURLs returns download URLs for Arch Linux.
// Arch provides a bootstrap tarball that contains the base system.
// Kernel needs to be extracted from the rootfs /boot directory.
func (p *ArchProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	// Arch provides cloudimg which has kernel inside
	// We'll use the cloud image rootfs
	return &AssetURLs{
		Kernel:  "", // Extracted from rootfs
		Initrd:  "", // Extracted from rootfs
		Rootfs:  fmt.Sprintf("%s/%s/Arch-Linux-x86_64-cloudimg.qcow2", archBaseURL, p.version),
	}, nil
}

// BootConfig returns the kernel boot configuration for Arch.
func (p *ArchProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		Cmdline:       "console=hvc0 root=/dev/vda rw init=/usr/lib/systemd/systemd",
		RootDevice:    "/dev/vda",
		RootFSType:    "ext4",
		ConsoleDevice: "hvc0",
		ExtraModules:  "",
	}
}

// SetupRequirements returns setup requirements for Arch.
func (p *ArchProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "ext4",
		NeedsExtraction: true,
	}
}

// NeedsKernelExtraction returns true because Arch kernel is inside rootfs.
func (p *ArchProvider) NeedsKernelExtraction() bool {
	return true
}

func init() {
	Register(NewArchProvider())
}
