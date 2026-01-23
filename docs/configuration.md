# Configuration

VMTerminal can be configured through a YAML config file or environment variables.

## Config File Location

The config file is located at `~/.vmterminal/config.yaml`.

## Configuration Options

### Example Config File

```yaml
# VM Settings
vm_name: default
distro: alpine
cpus: 4
memory_mb: 4096
disk_size_mb: 20480

# Host Directories to Share
shared_dirs:
  - /Users/username
  - /Users/username/projects

# Networking
enable_network: true
mac_address: ""  # Leave empty for auto-generated

# SSH Configuration
ssh_user: root
ssh_port: 22
ssh_key_path: ~/.ssh/vmterminal
ssh_host_port: 2222
vm_ip: ""  # Set after finding VM's IP
```

### All Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `vm_name` | string | `vmterminal` | Default VM instance name |
| `distro` | string | `alpine` | Linux distribution |
| `cpus` | int | Host CPU count | Number of virtual CPUs |
| `memory_mb` | int | `2048` | RAM in megabytes |
| `disk_size_mb` | int | `10240` | Disk size in megabytes |
| `disk_path` | string | `~/.vmterminal/data/disk.img` | Path to disk image |
| `shared_dirs` | list | `[~]` | Host directories to mount |
| `enable_network` | bool | `true` | Enable VM networking |
| `mac_address` | string | (auto) | Custom MAC address |
| `ssh_user` | string | `root` | SSH username |
| `ssh_port` | int | `22` | SSH port in VM |
| `ssh_key_path` | string | (none) | Path to SSH private key |
| `ssh_host_port` | int | `2222` | Host port for SSH forwarding |
| `vm_ip` | string | (none) | VM IP address for SSH |

## Environment Variables

All config options can be set via environment variables with the `VMTERMINAL_` prefix:

```bash
export VMTERMINAL_CPUS=4
export VMTERMINAL_MEMORY_MB=4096
export VMTERMINAL_DISTRO=alpine
export VMTERMINAL_ENABLE_NETWORK=true
export VMTERMINAL_VM_IP=192.168.64.2
```

Environment variables override config file settings.

## Shared Directories

On macOS, you can share host directories with the VM using virtio-fs:

```yaml
shared_dirs:
  - /Users/username
  - /Users/username/projects
```

Inside the VM, mount them with:

```bash
mkdir -p /mnt/home
mount -t virtiofs share0 /mnt/home

mkdir -p /mnt/projects
mount -t virtiofs share1 /mnt/projects
```

The tags are `share0`, `share1`, etc., corresponding to the order in the config.

## Networking Configuration

### Enabling Network

Networking is enabled by default on macOS:

```yaml
enable_network: true
```

The VM uses NAT networking and can access the internet through the host.

### Finding VM IP

After the VM boots with networking enabled, find its IP:

```bash
# Inside the VM
ip addr show eth0
```

Then configure it:

```yaml
vm_ip: 192.168.64.2  # Your VM's IP
```

### MAC Address

By default, a random MAC address is generated. You can specify a fixed one:

```yaml
mac_address: "02:00:00:00:00:01"
```

## SSH Configuration

See [SSH Setup](ssh-setup.md) for detailed SSH configuration.

Basic SSH config:

```yaml
ssh_user: root
ssh_port: 22
ssh_key_path: ~/.ssh/vmterminal
vm_ip: 192.168.64.2
```

## Per-VM Configuration

Each VM has its own settings stored in `~/.vmterminal/vms.json`. Use `vmterminal vm create` flags to set per-VM options:

```bash
vmterminal vm create dev --cpus 8 --memory 16384 --disk-size 102400
```

The global config file provides defaults; per-VM settings override them.
