package distro

import "fmt"

const (
	rockyVersion = "9"
	rockyBaseURL = "https://dl.rockylinux.org/pub/rocky"
)

// RockyProvider implements Provider for Rocky Linux.
type RockyProvider struct {
	BaseProvider
}

// NewRockyProvider creates a new Rocky Linux provider.
func NewRockyProvider() *RockyProvider {
	return &RockyProvider{
		BaseProvider: BaseProvider{
			id:      Rocky,
			name:    "Rocky Linux",
			version: rockyVersion,
			archs:   []Arch{ArchAMD64, ArchARM64},
		},
	}
}

// AssetURLs returns download URLs for Rocky Linux.
// Rocky provides cloud images with kernel inside.
func (p *RockyProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	rockyArch := p.toRockyArch(arch)

	// Rocky provides GenericCloud images
	return &AssetURLs{
		Kernel:  "", // Extracted from rootfs
		Initrd:  "", // Extracted from rootfs
		Rootfs:  fmt.Sprintf("%s/%s/images/%s/Rocky-%s-GenericCloud.latest.%s.qcow2", rockyBaseURL, p.version, rockyArch, p.version, rockyArch),
	}, nil
}

// BootConfig returns the kernel boot configuration for Rocky.
func (p *RockyProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		// Rocky uses XFS by default and we disable SELinux for simplicity
		Cmdline:       "console=hvc0 root=/dev/vda rw rootfstype=xfs selinux=0",
		RootDevice:    "/dev/vda",
		RootFSType:    "xfs",
		ConsoleDevice: "hvc0",
		ExtraModules:  "",
	}
}

// SetupRequirements returns setup requirements for Rocky.
func (p *RockyProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "xfs",
		NeedsExtraction: true,
	}
}

// toRockyArch converts our arch to Rocky's arch naming.
func (p *RockyProvider) toRockyArch(arch Arch) string {
	switch arch {
	case ArchAMD64:
		return "x86_64"
	case ArchARM64:
		return "aarch64"
	default:
		return ""
	}
}

// KernelLocator returns patterns for finding kernel in Rocky qcow2 image.
func (p *RockyProvider) KernelLocator() *KernelLocator {
	return &KernelLocator{
		KernelPatterns: []string{
			"boot/vmlinuz-*",
		},
		InitrdPatterns: []string{
			"boot/initramfs-*.img",
		},
		ArchiveType: "qcow2",
	}
}

func init() {
	Register(NewRockyProvider())
}
