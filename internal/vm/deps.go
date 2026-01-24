package vm

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Dependency represents a required external tool.
type Dependency struct {
	Name        string            // Tool name (e.g., "guestfish")
	Command     string            // Command to check (e.g., "guestfish")
	Packages    map[string]string // OS -> package name mapping
	Description string            // Human-readable description
}

// DependencyManager handles checking and installing dependencies.
type DependencyManager struct {
	hostOS string // "arch", "ubuntu", "debian", "fedora", "macos", etc.
}

// NewDependencyManager creates a dependency manager for the current host.
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{
		hostOS: detectHostOS(),
	}
}

// Required dependencies for qcow2 extraction.
var qcow2Deps = []Dependency{
	{
		Name:        "guestfish",
		Command:     "guestfish",
		Description: "Extract files from qcow2 disk images",
		Packages: map[string]string{
			"arch":       "libguestfs",
			"manjaro":    "libguestfs",
			"endeavouros": "libguestfs",
			"ubuntu":     "libguestfs-tools",
			"debian":     "libguestfs-tools",
			"linuxmint":  "libguestfs-tools",
			"pop":        "libguestfs-tools",
			"fedora":     "libguestfs-tools-c",
			"rhel":       "libguestfs-tools-c",
			"centos":     "libguestfs-tools-c",
			"rocky":      "libguestfs-tools-c",
			"almalinux":  "libguestfs-tools-c",
			"opensuse":   "libguestfs",
			"suse":       "libguestfs",
			"macos":      "", // Not available on macOS
		},
	},
}

// Required dependencies for ISO extraction.
var isoDeps = []Dependency{
	{
		Name:        "bsdtar",
		Command:     "bsdtar",
		Description: "Extract files from ISO images",
		Packages: map[string]string{
			"arch":       "libarchive",
			"manjaro":    "libarchive",
			"endeavouros": "libarchive",
			"ubuntu":     "libarchive-tools",
			"debian":     "libarchive-tools",
			"linuxmint":  "libarchive-tools",
			"pop":        "libarchive-tools",
			"fedora":     "bsdtar",
			"rhel":       "bsdtar",
			"centos":     "bsdtar",
			"rocky":      "bsdtar",
			"almalinux":  "bsdtar",
			"opensuse":   "libarchive",
			"suse":       "libarchive",
			"macos":      "libarchive", // brew install libarchive
		},
	},
}

// detectHostOS returns the host OS family.
func detectHostOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}

	// Read /etc/os-release for Linux distros
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux"
	}

	content := string(data)

	// Check ID field
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, "\"")
			return id
		}
	}

	// Check ID_LIKE for derivatives
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike := strings.TrimPrefix(line, "ID_LIKE=")
			idLike = strings.Trim(idLike, "\"")
			// Return first parent distro
			if strings.Contains(idLike, "arch") {
				return "arch"
			}
			if strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
				return "debian"
			}
			if strings.Contains(idLike, "fedora") || strings.Contains(idLike, "rhel") {
				return "fedora"
			}
		}
	}

	return "linux"
}

// CheckDependency checks if a dependency is installed.
func (m *DependencyManager) CheckDependency(dep Dependency) bool {
	_, err := exec.LookPath(dep.Command)
	return err == nil
}

// InstallDependency installs a dependency using the appropriate package manager.
func (m *DependencyManager) InstallDependency(dep Dependency) error {
	pkg, ok := dep.Packages[m.hostOS]
	if !ok || pkg == "" {
		return fmt.Errorf("%s is not available on %s", dep.Name, m.hostOS)
	}

	var cmd *exec.Cmd

	switch m.hostOS {
	case "arch", "manjaro", "endeavouros":
		cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)
	case "ubuntu", "debian", "linuxmint", "pop":
		cmd = exec.Command("sudo", "apt-get", "install", "-y", pkg)
	case "fedora":
		cmd = exec.Command("sudo", "dnf", "install", "-y", pkg)
	case "rhel", "centos", "rocky", "almalinux":
		cmd = exec.Command("sudo", "yum", "install", "-y", pkg)
	case "opensuse", "suse":
		cmd = exec.Command("sudo", "zypper", "install", "-y", pkg)
	case "macos":
		cmd = exec.Command("brew", "install", pkg)
	default:
		return fmt.Errorf("unsupported host OS: %s (install %s manually)", m.hostOS, pkg)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Installing %s (%s)...\n", dep.Name, pkg)
	return cmd.Run()
}

// EnsureDependencies checks and installs required dependencies.
func (m *DependencyManager) EnsureDependencies(deps []Dependency) error {
	var missing []Dependency

	for _, dep := range deps {
		if !m.CheckDependency(dep) {
			missing = append(missing, dep)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	fmt.Printf("Installing missing dependencies for %s...\n", m.hostOS)

	for _, dep := range missing {
		if err := m.InstallDependency(dep); err != nil {
			return fmt.Errorf("install %s: %w", dep.Name, err)
		}
	}

	return nil
}

// EnsureQcow2Deps ensures dependencies for qcow2 extraction are installed.
func EnsureQcow2Deps() error {
	dm := NewDependencyManager()
	return dm.EnsureDependencies(qcow2Deps)
}

// EnsureISODeps ensures dependencies for ISO extraction are installed.
func EnsureISODeps() error {
	dm := NewDependencyManager()
	return dm.EnsureDependencies(isoDeps)
}
