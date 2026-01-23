package distro

import "fmt"

const (
	openSUSEVersion = "15.6"
	openSUSEBaseURL = "https://download.opensuse.org/distribution/leap"
)

// OpenSUSEProvider implements Provider for OpenSUSE Leap.
type OpenSUSEProvider struct {
	BaseProvider
}

// NewOpenSUSEProvider creates a new OpenSUSE provider.
func NewOpenSUSEProvider() *OpenSUSEProvider {
	return &OpenSUSEProvider{
		BaseProvider: BaseProvider{
			id:      OpenSUSE,
			name:    "OpenSUSE Leap",
			version: openSUSEVersion,
			archs:   []Arch{ArchAMD64, ArchARM64},
		},
	}
}

// AssetURLs returns download URLs for OpenSUSE.
// OpenSUSE provides cloud images with kernel inside.
func (p *OpenSUSEProvider) AssetURLs(arch Arch) (*AssetURLs, error) {
	if !p.SupportsArch(arch) {
		return nil, &ErrUnsupportedArch{Distro: p.id, Arch: arch}
	}

	suseArch := p.toSUSEArch(arch)

	// OpenSUSE provides JeOS images (Just Enough OS)
	return &AssetURLs{
		Kernel:  "", // Extracted from rootfs
		Initrd:  "", // Extracted from rootfs
		Rootfs:  fmt.Sprintf("%s/%s/appliances/openSUSE-Leap-%s-Minimal-VM.%s-Cloud.qcow2", openSUSEBaseURL, p.version, p.version, suseArch),
	}, nil
}

// BootConfig returns the kernel boot configuration for OpenSUSE.
func (p *OpenSUSEProvider) BootConfig(arch Arch) *BootConfig {
	return &BootConfig{
		// OpenSUSE uses btrfs by default
		Cmdline:       "console=hvc0 root=/dev/vda rw rootfstype=btrfs",
		RootDevice:    "/dev/vda",
		RootFSType:    "btrfs",
		ConsoleDevice: "hvc0",
		ExtraModules:  "",
	}
}

// SetupRequirements returns setup requirements for OpenSUSE.
func (p *OpenSUSEProvider) SetupRequirements() *SetupRequirements {
	return &SetupRequirements{
		NeedsFormatting: true,
		FSType:          "btrfs",
		NeedsExtraction: true,
	}
}

// toSUSEArch converts our arch to SUSE's arch naming.
func (p *OpenSUSEProvider) toSUSEArch(arch Arch) string {
	switch arch {
	case ArchAMD64:
		return "x86_64"
	case ArchARM64:
		return "aarch64"
	default:
		return ""
	}
}

// NeedsKernelExtraction returns true because OpenSUSE kernel is inside rootfs.
func (p *OpenSUSEProvider) NeedsKernelExtraction() bool {
	return true
}

func init() {
	Register(NewOpenSUSEProvider())
}
