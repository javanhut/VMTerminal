# VMTerminal

A terminal tool that runs a Linux VM as your default shell on macOS and Linux. Open your terminal and you're in Linux — seamless, fast, with full access to host files. Like WSL, but for non-Windows systems.

## Features

- **Native hypervisor integration** - Virtualization.framework on macOS, KVM on Linux
- **Seamless terminal shell** - Set as login shell to drop directly into Linux
- **Host filesystem mounting** - Access host files from within VM (macOS)
- **NAT networking** - Internet access out of the box (macOS)
- **Multi-VM support** - Create and manage multiple VM instances
- **SSH access** - Connect to VMs via SSH
- **Container support** - Run Docker or Podman inside VMs
- **Snapshots** - Save and restore VM disk state
- **Package management** - Install packages via `vmterminal pkg`

## Requirements

### macOS
- macOS 12.0+ (Monterey or later)
- Apple Silicon (M1/M2/M3) or Intel Mac

### Linux
- Linux kernel with KVM support
- `/dev/kvm` accessible (user in `kvm` group)

```bash
# Check KVM access
ls -la /dev/kvm

# If permission denied:
sudo usermod -aG kvm $USER
# Log out and back in
```

## Installation

### Homebrew (Recommended for macOS)

```bash
brew tap javanstorm/vmterminal
brew install vmterminal
```

### Binary Download

Download from [GitHub Releases](https://github.com/javanstorm/vmterminal/releases):

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/javanstorm/vmterminal/releases/latest/download/vmterminal_darwin_arm64.tar.gz
tar xzf vmterminal_darwin_arm64.tar.gz
sudo mv vmterminal /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/javanstorm/vmterminal/releases/latest/download/vmterminal_darwin_amd64.tar.gz
tar xzf vmterminal_darwin_amd64.tar.gz
sudo mv vmterminal /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/javanstorm/vmterminal/releases/latest/download/vmterminal_linux_amd64.tar.gz
tar xzf vmterminal_linux_amd64.tar.gz
sudo mv vmterminal /usr/local/bin/

# Linux (ARM64)
curl -LO https://github.com/javanstorm/vmterminal/releases/latest/download/vmterminal_linux_arm64.tar.gz
tar xzf vmterminal_linux_arm64.tar.gz
sudo mv vmterminal /usr/local/bin/
```

### From Source

```bash
# Clone the repository
git clone https://github.com/javanstorm/vmterminal.git
cd vmterminal

# Build and install
make install
```

### Aliases

VMTerminal installs with convenient aliases:
- `vmterminal` - Full name
- `vmt` - Short alias
- `vmterm` - Medium alias

## Quick Start

```bash
# Create a VM
vmterminal vm create default

# Set up the disk (requires root)
sudo vmterminal setup

# Run the VM
vmterminal run
```

Press `Ctrl+C` to stop the VM.

## Documentation

Full documentation is available in the [docs/](docs/) directory:

- [Getting Started](docs/getting-started.md) - Installation and first steps
- [Commands Reference](docs/commands.md) - All CLI commands
- [Configuration](docs/configuration.md) - Config file and environment variables
- [Multi-VM Management](docs/multi-vm.md) - Working with multiple VMs
- [SSH Setup](docs/ssh-setup.md) - SSH access configuration
- [Containers](docs/containers.md) - Docker and Podman support
- [Snapshots](docs/snapshots.md) - Disk snapshots and rollback

## Usage

### Core Commands

```bash
vmterminal run                  # Start VM and attach to console
vmterminal shell                # Start VM in shell mode
vmterminal status               # Show VM status
vmterminal stop                 # Stop running VM
```

### VM Management

```bash
vmterminal vm create dev --cpus 4 --memory 4096
vmterminal vm list              # List all VMs (* = active)
vmterminal vm use dev           # Switch active VM
vmterminal vm show              # Show VM details
vmterminal vm delete old --data # Delete VM and data
```

### SSH Access

```bash
vmterminal ssh-setup            # Show setup instructions
vmterminal ssh-keygen           # Generate SSH keys
vmterminal ssh                  # Connect via SSH
```

### Package Management

```bash
vmterminal pkg install git vim  # Install packages
vmterminal pkg search nginx     # Search packages
vmterminal pkg update           # Update package index
vmterminal pkg upgrade          # Upgrade all packages
vmterminal pkg list             # List installed packages
```

### Snapshots

```bash
vmterminal snapshot create backup -d "Before upgrade"
vmterminal snapshot list
vmterminal snapshot restore backup
vmterminal snapshot delete old-backup
```

### Containers

```bash
vmterminal docker-setup         # Show Docker install instructions
vmterminal docker ps            # Run Docker commands
vmterminal docker run -it alpine sh

vmterminal podman-setup         # Show Podman install instructions
vmterminal podman run -it alpine sh
```

## Configuration

Configuration file: `~/.vmterminal/config.yaml`

```yaml
# VM defaults
distro: alpine
cpus: 4
memory_mb: 4096
disk_size_mb: 20480

# Shared directories (macOS)
shared_dirs:
  - /Users/username

# Networking
enable_network: true

# SSH (after setup)
vm_ip: 192.168.64.2
ssh_user: root
ssh_key_path: ~/.ssh/vmterminal
```

Environment variables (override config):
```bash
export VMTERMINAL_CPUS=4
export VMTERMINAL_MEMORY_MB=4096
export VMTERMINAL_VM_IP=192.168.64.2
```

## Setting as Login Shell

```bash
# View instructions
vmterminal install

# Add to /etc/shells
sudo sh -c 'echo /usr/local/bin/vmterminal >> /etc/shells'

# Set as default shell
chsh -s /usr/local/bin/vmterminal
```

Now opening a terminal drops you into Linux. Revert with `chsh -s /bin/bash`.

## Directory Structure

```
~/.vmterminal/
├── config.yaml          # Configuration
├── vms.json             # VM registry
├── active               # Active VM name
├── cache/               # Shared assets (kernel, rootfs)
│   └── alpine/
└── data/                # Per-VM data
    └── <vm-name>/
        ├── disk.raw     # VM disk
        ├── state.json   # Persistent state
        └── snapshots/   # Disk snapshots
```

## Platform Support

| Feature | macOS | Linux |
|---------|-------|-------|
| VM Execution | Yes | Yes |
| NAT Networking | Yes | Planned |
| Filesystem Sharing | Yes (virtio-fs) | Planned |
| SSH Access | Yes | Yes |
| Containers | Yes | Yes |
| Snapshots | Yes | Yes |

## Architecture

```
cmd/vmterminal/          # Entry point
internal/
├── cli/                 # CLI commands (Cobra)
├── config/              # Configuration (Viper)
├── distro/              # Linux distribution providers
├── terminal/            # Terminal handling
└── vm/                  # VM lifecycle, registry, snapshots
pkg/
└── hypervisor/          # Hypervisor abstraction
    ├── driver_darwin.go # macOS (Virtualization.framework)
    └── driver_linux.go  # Linux (KVM via hype)
```

## Troubleshooting

### "kvmDriver: /dev/kvm not accessible"
```bash
sudo usermod -aG kvm $USER
# Log out and back in
```

### "VM disk is not set up"
```bash
sudo vmterminal setup
```

### "VM IP not configured" (for SSH/pkg commands)
1. Start VM: `vmterminal run`
2. Find IP inside VM: `ip addr show eth0`
3. Configure: Add `vm_ip: <IP>` to `~/.vmterminal/config.yaml`

### Console output is garbled
```bash
reset
```

## Development

```bash
# Build
go build -o vmterminal ./cmd/vmterminal

# Verify
go build ./...
go vet ./...
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/Code-Hex/vz/v3` - macOS Virtualization.framework
- `github.com/c35s/hype` - Linux KVM
- `golang.org/x/term` - Terminal handling

## License

MIT

## Acknowledgments

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Go bindings for Virtualization.framework
- [c35s/hype](https://github.com/c35s/hype) - Lightweight KVM hypervisor in Go
- [Alpine Linux](https://alpinelinux.org/) - Lightweight Linux distribution
