package distro

import "fmt"

const (
	archVersion = "latest"
	archISOBase = "https://geo.mirror.pkgbuild.com/iso/latest"
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
// Kernel and initramfs are extracted from the ISO.
// Rootfs uses the bootstrap tarball.
func (p *ArchProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	// Use ISO for kernel extraction (marked with iso: prefix)
	// The asset manager will handle downloading the ISO and extracting
	isoURL := fmt.Sprintf("%s/archlinux-x86_64.iso", archISOBase)

	return &AssetURLs{
		Kernel: fmt.Sprintf("iso:%s#/arch/boot/x86_64/vmlinuz-linux", isoURL),
		Initrd: fmt.Sprintf("iso:%s#/arch/boot/x86_64/initramfs-linux.img", isoURL),
		Rootfs: fmt.Sprintf("%s/archlinux-bootstrap-x86_64.tar.zst", archISOBase),
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

// KernelLocator returns nil because Arch uses the iso: URL scheme for direct extraction.
// The kernel and initrd paths are specified in AssetURLs using the iso: prefix.
func (p *ArchProvider) KernelLocator() *KernelLocator {
	return nil
}

func init() {
	Register(NewArchProvider())
}
