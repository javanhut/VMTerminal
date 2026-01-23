# Commands Reference

Complete reference for all VMTerminal commands.

## Core Commands

### vmterminal run

Start and run the VM in the foreground.

```bash
vmterminal run [flags]
```

**Flags:**
- `-c, --cpus int` - Number of virtual CPUs
- `-m, --memory int` - Memory in MB
- `-d, --distro string` - Linux distribution to use
- `--vm string` - VM to run (default: active VM)

**Examples:**
```bash
# Run with default settings
vmterminal run

# Run with custom resources
vmterminal run --cpus 4 --memory 4096

# Run a specific VM
vmterminal run --vm myvm
```

### vmterminal shell

Start VM and attach to shell (seamless mode).

```bash
vmterminal shell [flags]
```

**Flags:**
- `--vm string` - VM to use (default: active VM)

**Example:**
```bash
vmterminal shell
```

### vmterminal setup

Set up the VM disk with the Linux rootfs. Requires root privileges.

```bash
sudo vmterminal setup [flags]
```

**Flags:**
- `-d, --distro string` - Linux distribution to set up
- `-f, --force` - Force setup even if already done
- `--vm string` - VM to set up (default: active VM)

**Example:**
```bash
sudo vmterminal setup --force
```

### vmterminal status

Show VM status.

```bash
vmterminal status [flags]
```

**Flags:**
- `--vm string` - VM to check (default: active VM)

### vmterminal stop

Stop a running VM.

```bash
vmterminal stop
```

### vmterminal install

Show instructions for setting VMTerminal as your login shell.

```bash
vmterminal install
```

### vmterminal version

Show version information.

```bash
vmterminal version
```

---

## VM Management

### vmterminal vm create

Create a new VM instance.

```bash
vmterminal vm create <name> [flags]
```

**Flags:**
- `-c, --cpus int` - Number of virtual CPUs (default: host CPU count)
- `-m, --memory int` - Memory in MB (default: 2048)
- `-s, --disk-size int` - Disk size in MB (default: 10240)
- `-d, --distro string` - Linux distribution (default: alpine)

**Example:**
```bash
vmterminal vm create dev --cpus 4 --memory 8192 --disk-size 51200
```

### vmterminal vm list

List all VMs. Active VM is marked with `*`.

```bash
vmterminal vm list
```

**Output:**
```
VMs:
  * default (alpine, 4 CPUs, 2048 MB)
    dev (alpine, 8 CPUs, 8192 MB)
```

### vmterminal vm use

Set a VM as the active (default) VM.

```bash
vmterminal vm use <name>
```

**Example:**
```bash
vmterminal vm use dev
```

### vmterminal vm show

Show VM details.

```bash
vmterminal vm show [name]
```

If no name is given, shows the active VM.

### vmterminal vm delete

Delete a VM.

```bash
vmterminal vm delete <name> [flags]
```

**Flags:**
- `--data` - Also delete VM data (disk, state, snapshots)

**Example:**
```bash
vmterminal vm delete oldvm --data
```

---

## SSH Commands

### vmterminal ssh

Connect to the VM via SSH.

```bash
vmterminal ssh [ssh-args...]
```

Requires `vm_ip` to be configured. Additional arguments are passed to the ssh command.

**Example:**
```bash
vmterminal ssh
vmterminal ssh -L 8080:localhost:80
```

### vmterminal ssh-keygen

Generate SSH keys for VM authentication.

```bash
vmterminal ssh-keygen [flags]
```

**Flags:**
- `-f, --file string` - Output file path
- `-t, --type string` - Key type (default: ed25519)

### vmterminal ssh-setup

Show comprehensive SSH setup instructions.

```bash
vmterminal ssh-setup
```

---

## Package Management

All `pkg` commands execute via SSH and require `vm_ip` to be configured.

### vmterminal pkg install

Install packages.

```bash
vmterminal pkg install <packages...> [flags]
```

**Flags:**
- `--vm string` - VM to target

**Example:**
```bash
vmterminal pkg install git vim curl
```

### vmterminal pkg remove

Remove packages.

```bash
vmterminal pkg remove <packages...>
```

### vmterminal pkg search

Search for packages.

```bash
vmterminal pkg search <query>
```

### vmterminal pkg update

Update the package index.

```bash
vmterminal pkg update
```

### vmterminal pkg upgrade

Upgrade all installed packages.

```bash
vmterminal pkg upgrade
```

### vmterminal pkg list

List installed packages.

```bash
vmterminal pkg list
```

---

## Snapshot Commands

### vmterminal snapshot create

Create a snapshot of the VM disk.

```bash
vmterminal snapshot create <name> [flags]
```

**Flags:**
- `-d, --description string` - Description for the snapshot
- `--vm string` - VM to snapshot

**Example:**
```bash
vmterminal snapshot create before-upgrade -d "Before system upgrade"
```

### vmterminal snapshot list

List all snapshots for a VM.

```bash
vmterminal snapshot list [flags]
```

**Flags:**
- `--vm string` - VM to list snapshots for

### vmterminal snapshot restore

Restore a VM from a snapshot. The VM must be stopped.

```bash
vmterminal snapshot restore <name> [flags]
```

**Flags:**
- `--vm string` - VM to restore

**Example:**
```bash
vmterminal snapshot restore before-upgrade
```

### vmterminal snapshot delete

Delete a snapshot.

```bash
vmterminal snapshot delete <name> [flags]
```

**Flags:**
- `--vm string` - VM owning the snapshot

### vmterminal snapshot show

Show snapshot details.

```bash
vmterminal snapshot show <name> [flags]
```

**Flags:**
- `--vm string` - VM owning the snapshot

---

## Container Commands

### vmterminal docker-setup

Show Docker installation instructions.

```bash
vmterminal docker-setup
```

### vmterminal docker

Run Docker commands in the VM via SSH.

```bash
vmterminal docker [docker-args...]
```

**Example:**
```bash
vmterminal docker ps
vmterminal docker run -it alpine sh
```

### vmterminal podman-setup

Show Podman installation instructions.

```bash
vmterminal podman-setup
```

### vmterminal podman

Run Podman commands in the VM via SSH.

```bash
vmterminal podman [podman-args...]
```

**Example:**
```bash
vmterminal podman run -it alpine sh
```
