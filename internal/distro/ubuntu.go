package distro

import "fmt"

const (
	ubuntuVersion  = "24.04"
	ubuntuCodename = "noble"
	ubuntuBaseURL  = "https://cloud-images.ubuntu.com"
)

// UbuntuProvider implements Provider for Ubuntu.
type UbuntuProvider struct {
	BaseProvider
}

// NewUbuntuProvider creates a new Ubuntu provider.
func NewUbuntuProvider() *UbuntuProvider {
	return &UbuntuProvider{
		BaseProvider: BaseProvider{
			id:      Ubuntu,
			name:    "Ubuntu",
			version: ubuntuVersion,
			archs:   []Arch{ArchAMD64, ArchARM64},
		},
	}
}

// AssetURLs returns download URLs for Ubuntu.
// Ubuntu cloud images include kernel and initrd inside the rootfs,
// so we need to extract them after downloading.
func (p *UbuntuProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	ubuntuArch := p.toUbuntuArch(arch)

	// Ubuntu provides root.tar.xz which contains the rootfs
	// Kernel and initrd are inside /boot/ in the rootfs
	return &AssetURLs{
		// Ubuntu cloud images need kernel extracted from rootfs
		// We'll use their -root.tar.xz which is a rootfs tarball
		Kernel:  "", // Extracted from rootfs
		Initrd:  "", // Extracted from rootfs
		Rootfs:  fmt.Sprintf("%s/%s/current/%s-server-cloudimg-%s-root.tar.xz", ubuntuBaseURL, ubuntuCodename, ubuntuCodename, ubuntuArch),
	}, nil
}

// BootConfig returns the kernel boot configuration for Ubuntu.
func (p *UbuntuProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		Cmdline:       "console=hvc0 root=/dev/vda rw rootfstype=ext4",
		RootDevice:    "/dev/vda",
		RootFSType:    "ext4",
		ConsoleDevice: "hvc0",
		ExtraModules:  "",
	}
}

// SetupRequirements returns setup requirements for Ubuntu.
func (p *UbuntuProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "ext4",
		NeedsExtraction: true,
	}
}

// toUbuntuArch converts our arch to Ubuntu's arch naming.
func (p *UbuntuProvider) toUbuntuArch(arch Arch) string {
	switch arch {
	case ArchAMD64:
		return "amd64"
	case ArchARM64:
		return "arm64"
	default:
		return ""
	}
}

// NeedsKernelExtraction returns true because Ubuntu kernel is inside rootfs.
func (p *UbuntuProvider) NeedsKernelExtraction() bool {
	return true
}

func init() {
	Register(NewUbuntuProvider())
}
