package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RootfsManager handles disk formatting and rootfs extraction.
type RootfsManager struct {
	dataDir string
}

// NewRootfsManager creates a new rootfs manager.
func NewRootfsManager(dataDir string) *RootfsManager {
	return &RootfsManager{dataDir: dataDir}
}

// SetupState represents the state of rootfs setup.
type SetupState struct {
	DiskExists      bool
	DiskFormatted   bool
	RootfsExtracted bool
	FSType          string
}

// CheckSetupState checks the current state of rootfs setup.
func (m *RootfsManager) CheckSetupState(diskName string) (*SetupState, error) {
	state := &SetupState{}
	diskPath := filepath.Join(m.dataDir, diskName+".raw")

	// Check if disk exists
	info, err := os.Stat(diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("stat disk: %w", err)
	}
	state.DiskExists = true

	// Check if disk has a filesystem by looking at its actual size
	// A sparse file that's been formatted will have allocated blocks
	if info.Size() > 0 {
		// Try to detect filesystem type using blkid
		fsType, err := m.detectFSType(diskPath)
		if err == nil && fsType != "" {
			state.DiskFormatted = true
			state.FSType = fsType
			state.RootfsExtracted = true // If formatted, assume extracted
		}
	}

	return state, nil
}

// detectFSType uses blkid to detect the filesystem type on a disk.
func (m *RootfsManager) detectFSType(diskPath string) (string, error) {
	cmd := exec.Command("blkid", "-o", "value", "-s", "TYPE", diskPath)
	output, err := cmd.Output()
	if err != nil {
		// blkid returns error if no filesystem found
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// FormatDisk formats the disk with the specified filesystem.
// Requires root privileges.
func (m *RootfsManager) FormatDisk(diskName, fsType string) error {
	diskPath := filepath.Join(m.dataDir, diskName+".raw")

	// Verify disk exists
	if _, err := os.Stat(diskPath); err != nil {
		return fmt.Errorf("disk not found: %w", err)
	}

	var cmd *exec.Cmd
	switch fsType {
	case "ext4":
		// -F forces creation even though it's not a real block device
		cmd = exec.Command("mkfs.ext4", "-F", "-L", "vmterminal", diskPath)
	case "xfs":
		cmd = exec.Command("mkfs.xfs", "-f", "-L", "vmterminal", diskPath)
	case "btrfs":
		cmd = exec.Command("mkfs.btrfs", "-f", "-L", "vmterminal", diskPath)
	default:
		return fmt.Errorf("unsupported filesystem type: %s", fsType)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("format disk: %w", err)
	}

	return nil
}

// ExtractRootfs extracts a rootfs tarball to the disk.
// This requires mounting the disk, which needs root privileges.
func (m *RootfsManager) ExtractRootfs(diskName, rootfsPath string) error {
	diskPath := filepath.Join(m.dataDir, diskName+".raw")

	// Create a temporary mount point
	mountPoint, err := os.MkdirTemp("", "vmterminal-mount-")
	if err != nil {
		return fmt.Errorf("create mount point: %w", err)
	}
	defer os.RemoveAll(mountPoint)

	// Mount the disk
	if err := m.mountDisk(diskPath, mountPoint); err != nil {
		return fmt.Errorf("mount disk: %w", err)
	}
	defer m.unmountDisk(mountPoint)

	// Determine extraction command based on archive type
	// sudo is required since the mount point is owned by root
	var extractCmd *exec.Cmd
	if strings.HasSuffix(rootfsPath, ".tar.gz") || strings.HasSuffix(rootfsPath, ".tgz") {
		extractCmd = exec.Command("sudo", "tar", "-xzf", rootfsPath, "-C", mountPoint)
	} else if strings.HasSuffix(rootfsPath, ".tar.xz") {
		extractCmd = exec.Command("sudo", "tar", "-xJf", rootfsPath, "-C", mountPoint)
	} else if strings.HasSuffix(rootfsPath, ".tar.zst") {
		// Arch Linux uses zstd compression
		// Bootstrap tarball has root.x86_64/ prefix that needs stripping
		extractCmd = exec.Command("sudo", "tar", "--zstd", "-xf", rootfsPath, "-C", mountPoint, "--strip-components=1")
	} else if strings.HasSuffix(rootfsPath, ".tar") {
		extractCmd = exec.Command("sudo", "tar", "-xf", rootfsPath, "-C", mountPoint)
	} else if strings.HasSuffix(rootfsPath, ".qcow2") {
		// For qcow2 images, we need to use qemu-img and then copy
		return m.extractQcow2(rootfsPath, mountPoint)
	} else {
		return fmt.Errorf("unsupported archive format: %s", rootfsPath)
	}

	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr

	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("extract rootfs: %w", err)
	}

	return nil
}

// mountDisk mounts a disk image to a mount point.
func (m *RootfsManager) mountDisk(diskPath, mountPoint string) error {
	cmd := exec.Command("sudo", "mount", "-o", "loop", diskPath, mountPoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// unmountDisk unmounts a disk from a mount point.
func (m *RootfsManager) unmountDisk(mountPoint string) error {
	cmd := exec.Command("sudo", "umount", mountPoint)
	return cmd.Run()
}

// extractQcow2 extracts contents from a qcow2 image.
// This requires qemu-nbd or libguestfs tools.
func (m *RootfsManager) extractQcow2(qcow2Path, mountPoint string) error {
	// First, try using guestfish if available
	if _, err := exec.LookPath("guestfish"); err == nil {
		return m.extractWithGuestfish(qcow2Path, mountPoint)
	}

	// Fall back to qemu-nbd
	if _, err := exec.LookPath("qemu-nbd"); err == nil {
		return m.extractWithNBD(qcow2Path, mountPoint)
	}

	return fmt.Errorf("qcow2 extraction requires guestfish or qemu-nbd; please install libguestfs-tools or qemu-utils")
}

// extractWithGuestfish uses libguestfs to extract qcow2 contents.
func (m *RootfsManager) extractWithGuestfish(qcow2Path, mountPoint string) error {
	// Use guestfish to copy files
	script := fmt.Sprintf(`
add %s
run
mount /dev/sda1 /
copy-out / %s
`, qcow2Path, mountPoint)

	cmd := exec.Command("guestfish")
	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// extractWithNBD uses qemu-nbd to extract qcow2 contents.
func (m *RootfsManager) extractWithNBD(qcow2Path, mountPoint string) error {
	// This is more complex and requires managing NBD device lifecycle
	// For now, return an error suggesting guestfish
	return fmt.Errorf("qemu-nbd extraction not yet implemented; please install libguestfs-tools and use guestfish")
}

// SetupDisk performs full disk setup: format and extract rootfs.
func (m *RootfsManager) SetupDisk(diskName, fsType, rootfsPath string) error {
	fmt.Printf("Formatting disk with %s filesystem...\n", fsType)
	if err := m.FormatDisk(diskName, fsType); err != nil {
		return err
	}

	fmt.Printf("Extracting rootfs from %s...\n", filepath.Base(rootfsPath))
	if err := m.ExtractRootfs(diskName, rootfsPath); err != nil {
		return err
	}

	fmt.Println("Disk setup complete.")
	return nil
}

// DiskPath returns the path to a disk image.
func (m *RootfsManager) DiskPath(diskName string) string {
	return filepath.Join(m.dataDir, diskName+".raw")
}
