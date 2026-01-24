package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/javanstorm/vmterminal/internal/distro"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cached assets (kernels, rootfs)",
	Long:  `Manage downloaded distro assets like kernels, initramfs, and rootfs images.`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear [distro]",
	Short: "Clear cached assets",
	Long: `Clear cached assets for a specific distro or all distros.

Examples:
  vmt cache clear              # Clear all cached assets
  vmt cache clear ubuntu       # Clear only Ubuntu cache
  vmt cache clear --disk       # Clear all cache and disk images
  vmt cache clear ubuntu --disk # Clear Ubuntu cache and reset disk`,
	RunE: runCacheClear,
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached assets",
	RunE:  runCacheList,
}

var cacheClearDisk bool

func init() {
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheClearCmd.Flags().BoolVar(&cacheClearDisk, "disk", false, "Also remove disk images (full reset)")
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	baseDir := filepath.Join(homeDir, ".vmterminal")
	cacheDir := filepath.Join(baseDir, "cache")
	dataDir := filepath.Join(baseDir, "data", "default")

	if len(args) > 0 {
		// Clear specific distro
		distroID := args[0]

		// Validate distro ID before using in path
		if !distro.IsRegistered(distro.ID(distroID)) {
			validIDs := distro.List()
			validNames := make([]string, len(validIDs))
			for i, id := range validIDs {
				validNames[i] = string(id)
			}
			return fmt.Errorf("unknown distro: %s (valid: %s)", distroID, strings.Join(validNames, ", "))
		}

		// Cache structure is <distro>/<version>/<arch>, so we clear the distro dir
		targetDir := filepath.Join(cacheDir, distroID)

		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			fmt.Printf("No cache found for %s\n", distroID)
		} else {
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("clear %s cache: %w", distroID, err)
			}
			fmt.Printf("Cleared cache for %s\n", distroID)
		}

		// Also clear disk if requested
		if cacheClearDisk {
			// Remove disk.raw and any rootfs.raw that might be in cache
			diskPath := filepath.Join(dataDir, "disk.raw")
			if _, err := os.Stat(diskPath); err == nil {
				if err := os.Remove(diskPath); err != nil {
					return fmt.Errorf("remove disk: %w", err)
				}
				fmt.Println("Removed disk image")
			}
			// Reset state
			statePath := filepath.Join(dataDir, "state.json")
			if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove state: %w", err)
			}
			fmt.Println("Reset VM state")
		}
	} else {
		// Clear all
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			fmt.Println("No cached assets to clear")
		} else {
			if err := os.RemoveAll(cacheDir); err != nil {
				return fmt.Errorf("clear cache: %w", err)
			}
			fmt.Println("Cleared all cached assets")
		}

		if cacheClearDisk {
			// Remove all disk images
			diskPath := filepath.Join(dataDir, "disk.raw")
			if _, err := os.Stat(diskPath); err == nil {
				if err := os.Remove(diskPath); err != nil {
					return fmt.Errorf("remove disk: %w", err)
				}
				fmt.Println("Removed disk image")
			}
			// Reset state
			statePath := filepath.Join(dataDir, "state.json")
			if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove state: %w", err)
			}
			fmt.Println("Reset VM state")
		}
	}

	return nil
}

func runCacheList(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	cacheDir := filepath.Join(homeDir, ".vmterminal", "cache")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Println("No cached assets")
		return nil
	}

	fmt.Println("Cached assets:")

	var totalSize int64
	found := false

	for _, d := range distro.AllDistros() {
		distroDir := filepath.Join(cacheDir, string(d))
		if _, err := os.Stat(distroDir); err == nil {
			size, _ := dirSize(distroDir)
			fmt.Printf("  %s: %s\n", d, formatSize(size))
			totalSize += size
			found = true
		}
	}

	if !found {
		fmt.Println("  (none)")
	} else {
		fmt.Printf("\nTotal: %s\n", formatSize(totalSize))
	}

	return nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// Return error instead of silently skipping
			return err
		}
		if info != nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
