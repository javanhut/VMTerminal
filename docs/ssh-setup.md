# SSH Setup

This guide explains how to set up SSH access to your VMTerminal VM.

## Overview

SSH access allows you to:
- Connect to the VM from other terminals
- Run commands remotely
- Use `vmterminal pkg` for package management
- Use `vmterminal docker` and `vmterminal podman` for containers

## Quick Setup

For guided setup, run:

```bash
vmterminal ssh-setup
```

This shows step-by-step instructions for your system.

## Step-by-Step Setup

### 1. Start the VM with Networking

Ensure networking is enabled (default on macOS):

```yaml
# ~/.vmterminal/config.yaml
enable_network: true
```

Start the VM:

```bash
vmterminal run
```

### 2. Install and Start SSH Server (Inside VM)

In the VM console:

```bash
# Install OpenSSH
apk add openssh

# Generate host keys
ssh-keygen -A

# Start SSH daemon
rc-service sshd start

# Enable SSH on boot
rc-update add sshd
```

### 3. Find the VM's IP Address

Inside the VM:

```bash
ip addr show eth0
```

Look for the `inet` line, e.g., `192.168.64.2`.

### 4. Configure VMTerminal

Add the VM IP to your config:

```yaml
# ~/.vmterminal/config.yaml
vm_ip: 192.168.64.2
ssh_user: root
ssh_port: 22
```

Or set via environment:

```bash
export VMTERMINAL_VM_IP=192.168.64.2
```

### 5. Set Up SSH Keys (Recommended)

Generate a key pair for passwordless access:

```bash
vmterminal ssh-keygen
```

This creates `~/.ssh/vmterminal` and `~/.ssh/vmterminal.pub`.

Copy the public key to the VM. Inside the VM:

```bash
mkdir -p ~/.ssh
echo "YOUR_PUBLIC_KEY" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Configure the key path:

```yaml
# ~/.vmterminal/config.yaml
ssh_key_path: ~/.ssh/vmterminal
```

### 6. Test Connection

```bash
vmterminal ssh
```

## Configuration Reference

| Option | Default | Description |
|--------|---------|-------------|
| `vm_ip` | (none) | VM IP address - required for SSH |
| `ssh_user` | `root` | Username for SSH |
| `ssh_port` | `22` | SSH port in VM |
| `ssh_key_path` | (none) | Path to private key |
| `ssh_host_port` | `2222` | Host port for forwarding |

## Connecting

Once configured:

```bash
# Basic connection
vmterminal ssh

# Pass arguments to ssh
vmterminal ssh -L 8080:localhost:80  # Port forwarding
vmterminal ssh ls -la                 # Run command
```

## Using SSH Features

### Package Management

With SSH configured, you can manage packages:

```bash
vmterminal pkg install git vim
vmterminal pkg update
vmterminal pkg upgrade
```

### Container Commands

Run Docker or Podman in the VM:

```bash
vmterminal docker ps
vmterminal docker run -it alpine sh

vmterminal podman run -it alpine sh
```

## Troubleshooting

### "VM IP not configured"

Set `vm_ip` in your config:

```yaml
vm_ip: 192.168.64.2
```

### "Connection refused"

1. Check VM is running: `vmterminal status`
2. Check SSH is running in VM: `rc-service sshd status`
3. Verify the IP is correct: `ip addr` in VM

### "Permission denied"

1. Check username is correct (`ssh_user`)
2. If using keys, verify:
   - Key path is correct (`ssh_key_path`)
   - Public key is in VM's `~/.ssh/authorized_keys`
   - Permissions: `chmod 600 ~/.ssh/authorized_keys`

3. For password auth, ensure root login is allowed:
   ```bash
   # In /etc/ssh/sshd_config
   PermitRootLogin yes
   ```

### "Host key verification failed"

VMTerminal disables strict host checking by default. If you see this, try:

```bash
ssh-keygen -R 192.168.64.2  # Remove old key
```

## Security Notes

- VMTerminal uses `-o StrictHostKeyChecking=no` for convenience
- For production use, consider proper host key management
- Use SSH keys instead of passwords when possible
- The VM's SSH host keys persist across restarts

## Alternative: Direct SSH

You can also use regular ssh:

```bash
ssh -i ~/.ssh/vmterminal root@192.168.64.2
```

The `vmterminal ssh` command is a convenience wrapper that uses your config.
