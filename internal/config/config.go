package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all VMTerminal configuration.
type Config struct {
	// VMName is the name of the VM instance.
	VMName string `mapstructure:"vm_name"`

	// Distro is the Linux distribution to use.
	Distro string `mapstructure:"distro"`

	// CPUs is the number of virtual CPUs allocated to the VM.
	CPUs int `mapstructure:"cpus"`

	// MemoryMB is the amount of RAM in megabytes allocated to the VM.
	MemoryMB int `mapstructure:"memory_mb"`

	// DiskSizeMB is the disk image size in megabytes.
	DiskSizeMB int `mapstructure:"disk_size_mb"`

	// DiskPath is the path to the VM disk image.
	DiskPath string `mapstructure:"disk_path"`

	// SharedDirs are host directories mounted inside the VM.
	SharedDirs []string `mapstructure:"shared_dirs"`

	// EnableNetwork enables VM networking (NAT mode on macOS).
	EnableNetwork bool `mapstructure:"enable_network"`

	// MACAddress is an optional custom MAC address (empty = auto-generate).
	MACAddress string `mapstructure:"mac_address"`

	// SSHUser is the default username for SSH connections.
	SSHUser string `mapstructure:"ssh_user"`

	// SSHPort is the SSH port inside the VM.
	SSHPort int `mapstructure:"ssh_port"`

	// SSHKeyPath is the path to the SSH private key for authentication.
	SSHKeyPath string `mapstructure:"ssh_key_path"`

	// SSHHostPort is the host port for SSH port forwarding (0 = disabled).
	SSHHostPort int `mapstructure:"ssh_host_port"`

	// VMIP is the IP address of the VM for SSH connections.
	VMIP string `mapstructure:"vm_ip"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	paths, err := GetPaths()
	if err != nil {
		// Fallback if we can't determine home directory
		paths = &Paths{
			DataDir: "/tmp/vmterminal",
		}
	}

	home, _ := os.UserHomeDir()
	sharedDirs := []string{}
	if home != "" {
		sharedDirs = append(sharedDirs, home)
	}

	return &Config{
		VMName:        "vmterminal",
		Distro:        "alpine",
		CPUs:          runtime.NumCPU(),
		MemoryMB:      2048,
		DiskSizeMB:    10240, // 10GB
		DiskPath:      filepath.Join(paths.DataDir, "disk.img"),
		SharedDirs:    sharedDirs,
		EnableNetwork: true,
		MACAddress:    "",
		SSHUser:       "root",
		SSHPort:       22,
		SSHKeyPath:    "",
		SSHHostPort:   2222,
		VMIP:          "",
	}
}

// Global holds the loaded configuration.
var Global *Config

// Load reads configuration from file, environment, and defaults.
func Load() error {
	paths, err := GetPaths()
	if err != nil {
		return fmt.Errorf("failed to determine paths: %w", err)
	}

	// Set defaults
	defaults := DefaultConfig()
	viper.SetDefault("vm_name", defaults.VMName)
	viper.SetDefault("distro", defaults.Distro)
	viper.SetDefault("cpus", defaults.CPUs)
	viper.SetDefault("memory_mb", defaults.MemoryMB)
	viper.SetDefault("disk_size_mb", defaults.DiskSizeMB)
	viper.SetDefault("disk_path", defaults.DiskPath)
	viper.SetDefault("shared_dirs", defaults.SharedDirs)
	viper.SetDefault("enable_network", defaults.EnableNetwork)
	viper.SetDefault("mac_address", defaults.MACAddress)
	viper.SetDefault("ssh_user", defaults.SSHUser)
	viper.SetDefault("ssh_port", defaults.SSHPort)
	viper.SetDefault("ssh_key_path", defaults.SSHKeyPath)
	viper.SetDefault("ssh_host_port", defaults.SSHHostPort)
	viper.SetDefault("vm_ip", defaults.VMIP)

	// Config file settings
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(paths.DataDir)
	viper.AddConfigPath(paths.ConfigDir)

	// Environment variable support: VMTERMINAL_VM_NAME, VMTERMINAL_CPUS, etc.
	viper.SetEnvPrefix("VMTERMINAL")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (optional - not an error if missing)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found is OK - we use defaults
	}

	// Unmarshal into struct
	Global = &Config{}
	if err := viper.Unmarshal(Global); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// ConfigFileUsed returns the path of the config file being used, if any.
func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}
