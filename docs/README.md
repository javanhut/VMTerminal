# VMTerminal Documentation

VMTerminal runs a Linux VM as your default shell on macOS and Linux. Open your terminal and you're in Linux — seamless, fast, with full access to host files.

## Quick Start

```bash
# Create and set up your first VM
vmterminal vm create myvm
sudo vmterminal setup

# Run the VM
vmterminal run
```

## Documentation

- [Getting Started](getting-started.md) - Installation and first steps
- [Commands Reference](commands.md) - Full command documentation
- [Configuration](configuration.md) - Config file and environment variables
- [Multi-VM Management](multi-vm.md) - Working with multiple VMs
- [SSH Setup](ssh-setup.md) - SSH access configuration
- [Containers](containers.md) - Docker and Podman support
- [Snapshots](snapshots.md) - Disk snapshots and rollback

## Features

- **Native hypervisors**: Uses Virtualization.framework on macOS and KVM on Linux
- **Seamless shell**: Set as your login shell to drop directly into Linux
- **Host file access**: Mount host directories inside the VM (macOS)
- **Networking**: NAT networking with internet access (macOS)
- **Multi-VM support**: Create and manage multiple VM instances
- **SSH access**: Connect to VMs via SSH
- **Container support**: Run Docker or Podman inside VMs
- **Snapshots**: Save and restore VM disk state

## Requirements

- **macOS**: macOS 12+ with Apple Silicon or Intel
- **Linux**: Kernel with KVM support
- **Go**: 1.21+ (for building from source)

## Platform Support

| Feature | macOS | Linux |
|---------|-------|-------|
| VM Execution | Yes | Yes |
| Networking | NAT | Planned |
| Filesystem Sharing | virtio-fs | Planned |
| SSH Access | Yes | Yes |
