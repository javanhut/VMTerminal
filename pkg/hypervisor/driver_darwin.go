//go:build darwin

package hypervisor

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/Code-Hex/vz/v3"
)

// vzDriver implements Driver using macOS Virtualization.framework.
type vzDriver struct {
	mu         sync.Mutex
	cfg        *VMConfig
	vm         *vz.VirtualMachine
	vmCfg      *vz.VirtualMachineConfiguration
	state      driverState
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

// NewDriver creates a new vz-based driver for macOS.
func NewDriver() (Driver, error) {
	return &vzDriver{
		state: stateNew,
	}, nil
}

func (d *vzDriver) Info() Info {
	return Info{
		Name:    "vz",
		Version: "1.0.0",
		Arch:    runtime.GOARCH,
	}
}

func (d *vzDriver) Validate(ctx context.Context, cfg *VMConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	// vz-specific validation could go here
	return nil
}

func (d *vzDriver) Create(ctx context.Context, cfg *VMConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateNew {
		return fmt.Errorf("vzDriver: invalid state for Create")
	}

	// Create boot loader
	bootLoader, err := vz.NewLinuxBootLoader(cfg.Kernel,
		vz.WithCommandLine(cfg.Cmdline),
		vz.WithInitrd(cfg.Initrd),
	)
	if err != nil {
		return fmt.Errorf("vzDriver: create boot loader: %w", err)
	}

	// Create VM configuration
	vmCfg, err := vz.NewVirtualMachineConfiguration(
		bootLoader,
		uint(cfg.CPUs),
		uint64(cfg.MemoryMB)*1024*1024,
	)
	if err != nil {
		return fmt.Errorf("vzDriver: create VM config: %w", err)
	}

	// Set platform configuration
	platform, err := vz.NewGenericPlatformConfiguration()
	if err != nil {
		return fmt.Errorf("vzDriver: create platform config: %w", err)
	}
	vmCfg.SetPlatformVirtualMachineConfiguration(platform)

	// Create pipes for console I/O
	// inputReader is read by VM (we write to inputWriter)
	// outputWriter is written by VM (we read from outputReader)
	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("vzDriver: create input pipe: %w", err)
	}
	outputReader, outputWriter, err := os.Pipe()
	if err != nil {
		inputReader.Close()
		inputWriter.Close()
		return fmt.Errorf("vzDriver: create output pipe: %w", err)
	}

	// Store console handles for Console() method
	d.consoleIn = inputWriter
	d.consoleOut = outputReader
	d.inputWriter = inputWriter
	d.outputReader = outputReader

	// Create serial console for I/O with file handles
	serialCfg, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(
		vz.NewFileHandleSerialPortAttachment(inputReader, outputWriter),
	)
	if err != nil {
		return fmt.Errorf("vzDriver: create serial config: %w", err)
	}
	vmCfg.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		serialCfg,
	})

	// Add network device if enabled
	if cfg.EnableNetwork {
		// Create NAT attachment for network
		natAttachment, err := vz.NewNATNetworkDeviceAttachment()
		if err != nil {
			return fmt.Errorf("vzDriver: create NAT attachment: %w", err)
		}

		// Create network device configuration
		netConfig, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
		if err != nil {
			return fmt.Errorf("vzDriver: create network config: %w", err)
		}

		// Set MAC address (auto-generate or use provided)
		var macAddr *vz.MACAddress
		if cfg.MACAddress != "" {
			hwAddr, err := net.ParseMAC(cfg.MACAddress)
			if err != nil {
				return fmt.Errorf("vzDriver: parse MAC address: %w", err)
			}
			macAddr, err = vz.NewMACAddress(hwAddr)
			if err != nil {
				return fmt.Errorf("vzDriver: create MAC address: %w", err)
			}
		} else {
			macAddr, err = vz.NewRandomLocallyAdministeredMACAddress()
			if err != nil {
				return fmt.Errorf("vzDriver: generate random MAC: %w", err)
			}
		}
		netConfig.SetMACAddress(macAddr)

		vmCfg.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{netConfig})
	}

	// Add disk if specified
	if cfg.DiskPath != "" {
		diskAttachment, err := vz.NewDiskImageStorageDeviceAttachment(cfg.DiskPath, false)
		if err != nil {
			return fmt.Errorf("vzDriver: create disk attachment: %w", err)
		}
		blockDevice, err := vz.NewVirtioBlockDeviceConfiguration(diskAttachment)
		if err != nil {
			return fmt.Errorf("vzDriver: create block device: %w", err)
		}
		vmCfg.SetStorageDevicesVirtualMachineConfiguration([]vz.StorageDeviceConfiguration{blockDevice})
	}

	// Add shared directories via virtio-fs
	if len(cfg.SharedDirs) > 0 {
		var fsDevices []vz.DirectorySharingDeviceConfiguration

		for tag, hostPath := range cfg.SharedDirs {
			readOnly := cfg.SharedDirsReadOnly[tag]

			sharedDir, err := vz.NewSharedDirectory(hostPath, readOnly)
			if err != nil {
				return fmt.Errorf("vzDriver: create shared dir %s: %w", tag, err)
			}

			dirShare, err := vz.NewSingleDirectoryShare(sharedDir)
			if err != nil {
				return fmt.Errorf("vzDriver: create dir share %s: %w", tag, err)
			}

			fsConfig, err := vz.NewVirtioFileSystemDeviceConfiguration(tag)
			if err != nil {
				return fmt.Errorf("vzDriver: create fs config %s: %w", tag, err)
			}
			fsConfig.SetDirectoryShare(dirShare)

			fsDevices = append(fsDevices, fsConfig)
		}

		vmCfg.SetDirectorySharingDevicesVirtualMachineConfiguration(fsDevices)
	}

	// Validate configuration
	ok, err := vmCfg.Validate()
	if !ok || err != nil {
		return fmt.Errorf("vzDriver: invalid configuration: %w", err)
	}

	// Create VM
	vm, err := vz.NewVirtualMachine(vmCfg)
	if err != nil {
		return fmt.Errorf("vzDriver: create VM: %w", err)
	}

	d.cfg = cfg
	d.vmCfg = vmCfg
	d.vm = vm
	d.state = stateCreated

	return nil
}

func (d *vzDriver) Start(ctx context.Context) (chan error, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateCreated && d.state != stateStopped {
		return nil, ErrNotCreated
	}

	errCh := make(chan error, 1)

	if err := d.vm.Start(); err != nil {
		return nil, fmt.Errorf("vzDriver: start VM: %w", err)
	}

	d.state = stateRunning

	// Monitor VM state in background
	go func() {
		<-d.vm.StateChangedNotify()
		state := d.vm.State()
		if state == vz.VirtualMachineStateStopped || state == vz.VirtualMachineStateError {
			d.mu.Lock()
			d.state = stateStopped
			d.mu.Unlock()
			errCh <- nil
		}
	}()

	return errCh, nil
}

func (d *vzDriver) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateRunning {
		return ErrNotRunning
	}

	canStop, err := d.vm.CanRequestStop()
	if err != nil {
		return fmt.Errorf("vzDriver: check can stop: %w", err)
	}

	if canStop {
		ok, err := d.vm.RequestStop()
		if err != nil || !ok {
			return fmt.Errorf("vzDriver: request stop failed: %w", err)
		}
	}

	d.state = stateStopped
	return nil
}

func (d *vzDriver) Kill(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != stateRunning {
		return ErrNotRunning
	}

	if err := d.vm.Stop(); err != nil {
		return fmt.Errorf("vzDriver: force stop: %w", err)
	}

	d.state = stateStopped
	return nil
}

func (d *vzDriver) Console() (io.Writer, io.Reader, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.consoleIn == nil || d.consoleOut == nil {
		return nil, nil, fmt.Errorf("vzDriver: console not initialized")
	}

	return d.consoleIn, d.consoleOut, nil
}

func (d *vzDriver) CloseConsole() error {
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
		return fmt.Errorf("vzDriver: close console: %v", errs)
	}
	return nil
}

func (d *vzDriver) Capabilities() Capabilities {
	return Capabilities{
		SharedDirs: true,  // virtio-fs supported
		Networking: true,  // virtio-net supported
		Snapshots:  false, // Not yet implemented
	}
}
