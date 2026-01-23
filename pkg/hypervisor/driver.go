// Package hypervisor provides a unified interface for VM management
// across different hypervisor backends (macOS Virtualization.framework, Linux KVM).
package hypervisor

import (
	"context"
	"io"
)

// Driver is the main interface for hypervisor operations.
// Platform-specific implementations (vz, kvm) satisfy this interface.
type Driver interface {
	Lifecycle
	Info() Info
	// Console returns VM console I/O handles. Only valid after Start().
	Console() (in io.Writer, out io.Reader, err error)
}

// Lifecycle defines VM lifecycle operations.
type Lifecycle interface {
	// Validate checks if the configuration is valid for this driver.
	Validate(ctx context.Context, cfg *VMConfig) error

	// Create initializes VM resources without starting.
	Create(ctx context.Context, cfg *VMConfig) error

	// Start boots the VM. Returns a channel that receives an error when VM exits.
	Start(ctx context.Context) (chan error, error)

	// Stop gracefully shuts down the VM.
	Stop(ctx context.Context) error

	// Kill forcefully terminates the VM.
	Kill(ctx context.Context) error
}

// Info contains driver metadata.
type Info struct {
	Name    string // "vz" or "kvm"
	Version string // Driver version
	Arch    string // "arm64" or "amd64"
}
