# Multi-VM Management

VMTerminal supports running multiple VM instances with different configurations.

## Creating VMs

Create a new VM with `vm create`:

```bash
# Create with default settings
vmterminal vm create myvm

# Create with custom settings
vmterminal vm create dev --cpus 8 --memory 8192 --disk-size 51200

# Create with a specific distro
vmterminal vm create test --distro alpine
```

### Create Options

| Flag | Default | Description |
|------|---------|-------------|
| `--cpus, -c` | Host CPU count | Virtual CPUs |
| `--memory, -m` | 2048 | Memory in MB |
| `--disk-size, -s` | 10240 | Disk size in MB |
| `--distro, -d` | alpine | Linux distribution |

## Listing VMs

View all VMs:

```bash
vmterminal vm list
```

Output:
```
VMs:
  * default (alpine, 4 CPUs, 2048 MB)
    dev (alpine, 8 CPUs, 8192 MB)
    test (alpine, 2 CPUs, 1024 MB)
```

The `*` marks the active VM.

## Switching VMs

Set a different VM as active:

```bash
vmterminal vm use dev
```

Now all commands without `--vm` will use `dev`.

## Running a Specific VM

Use `--vm` to target a specific VM without changing the active VM:

```bash
# Run a specific VM
vmterminal run --vm test

# Set up a specific VM
sudo vmterminal setup --vm dev

# Check status of a specific VM
vmterminal status --vm test
```

## VM Details

View detailed information about a VM:

```bash
# Show active VM
vmterminal vm show

# Show specific VM
vmterminal vm show dev
```

Output:
```
VM: dev (active)
  Distro: alpine
  CPUs: 8
  Memory: 8192 MB
  Disk Size: 51200 MB
  Created: 2026-01-23 10:30:45
  Data Dir: /home/user/.vmterminal/data/dev
```

## Deleting VMs

Remove a VM from the registry:

```bash
# Remove from registry only (keeps data)
vmterminal vm delete oldvm

# Remove registry entry AND delete all data
vmterminal vm delete oldvm --data
```

The `--data` flag removes:
- Disk image
- State files
- Snapshots

## Data Isolation

Each VM has its own data directory:

```
~/.vmterminal/
├── vms.json              # VM registry
├── active                # Active VM name
└── data/
    ├── default/          # Default VM data
    │   ├── disk.raw
    │   ├── state.json
    │   └── snapshots/
    ├── dev/              # Dev VM data
    │   ├── disk.raw
    │   └── snapshots/
    └── test/             # Test VM data
        └── disk.raw
```

## Shared Cache

Asset downloads are shared across all VMs:

```
~/.vmterminal/
└── cache/
    └── alpine/           # Shared by all Alpine VMs
        ├── kernel
        ├── initramfs
        └── rootfs.tar.gz
```

This means setting up multiple VMs with the same distro only downloads assets once.

## Workflow Example

Here's a typical multi-VM workflow:

```bash
# Create VMs for different purposes
vmterminal vm create production --cpus 4 --memory 4096
vmterminal vm create development --cpus 8 --memory 8192
vmterminal vm create testing --cpus 2 --memory 2048

# Set up each VM
sudo vmterminal setup --vm production
sudo vmterminal setup --vm development
sudo vmterminal setup --vm testing

# Work primarily in development
vmterminal vm use development
vmterminal run

# Quick test in testing VM (doesn't change active)
vmterminal run --vm testing

# Create snapshot before deploying to production
vmterminal snapshot create pre-deploy --vm production
vmterminal run --vm production
```

## Commands Supporting --vm Flag

These commands accept `--vm` to target a specific VM:

- `vmterminal run --vm <name>`
- `vmterminal shell --vm <name>`
- `vmterminal setup --vm <name>`
- `vmterminal status --vm <name>`
- `vmterminal pkg * --vm <name>`
- `vmterminal snapshot * --vm <name>`
