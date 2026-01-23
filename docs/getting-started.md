# Getting Started

This guide walks you through installing and running your first VMTerminal VM.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/javanstorm/vmterminal.git
cd vmterminal

# Build
go build -o vmterminal ./cmd/vmterminal

# Install (optional)
sudo mv vmterminal /usr/local/bin/
```

## First Run

### 1. Create a VM

Create your first VM with default settings:

```bash
vmterminal vm create default
```

Or customize the VM:

```bash
vmterminal vm create myvm --cpus 4 --memory 4096 --disk-size 20480
```

### 2. Set Up the Disk

The setup command downloads the Linux distribution and prepares the disk. This requires root privileges:

```bash
sudo vmterminal setup
```

This will:
- Download the Alpine Linux kernel and rootfs
- Create a disk image
- Format and extract the rootfs

### 3. Run the VM

Start the VM:

```bash
vmterminal run
```

You'll see the VM boot and get a login prompt. The default credentials for Alpine are:
- Username: `root`
- Password: (none - just press Enter)

Press `Ctrl+C` to stop the VM.

## Shell Mode

For a more seamless experience, use shell mode:

```bash
vmterminal shell
```

This starts the VM and attaches directly to the console.

## Setting as Login Shell

To make VMTerminal your default shell:

```bash
# View installation instructions
vmterminal install

# Add to /etc/shells (requires root)
sudo sh -c 'echo /usr/local/bin/vmterminal >> /etc/shells'

# Set as your shell
chsh -s /usr/local/bin/vmterminal
```

Now when you open a terminal, you'll automatically be in Linux.

To revert:
```bash
chsh -s /bin/bash
```

## Directory Structure

VMTerminal stores data in `~/.vmterminal/`:

```
~/.vmterminal/
├── config.yaml          # Configuration file
├── vms.json            # VM registry
├── active              # Active VM name
├── cache/              # Downloaded assets (kernel, rootfs)
│   └── alpine/
└── data/               # Per-VM data
    └── <vm-name>/
        ├── disk.raw    # VM disk image
        ├── state.json  # Persistent state
        └── snapshots/  # Disk snapshots
```

## Next Steps

- [Configure your VM](configuration.md)
- [Set up SSH access](ssh-setup.md)
- [Run containers](containers.md)
- [Create snapshots](snapshots.md)
