package distro

import "fmt"

const (
	alpineVersion = "3.21"
	alpineBaseURL = "https://dl-cdn.alpinelinux.org/alpine/v%s/releases/%s"
)

// AlpineProvider implements Provider for Alpine Linux.
type AlpineProvider struct {
	BaseProvider
}

// NewAlpineProvider creates a new Alpine Linux provider.
func NewAlpineProvider() *AlpineProvider {
	return &AlpineProvider{
		BaseProvider: BaseProvider{
			id:      Alpine,
			name:    "Alpine Linux",
			version: alpineVersion,
			archs:   []Arch{ArchAMD64, ArchARM64},
		},
	}
}

// AssetURLs returns download URLs for Alpine Linux.
func (p *AlpineProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	alpineArch := p.toAlpineArch(arch)
	baseURL := fmt.Sprintf(alpineBaseURL, p.version, alpineArch)

	// Note: For kernel and initramfs, we use netboot directory
	// For rootfs, we use the minirootfs tarball
	netbootURL := baseURL + "/netboot"

	return &AssetURLs{
		Kernel:  netbootURL + "/vmlinuz-virt",
		Initrd:  netbootURL + "/initramfs-virt",
		Rootfs:  fmt.Sprintf("%s/alpine-minirootfs-%s.0-%s.tar.gz", baseURL, p.version, alpineArch),
	}, nil
}

// BootConfig returns the kernel boot configuration for Alpine.
func (p *AlpineProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		Cmdline:       "console=hvc0 root=/dev/vda rw rootfstype=ext4 modules=virtio_blk",
		RootDevice:    "/dev/vda",
		RootFSType:    "ext4",
		ConsoleDevice: "hvc0",
		ExtraModules:  "virtio_blk",
	}
}

// SetupRequirements returns setup requirements for Alpine.
func (p *AlpineProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "ext4",
		NeedsExtraction: true,
	}
}

// toAlpineArch converts our arch to Alpine's arch naming.
func (p *AlpineProvider) toAlpineArch(arch Arch) string {
	switch arch {
	case ArchAMD64:
		return "x86_64"
	case ArchARM64:
		return "aarch64"
	default:
		return ""
	}
}

func init() {
	Register(NewAlpineProvider())
}
