package hypervisor

// VMConfig holds VM configuration parameters.
type VMConfig struct {
	// CPUs is the number of virtual CPUs.
	CPUs int

	// MemoryMB is the amount of memory in megabytes.
	MemoryMB int

	// Kernel is the path to the Linux kernel image.
	Kernel string

	// Initrd is the path to the initial ramdisk (optional).
	Initrd string

	// Cmdline is the kernel command line.
	Cmdline string

	// DiskPath is the path to the root disk image.
	DiskPath string

	// SharedDirs maps mount tags to host directory paths.
	// Key: mount tag (used by guest to mount via "mount -t virtiofs <tag> <mountpoint>")
	// Value: host directory path to share
	SharedDirs map[string]string

	// SharedDirsReadOnly specifies which shares are read-only.
	// Key: mount tag, Value: true if read-only
	SharedDirsReadOnly map[string]bool

	// EnableNetwork enables VM networking.
	EnableNetwork bool

	// NetworkMode specifies the network mode ("nat" or "bridged").
	// Currently only "nat" is supported.
	NetworkMode string

	// MACAddress is an optional custom MAC address.
	// If empty, a random locally-administered MAC will be generated.
	MACAddress string

	// PortForwards maps host ports to guest ports for NAT networking.
	// Key: host port, Value: guest port
	// Example: {2222: 22} forwards host:2222 to guest:22
	PortForwards map[int]int
}

// Validate performs basic validation of the configuration.
func (c *VMConfig) Validate() error {
	if c.CPUs < 1 {
		return ErrInvalidCPUCount
	}
	if c.MemoryMB < 128 {
		return ErrInsufficientMemory
	}
	if c.Kernel == "" {
		return ErrMissingKernel
	}
	// Validate network config if enabled
	if c.EnableNetwork {
		if c.NetworkMode == "" {
			c.NetworkMode = "nat" // Default to NAT
		}
		if c.NetworkMode != "nat" && c.NetworkMode != "bridged" {
			return ErrInvalidNetworkMode
		}
	}
	return nil
}
