package distro

import "fmt"

const (
	debianVersion  = "12"
	debianCodename = "bookworm"
	debianBaseURL  = "https://cloud.debian.org/images/cloud"
)

// DebianProvider implements Provider for Debian.
type DebianProvider struct {
	BaseProvider
}

// NewDebianProvider creates a new Debian provider.
func NewDebianProvider() *DebianProvider {
	return &DebianProvider{
		BaseProvider: BaseProvider{
			id:      Debian,
			name:    "Debian",
			version: debianVersion,
			archs:   []Arch{ArchAMD64, ArchARM64},
		},
	}
}

// AssetURLs returns download URLs for Debian.
// Debian cloud images include kernel inside the rootfs.
func (p *DebianProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	debianArch := p.toDebianArch(arch)

	// Debian provides rootfs tarballs in genericcloud images
	return &AssetURLs{
		Kernel:  "", // Extracted from rootfs
		Initrd:  "", // Extracted from rootfs
		Rootfs:  fmt.Sprintf("%s/%s/latest/debian-%s-genericcloud-%s.tar.xz", debianBaseURL, debianCodename, debianVersion, debianArch),
	}, nil
}

// BootConfig returns the kernel boot configuration for Debian.
func (p *DebianProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		Cmdline:       "console=hvc0 root=/dev/vda rw rootfstype=ext4",
		RootDevice:    "/dev/vda",
		RootFSType:    "ext4",
		ConsoleDevice: "hvc0",
		ExtraModules:  "",
	}
}

// SetupRequirements returns setup requirements for Debian.
func (p *DebianProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "ext4",
		NeedsExtraction: true,
	}
}

// toDebianArch converts our arch to Debian's arch naming.
func (p *DebianProvider) toDebianArch(arch Arch) string {
	switch arch {
	case ArchAMD64:
		return "amd64"
	case ArchARM64:
		return "arm64"
	default:
		return ""
	}
}

// NeedsKernelExtraction returns true because Debian kernel is inside rootfs.
func (p *DebianProvider) NeedsKernelExtraction() bool {
	return true
}

func init() {
	Register(NewDebianProvider())
}
