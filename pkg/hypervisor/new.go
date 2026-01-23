package hypervisor

import "runtime"

// SupportedPlatform returns true if the current platform has a hypervisor driver.
func SupportedPlatform() bool {
	switch runtime.GOOS {
	case "darwin", "linux":
		return true
	default:
		return false
	}
}

// NewDriver creates a new hypervisor driver for the current platform.
// This function is implemented in platform-specific files using build tags.
// See driver_darwin.go and driver_linux.go.
