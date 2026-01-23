# Snapshots

VMTerminal supports disk snapshots for backup and rollback.

## Overview

Snapshots capture the entire VM disk state, allowing you to:
- Save state before risky operations
- Roll back after failed upgrades
- Create restore points
- Duplicate VM configurations

Snapshots are compressed with gzip for efficient storage.

## Creating Snapshots

Create a snapshot of the current VM disk:

```bash
# Basic snapshot
vmterminal snapshot create my-snapshot

# With description
vmterminal snapshot create before-upgrade -d "Before system upgrade"

# For a specific VM
vmterminal snapshot create backup --vm dev
```

**Note:** Creating snapshots works whether the VM is running or stopped. However, for consistency, it's recommended to stop the VM first.

## Listing Snapshots

View all snapshots for a VM:

```bash
# List for active VM
vmterminal snapshot list

# List for specific VM
vmterminal snapshot list --vm dev
```

Output:
```
Snapshots for VM 'default':
  before-upgrade
    Created: 2026-01-23 10:30:45
    Description: Before system upgrade
    Original size: 1024.00 MB
    Compressed size: 256.00 MB
  clean-install
    Created: 2026-01-22 14:15:00
    Original size: 512.00 MB
    Compressed size: 128.00 MB
```

## Viewing Snapshot Details

Get detailed information about a snapshot:

```bash
vmterminal snapshot show before-upgrade
```

Output:
```
Snapshot: before-upgrade
  VM: default
  Created: 2026-01-23 10:30:45
  Description: Before system upgrade
  Original disk size: 1024.00 MB
  Compressed size: 256.00 MB
  Compression ratio: 25.0%
```

## Restoring Snapshots

Restore a VM to a previous snapshot state:

```bash
vmterminal snapshot restore before-upgrade
```

**Important:** The VM must be stopped before restoring.

```bash
# Stop VM if running
vmterminal stop

# Restore
vmterminal snapshot restore before-upgrade

# Start VM
vmterminal run
```

**Warning:** Restoring overwrites the current disk. Any changes since the snapshot will be lost.

## Deleting Snapshots

Remove a snapshot to free disk space:

```bash
vmterminal snapshot delete old-snapshot

# For specific VM
vmterminal snapshot delete old-snapshot --vm dev
```

## Storage

Snapshots are stored per-VM:

```
~/.vmterminal/
└── data/
    └── <vm-name>/
        ├── disk.raw           # Current disk
        ├── snapshots.json     # Snapshot metadata
        └── snapshots/
            ├── before-upgrade.raw.gz
            └── clean-install.raw.gz
```

## Compression

Snapshots use gzip compression:

- Sparse disk areas compress very well
- Typical compression: 4:1 to 10:1
- A 10GB disk might compress to 1-2GB

The compression happens automatically when creating snapshots.

## Best Practices

### When to Create Snapshots

- Before system upgrades: `vmterminal pkg upgrade`
- Before installing new software
- Before configuration changes
- Before risky experiments
- After completing a stable setup

### Naming Convention

Use descriptive names:

```bash
vmterminal snapshot create clean-install
vmterminal snapshot create pre-docker-setup
vmterminal snapshot create working-2026-01-23
```

### Regular Cleanup

Snapshots consume disk space. Periodically clean up old ones:

```bash
vmterminal snapshot list
vmterminal snapshot delete outdated-snapshot
```

## Workflow Examples

### Safe System Upgrade

```bash
# Create safety snapshot
vmterminal snapshot create pre-upgrade -d "Before apk upgrade"

# Perform upgrade
vmterminal run
# Inside VM: apk upgrade

# If something breaks:
vmterminal stop
vmterminal snapshot restore pre-upgrade
vmterminal run
```

### Experiment Safely

```bash
# Save current state
vmterminal snapshot create baseline

# Try something risky
vmterminal run
# Experiment...

# Didn't work? Restore
vmterminal stop
vmterminal snapshot restore baseline
```

### Clone VM Configuration

```bash
# Create a snapshot of well-configured VM
vmterminal snapshot create configured --vm production

# Create new VM
vmterminal vm create staging --cpus 4 --memory 4096

# Copy snapshot file (manual)
cp ~/.vmterminal/data/production/snapshots/configured.raw.gz \
   ~/.vmterminal/data/staging/snapshots/

# Restore in new VM
vmterminal snapshot restore configured --vm staging
```

## Limitations

- Snapshots capture disk only (not memory state)
- VM should be stopped for consistent snapshots
- Large disks take longer to snapshot/restore
- No incremental snapshots (each is a full copy)

## Troubleshooting

### "VM is running"

Stop the VM before restoring:

```bash
vmterminal stop
vmterminal snapshot restore my-snapshot
```

### Slow Snapshot Creation

Large disks take time. The progress is shown:

```
Creating snapshot 'backup' for VM 'default'...
This may take a while depending on disk size...
Snapshot created: backup (2048.00 MB compressed)
```

Consider using smaller disks for VMs that need frequent snapshots.

### Disk Space Issues

Check available space:

```bash
df -h ~/.vmterminal
```

Delete old snapshots to free space:

```bash
vmterminal snapshot list
vmterminal snapshot delete old-snapshot
```
