package hypervisor

import "errors"

// Configuration errors
var (
	ErrInvalidCPUCount    = errors.New("hypervisor: CPU count must be at least 1")
	ErrInsufficientMemory = errors.New("hypervisor: memory must be at least 128MB")
	ErrMissingKernel      = errors.New("hypervisor: kernel path is required")
	ErrInvalidNetworkMode = errors.New("hypervisor: network mode must be 'nat' or 'bridged'")
)

// Runtime errors
var (
	ErrNotCreated     = errors.New("hypervisor: VM not created")
	ErrAlreadyRunning = errors.New("hypervisor: VM is already running")
	ErrNotRunning     = errors.New("hypervisor: VM is not running")
)

// Platform errors
var (
	ErrUnsupportedPlatform = errors.New("hypervisor: platform not supported")
)
