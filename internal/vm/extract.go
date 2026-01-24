package vm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/javanstorm/vmterminal/internal/distro"
)

// KernelExtractor handles extracting kernel and initrd from various archive formats.
type KernelExtractor struct {
	cacheDir string
}

// NewKernelExtractor creates a new kernel extractor with the given cache directory.
func NewKernelExtractor(cacheDir string) *KernelExtractor {
	return &KernelExtractor{cacheDir: cacheDir}
}

// ExtractKernel extracts kernel and initrd from an archive.
// Returns paths to the extracted kernel and initrd files.
func (e *KernelExtractor) ExtractKernel(archivePath string, locator *distro.KernelLocator) (kernelPath, initrdPath string, err error) {
	switch locator.ArchiveType {
	case "tarball":
		return e.extractFromTarball(archivePath, locator)
	case "qcow2":
		return e.extractFromQcow2(archivePath, locator)
	case "iso":
		return e.extractFromISO(archivePath, locator)
	default:
		return "", "", fmt.Errorf("unsupported archive type: %s", locator.ArchiveType)
	}
}

// extractFromTarball handles .tar.gz, .tar.xz, .tar.zst archives.
func (e *KernelExtractor) extractFromTarball(archivePath string, locator *distro.KernelLocator) (string, string, error) {
	// List archive contents
	files, err := e.listTarball(archivePath)
	if err != nil {
		return "", "", fmt.Errorf("list archive: %w", err)
	}

	// Find kernel
	kernelFile := e.findMatchingFile(files, locator.KernelPatterns)
	if kernelFile == "" {
		return "", "", fmt.Errorf("kernel not found in archive (tried: %v)", locator.KernelPatterns)
	}

	// Find initrd
	initrdFile := e.findMatchingFile(files, locator.InitrdPatterns)
	if initrdFile == "" {
		return "", "", fmt.Errorf("initrd not found in archive (tried: %v)", locator.InitrdPatterns)
	}

	// Extract kernel
	kernelDest := filepath.Join(e.cacheDir, "vmlinuz")
	if err := e.extractTarFile(archivePath, kernelFile, kernelDest); err != nil {
		return "", "", fmt.Errorf("extract kernel: %w", err)
	}

	// Extract initrd
	initrdDest := filepath.Join(e.cacheDir, "initramfs")
	if err := e.extractTarFile(archivePath, initrdFile, initrdDest); err != nil {
		return "", "", fmt.Errorf("extract initrd: %w", err)
	}

	return kernelDest, initrdDest, nil
}

// listTarball lists all files in a tarball archive.
func (e *KernelExtractor) listTarball(archivePath string) ([]string, error) {
	var cmd *exec.Cmd

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		cmd = exec.Command("tar", "-tzf", archivePath)
	case strings.HasSuffix(archivePath, ".tar.xz"):
		cmd = exec.Command("tar", "-tJf", archivePath)
	case strings.HasSuffix(archivePath, ".tar.zst"):
		cmd = exec.Command("tar", "--zstd", "-tf", archivePath)
	default:
		cmd = exec.Command("tar", "-tf", archivePath)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasSuffix(line, "/") {
			// Remove leading ./ if present
			line = strings.TrimPrefix(line, "./")
			files = append(files, line)
		}
	}

	return files, nil
}

// extractTarFile extracts a single file from a tarball to destination.
func (e *KernelExtractor) extractTarFile(archive, file, dest string) error {
	var cmd *exec.Cmd

	switch {
	case strings.HasSuffix(archive, ".tar.gz") || strings.HasSuffix(archive, ".tgz"):
		cmd = exec.Command("tar", "-xzf", archive, "-O", file)
	case strings.HasSuffix(archive, ".tar.xz"):
		cmd = exec.Command("tar", "-xJf", archive, "-O", file)
	case strings.HasSuffix(archive, ".tar.zst"):
		cmd = exec.Command("tar", "--zstd", "-xf", archive, "-O", file)
	default:
		cmd = exec.Command("tar", "-xf", archive, "-O", file)
	}

	outFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}

	return nil
}

// extractFromQcow2 handles .qcow2 images using guestfish or qemu-nbd.
func (e *KernelExtractor) extractFromQcow2(archivePath string, locator *distro.KernelLocator) (string, string, error) {
	// Ensure dependencies are installed
	if err := EnsureQcow2Deps(); err != nil {
		return "", "", fmt.Errorf("install dependencies: %w", err)
	}

	// Prefer guestfish (simpler API, doesn't require sudo)
	if _, err := exec.LookPath("guestfish"); err == nil {
		return e.extractWithGuestfish(archivePath, locator)
	}

	// Fallback to qemu-nbd + mount (requires sudo)
	if _, err := exec.LookPath("qemu-nbd"); err == nil {
		return e.extractWithNBD(archivePath, locator)
	}

	return "", "", fmt.Errorf("qcow2 extraction requires guestfish (libguestfs-tools) or qemu-nbd (qemu-utils)")
}

// extractWithGuestfish extracts kernel/initrd using libguestfs.
// Uses the -i flag for automatic inspection and mounting of filesystems.
func (e *KernelExtractor) extractWithGuestfish(qcow2Path string, locator *distro.KernelLocator) (string, string, error) {
	// Use guestfish with -i flag for auto-inspection and mounting
	// This is more reliable than manual mount commands which can fail silently
	cmd := exec.Command("guestfish", "--ro", "-a", qcow2Path, "-i", "ls", "/boot")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("guestfish ls /boot failed: %w: %s", err, stderr.String())
	}

	// Parse output to find files - ls /boot returns just filenames
	files := strings.Split(stdout.String(), "\n")
	var bootFiles []string
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			// Add boot/ prefix since ls /boot returns just filenames
			bootFiles = append(bootFiles, "boot/"+f)
		}
	}

	// Find matching kernel and initrd
	kernelFile := e.findMatchingFile(bootFiles, locator.KernelPatterns)
	if kernelFile == "" {
		return "", "", fmt.Errorf("kernel not found in qcow2 (tried: %v, found: %v)", locator.KernelPatterns, bootFiles)
	}

	initrdFile := e.findMatchingFile(bootFiles, locator.InitrdPatterns)
	if initrdFile == "" {
		return "", "", fmt.Errorf("initrd not found in qcow2 (tried: %v, found: %v)", locator.InitrdPatterns, bootFiles)
	}

	// Extract kernel
	kernelDest := filepath.Join(e.cacheDir, "vmlinuz")
	if err := e.guestfishCopyOut(qcow2Path, "/"+kernelFile, kernelDest); err != nil {
		return "", "", fmt.Errorf("extract kernel: %w", err)
	}

	// Extract initrd
	initrdDest := filepath.Join(e.cacheDir, "initramfs")
	if err := e.guestfishCopyOut(qcow2Path, "/"+initrdFile, initrdDest); err != nil {
		return "", "", fmt.Errorf("extract initrd: %w", err)
	}

	return kernelDest, initrdDest, nil
}

// guestfishCopyOut copies a file out of a qcow2 image using guestfish.
// Uses the -i flag for automatic inspection and mounting of filesystems.
func (e *KernelExtractor) guestfishCopyOut(qcow2Path, srcPath, destPath string) error {
	destDir := filepath.Dir(destPath)
	cmd := exec.Command("guestfish", "--ro", "-a", qcow2Path, "-i",
		"copy-out", srcPath, destDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("guestfish copy-out failed: %w: %s", err, stderr.String())
	}

	// guestfish copy-out puts the file with its original name, rename to our dest
	extractedPath := filepath.Join(destDir, filepath.Base(srcPath))
	if extractedPath != destPath {
		return os.Rename(extractedPath, destPath)
	}
	return nil
}

// extractWithNBD extracts kernel/initrd using qemu-nbd.
// This is a fallback that requires sudo.
func (e *KernelExtractor) extractWithNBD(qcow2Path string, locator *distro.KernelLocator) (string, string, error) {
	// This method requires root privileges and is more complex
	// It's provided as a fallback when guestfish is not available
	return "", "", fmt.Errorf("qemu-nbd extraction not yet implemented; please install libguestfs-tools (guestfish)")
}

// extractFromISO handles .iso images using bsdtar or 7z.
func (e *KernelExtractor) extractFromISO(archivePath string, locator *distro.KernelLocator) (string, string, error) {
	// Ensure dependencies are installed
	if err := EnsureISODeps(); err != nil {
		return "", "", fmt.Errorf("install dependencies: %w", err)
	}

	// List ISO contents
	files, err := e.listISO(archivePath)
	if err != nil {
		return "", "", fmt.Errorf("list ISO: %w", err)
	}

	// Find kernel
	kernelFile := e.findMatchingFile(files, locator.KernelPatterns)
	if kernelFile == "" {
		return "", "", fmt.Errorf("kernel not found in ISO (tried: %v)", locator.KernelPatterns)
	}

	// Find initrd
	initrdFile := e.findMatchingFile(files, locator.InitrdPatterns)
	if initrdFile == "" {
		return "", "", fmt.Errorf("initrd not found in ISO (tried: %v)", locator.InitrdPatterns)
	}

	// Extract kernel
	kernelDest := filepath.Join(e.cacheDir, "vmlinuz")
	if err := e.extractISOFile(archivePath, kernelFile, kernelDest); err != nil {
		return "", "", fmt.Errorf("extract kernel: %w", err)
	}

	// Extract initrd
	initrdDest := filepath.Join(e.cacheDir, "initramfs")
	if err := e.extractISOFile(archivePath, initrdFile, initrdDest); err != nil {
		return "", "", fmt.Errorf("extract initrd: %w", err)
	}

	return kernelDest, initrdDest, nil
}

// listISO lists files in an ISO image.
func (e *KernelExtractor) listISO(isoPath string) ([]string, error) {
	// Try bsdtar first
	if _, err := exec.LookPath("bsdtar"); err == nil {
		cmd := exec.Command("bsdtar", "-tf", isoPath)
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		var files []string
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasSuffix(line, "/") {
				files = append(files, line)
			}
		}
		return files, nil
	}

	// Try 7z
	if _, err := exec.LookPath("7z"); err == nil {
		cmd := exec.Command("7z", "l", "-slt", isoPath)
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		var files []string
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "Path = ") {
				path := strings.TrimPrefix(line, "Path = ")
				path = strings.TrimSpace(path)
				if path != "" && path != isoPath {
					files = append(files, path)
				}
			}
		}
		return files, nil
	}

	return nil, fmt.Errorf("ISO listing requires bsdtar (libarchive) or 7z (p7zip)")
}

// extractISOFile extracts a single file from an ISO.
func (e *KernelExtractor) extractISOFile(isoPath, file, dest string) error {
	// Try bsdtar first
	if _, err := exec.LookPath("bsdtar"); err == nil {
		outFile, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer outFile.Close()

		cmd := exec.Command("bsdtar", "-xOf", isoPath, file)
		cmd.Stdout = outFile
		return cmd.Run()
	}

	// Try 7z
	if _, err := exec.LookPath("7z"); err == nil {
		tmpDir, err := os.MkdirTemp("", "vmterminal-iso-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		cmd := exec.Command("7z", "x", "-o"+tmpDir, isoPath, file)
		if err := cmd.Run(); err != nil {
			return err
		}

		extractedPath := filepath.Join(tmpDir, file)
		return os.Rename(extractedPath, dest)
	}

	return fmt.Errorf("ISO extraction requires bsdtar (libarchive) or 7z (p7zip)")
}

// findMatchingFile finds the first file matching any pattern in the list.
// Returns the match with the highest version number if multiple matches exist.
func (e *KernelExtractor) findMatchingFile(files []string, patterns []string) string {
	for _, pattern := range patterns {
		var matches []string
		for _, f := range files {
			// Try matching both with and without leading path separators
			name := strings.TrimPrefix(f, "/")
			name = strings.TrimPrefix(name, "./")

			if matched, _ := filepath.Match(pattern, name); matched {
				matches = append(matches, f)
			}
			// Also try matching just the basename for patterns without path
			if !strings.Contains(pattern, "/") {
				if matched, _ := filepath.Match(pattern, filepath.Base(name)); matched {
					matches = append(matches, f)
				}
			}
		}
		if len(matches) > 0 {
			// Sort to get highest version (assumes version numbers sort correctly)
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			return matches[0]
		}
	}
	return ""
}
