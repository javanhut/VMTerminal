# Container Support

VMTerminal supports running Docker and Podman containers inside VMs.

## Prerequisites

Before using containers, you need:

1. A running VM with networking enabled
2. SSH access configured (see [SSH Setup](ssh-setup.md))

## Docker

### Installation

View Docker installation instructions:

```bash
vmterminal docker-setup
```

Or install manually inside the VM:

```bash
# Install Docker
apk add docker

# Start Docker daemon
rc-service docker start

# Enable on boot
rc-update add docker

# (Optional) Add user to docker group
addgroup $USER docker
```

### Usage

Run Docker commands through VMTerminal:

```bash
# List containers
vmterminal docker ps

# Run a container
vmterminal docker run -it alpine sh

# Pull an image
vmterminal docker pull nginx

# Run in background
vmterminal docker run -d -p 8080:80 nginx

# View logs
vmterminal docker logs <container-id>

# Stop container
vmterminal docker stop <container-id>
```

All arguments after `vmterminal docker` are passed directly to the docker command inside the VM.

## Podman

### Installation

View Podman installation instructions:

```bash
vmterminal podman-setup
```

Or install manually inside the VM:

```bash
# Install Podman
apk add podman

# Configure for rootless (optional)
echo "$USER:100000:65536" | sudo tee /etc/subuid
echo "$USER:100000:65536" | sudo tee /etc/subgid
```

### Usage

Run Podman commands through VMTerminal:

```bash
# List containers
vmterminal podman ps

# Run a container
vmterminal podman run -it alpine sh

# Pull an image
vmterminal podman pull nginx

# Run rootless container
vmterminal podman run --userns=auto -d nginx
```

## Docker vs Podman

| Feature | Docker | Podman |
|---------|--------|--------|
| Daemon | Required | Daemonless |
| Root | Typically root | Rootless support |
| Compatibility | Industry standard | Docker CLI compatible |
| Compose | docker-compose | podman-compose |

Choose based on your needs:
- **Docker**: Better ecosystem, more documentation
- **Podman**: More secure (rootless), no daemon

## Accessing Container Services

When running services in containers, you can access them via the VM's IP:

```bash
# Run nginx in VM
vmterminal docker run -d -p 8080:80 nginx

# Access from host (if VM IP is 192.168.64.2)
curl http://192.168.64.2:8080
```

## Port Forwarding with SSH

You can forward ports through SSH:

```bash
# Forward local port 8080 to container port 80
vmterminal ssh -L 8080:localhost:80

# Now access at localhost:8080 from host
```

## Docker Compose

For Docker Compose, install it in the VM:

```bash
# Inside VM
apk add docker-compose
```

Then use via SSH or inside the VM directly.

## Persistent Data

Container data lives on the VM disk. Use volumes for persistence:

```bash
vmterminal docker run -v /data:/app/data myimage
```

If you need data to survive VM recreation, consider:
1. Mounting host directories (macOS only)
2. Using snapshots before major changes

## Troubleshooting

### "Cannot connect to Docker daemon"

```bash
# Check Docker is running
vmterminal ssh rc-service docker status

# Start Docker
vmterminal ssh rc-service docker start
```

### "VM IP not configured"

Set up SSH access first. See [SSH Setup](ssh-setup.md).

### "Permission denied"

For Docker, ensure the user is in the docker group or run as root.

For Podman rootless, configure subuid/subgid properly.

### Slow Image Pulls

Images are downloaded to the VM's disk. Ensure sufficient disk space:

```bash
vmterminal ssh df -h
```

Consider increasing disk size when creating the VM:

```bash
vmterminal vm create docker-vm --disk-size 51200  # 50GB
```
