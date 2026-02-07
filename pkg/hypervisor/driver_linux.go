//go:build linux

package hypervisor

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	hypeos "github.com/c35s/hype/os/linux"
	"github.com/c35s/hype/virtio"
	"github.com/c35s/hype/vmm"
)

// kvmDriver implements Driver using Linux KVM via hype.
type kvmDriver struct {
	mu         sync.Mutex
	cfg        *VMConfig
	vm         *vmm.VM
	state      driverState
	cancel     context.CancelFunc
	diskFile   *os.File
	consoleIn  io.Writer // Write to this to send to VM
	consoleOut io.Reader // Read from this to get VM output
	// Raw pipe handles for closing
	inputWriter  *os.File
	outputReader *os.File
}

type driverState int

const (
	stateNew driverState = iota
	stateCreated
	stateRunning
	stateStopped
)

// NewDriver creates a new KVM-based driver for Linux.
func NewDriver() (Driver, error) {
	// Check if /dev/kvm exists and is accessible
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return nil, fmt.Errorf("kvmDriver: /dev/kvm not accessible: %w", err)
	}
	return &kvmDriver{
		state: stateNew,
	}, nil
}

func (d *kvmDriver) Info() Info {
	return Info{
		Name:    "kvm",
		Version: "1.0.0",
		Arch:    runtime.GOARCH,
	}
}

func (d *kvmDriver) Validate(ctx context.Context, cfg *VMConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	// Check kernel file exists
	if _, err := os.Stat(cfg.Kernel); err != nil {
		return fmt.Errorf("kvmDriver: kernel not found: %w", err)
	}
	return nil
}

func (d *kvmDriver) Create(ctx context.Context, cfg *VMConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateNew {
		return fmt.Errorf("kvmDriver: invalid state for Create")
	}

	// Read kernel
	kernel, err := os.ReadFile(cfg.Kernel)
	if err != nil {
		return fmt.Errorf("kvmDriver: read kernel: %w", err)
	}

	// Read initrd if specified
	var initrd []byte
	if cfg.Initrd != "" {
		initrd, err = os.ReadFile(cfg.Initrd)
		if err != nil {
			return fmt.Errorf("kvmDriver: read initrd: %w", err)
		}
	}

	// Create pipes for console I/O
	// inputReader is read by VM (we write to inputWriter)
	// outputWriter is written by VM (we read from outputReader)
	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("kvmDriver: create input pipe: %w", err)
	}
	outputReader, outputWriter, err := os.Pipe()
	if err != nil {
		inputReader.Close()
		inputWriter.Close()
		return fmt.Errorf("kvmDriver: create output pipe: %w", err)
	}

	// Store console handles for Console() method
	d.consoleIn = inputWriter
	d.consoleOut = outputReader
	d.inputWriter = inputWriter
	d.outputReader = outputReader

	// Build hype configuration
	hypeCfg := vmm.Config{
		MemSize: int(cfg.MemoryMB) * 1024 * 1024,
		Devices: []virtio.DeviceConfig{
			&virtio.ConsoleDevice{
				In:  inputReader,
				Out: outputWriter,
			},
		},
		Loader: &hypeos.Loader{
			Kernel:  kernel,
			Initrd:  initrd,
			Cmdline: cfg.Cmdline,
		},
	}

	// Add block device if disk path specified
	if cfg.DiskPath != "" {
		diskFile, err := os.OpenFile(cfg.DiskPath, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("kvmDriver: open disk: %w", err)
		}
		hypeCfg.Devices = append(hypeCfg.Devices, &virtio.BlockDevice{
			Storage: &virtio.FileStorage{File: diskFile},
		})
		d.diskFile = diskFile
	}

	// Note: Warnings about unsupported SharedDirs and Networking are now
	// handled earlier by config.ValidateConfig() in the run command.

	// Create the VM (but don't run yet)
	vm, err := vmm.New(hypeCfg)
	if err != nil {
		if d.diskFile != nil {
			d.diskFile.Close()
			d.diskFile = nil
		}
		return fmt.Errorf("kvmDriver: create VM: %w", err)
	}

	d.cfg = cfg
	d.vm = vm
	d.state = stateCreated

	return nil
}

func (d *kvmDriver) Start(ctx context.Context) (chan error, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateCreated && d.state != stateStopped {
		return nil, ErrNotCreated
	}

	errCh := make(chan error, 1)
	startedCh := make(chan struct{})
	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	// Run VM in background
	go func() {
		// Lock OS thread for VCPU operations as per research
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		// Signal that goroutine has started before running VM
		close(startedCh)

		err := d.vm.Run(runCtx)
		d.mu.Lock()
		d.state = stateStopped
		d.mu.Unlock()
		errCh <- err
	}()

	// Wait for goroutine to actually start before setting state
	<-startedCh
	d.state = stateRunning

	return errCh, nil
}

func (d *kvmDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateRunning {
		return ErrNotRunning
	}

	if d.cancel != nil {
		d.cancel()
	}
	d.state = stateStopped
	return nil
}

func (d *kvmDriver) Kill(ctx context.Context) error {
	// For KVM, Kill is the same as Stop (context cancellation)
	err := d.Stop(ctx)

	// Close disk file if open
	d.mu.Lock()
	if d.diskFile != nil {
		d.diskFile.Close()
		d.diskFile = nil
	}
	d.mu.Unlock()

	return err
}

func (d *kvmDriver) Console() (io.Writer, io.Reader, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.consoleIn == nil || d.consoleOut == nil {
		return nil, nil, fmt.Errorf("kvmDriver: console not initialized")
	}

	return d.consoleIn, d.consoleOut, nil
}

func (d *kvmDriver) CloseConsole() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var errs []error

	if d.inputWriter != nil {
		if err := d.inputWriter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close input pipe: %w", err))
		}
		d.inputWriter = nil
		d.consoleIn = nil
	}

	if d.outputReader != nil {
		if err := d.outputReader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close output pipe: %w", err))
		}
		d.outputReader = nil
		d.consoleOut = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("kvmDriver: close console: %v", errs)
	}
	return nil
}

func (d *kvmDriver) Capabilities() Capabilities {
	return Capabilities{
		SharedDirs: false, // hype lacks virtio-fs/9p
		Networking: false, // hype lacks virtio-net
		Snapshots:  false, // Not implemented
	}
}
