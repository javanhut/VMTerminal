package distro

import (
	"fmt"
	"strings"
	"testing"
)

// Task 1: Provider configuration tests

func TestProviderAssetURLs(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				urls, err := p.AssetURLs(arch)
				if err != nil {
					t.Fatalf("AssetURLs(%q) failed: %v", arch, err)
				}
				if urls == nil {
					t.Fatal("AssetURLs returned nil")
				}
			})
		}
	}
}

func TestProviderBootConfig(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				bc := p.BootConfig(arch)
				if bc == nil {
					t.Fatal("BootConfig returned nil")
				}
				if bc.Cmdline == "" {
					t.Error("Cmdline should not be empty")
				}
				if bc.RootDevice == "" {
					t.Error("RootDevice should not be empty")
				}
				if bc.ConsoleDevice == "" {
					t.Error("ConsoleDevice should not be empty")
				}
			})
		}
	}
}

func TestProviderSetupRequirements(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		t.Run(string(id), func(t *testing.T) {
			sr := p.SetupRequirements()
			if sr == nil {
				t.Fatal("SetupRequirements returned nil")
			}
			// FSType should be set if formatting is needed
			if sr.NeedsFormatting && sr.FSType == "" {
				t.Error("FSType should be set when NeedsFormatting is true")
			}
		})
	}
}

func TestProviderCacheSubdir(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				subdir := p.CacheSubdir(arch)
				if subdir == "" {
					t.Error("CacheSubdir should not be empty")
				}

				// Should contain distro ID
				if !strings.Contains(subdir, string(id)) {
					t.Errorf("CacheSubdir %q should contain distro ID %q", subdir, id)
				}

				// Should contain version
				if !strings.Contains(subdir, p.Version()) {
					t.Errorf("CacheSubdir %q should contain version %q", subdir, p.Version())
				}

				// Should contain arch
				if !strings.Contains(subdir, string(arch)) {
					t.Errorf("CacheSubdir %q should contain arch %q", subdir, arch)
				}

				// Should be in format: distro/version/arch
				expected := fmt.Sprintf("%s/%s/%s", id, p.Version(), arch)
				if subdir != expected {
					t.Errorf("CacheSubdir = %q, want %q", subdir, expected)
				}
			})
		}
	}
}

func TestKernelLocatorPatterns(t *testing.T) {
	// Distros that use KernelLocator for extraction
	extractionDistros := []ID{Ubuntu, Debian, Rocky, OpenSUSE}

	for _, id := range extractionDistros {
		t.Run(string(id), func(t *testing.T) {
			p, err := Get(id)
			if err != nil {
				t.Fatalf("Get(%q) failed: %v", id, err)
			}

			loc := p.KernelLocator()
			if loc == nil {
				t.Error("expected KernelLocator for extraction distro")
				return
			}

			if len(loc.KernelPatterns) == 0 {
				t.Error("expected kernel patterns")
			}
			if len(loc.InitrdPatterns) == 0 {
				t.Error("expected initrd patterns")
			}
			if loc.ArchiveType == "" {
				t.Error("expected archive type")
			}
		})
	}
}

func TestDirectDownloadDistros(t *testing.T) {
	// Alpine uses direct download, no KernelLocator
	// Arch uses iso: URL scheme instead of KernelLocator
	directDownloadDistros := []ID{Alpine, ArchLinux}

	for _, id := range directDownloadDistros {
		t.Run(string(id), func(t *testing.T) {
			p, err := Get(id)
			if err != nil {
				t.Fatalf("Get(%q) failed: %v", id, err)
			}

			loc := p.KernelLocator()
			if loc != nil {
				t.Errorf("expected nil KernelLocator for %s (direct download)", id)
			}
		})
	}
}

// Task 2: URL validation tests

func TestAssetURLsWellFormed(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				urls, err := p.AssetURLs(arch)
				if err != nil {
					t.Fatalf("AssetURLs failed: %v", err)
				}

				// Rootfs always required
				if urls.Rootfs == "" {
					t.Error("Rootfs URL should not be empty")
				}
				if !strings.HasPrefix(urls.Rootfs, "http") {
					t.Errorf("Rootfs URL should start with http, got %q", urls.Rootfs)
				}

				// Check kernel/initrd based on extraction mode
				loc := p.KernelLocator()
				if loc == nil {
					// Direct download or iso: scheme - should have kernel/initrd URLs
					if urls.Kernel == "" {
						// Only error if not using iso: scheme
						if !strings.HasPrefix(urls.Kernel, "iso:") && id == Alpine {
							// Alpine has direct URLs
							t.Error("Alpine should have kernel URL")
						}
					}
					if urls.Initrd == "" && id == Alpine {
						t.Error("Alpine should have initrd URL")
					}

					// Validate URL format for Alpine (true direct download)
					if id == Alpine {
						if !strings.HasPrefix(urls.Kernel, "http") {
							t.Errorf("Alpine kernel URL should start with http, got %q", urls.Kernel)
						}
						if !strings.HasPrefix(urls.Initrd, "http") {
							t.Errorf("Alpine initrd URL should start with http, got %q", urls.Initrd)
						}
					}

					// Arch uses iso: scheme
					if id == ArchLinux {
						if !strings.HasPrefix(urls.Kernel, "iso:") {
							t.Errorf("Arch kernel URL should start with iso:, got %q", urls.Kernel)
						}
						if !strings.HasPrefix(urls.Initrd, "iso:") {
							t.Errorf("Arch initrd URL should start with iso:, got %q", urls.Initrd)
						}
					}
				} else {
					// Extraction distro - kernel/initrd come from rootfs
					if urls.Kernel != "" {
						t.Errorf("Extraction distro should not have kernel URL, got %q", urls.Kernel)
					}
					if urls.Initrd != "" {
						t.Errorf("Extraction distro should not have initrd URL, got %q", urls.Initrd)
					}
				}
			})
		}
	}
}

func TestAssetURLsUnsupportedArch(t *testing.T) {
	// Test that unsupported arch returns error
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		t.Run(string(id), func(t *testing.T) {
			// Find an unsupported arch
			var unsupported Arch
			for _, testArch := range []Arch{ArchAMD64, ArchARM64} {
				if !p.SupportsArch(testArch) {
					unsupported = testArch
					break
				}
			}

			if unsupported == "" {
				// All archs supported, skip this test
				t.Skip("all architectures supported")
			}

			_, err := p.AssetURLs(unsupported)
			if err == nil {
				t.Errorf("AssetURLs(%q) should return error for unsupported arch", unsupported)
			}
		})
	}
}

// Task 3: Boot config validation tests

func TestBootConfigComplete(t *testing.T) {
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				bc := p.BootConfig(arch)
				if bc == nil {
					t.Fatal("BootConfig returned nil")
				}

				// Cmdline required
				if bc.Cmdline == "" {
					t.Error("Cmdline should not be empty")
				}

				// Console device required (hvc0 for most virtualization)
				if bc.ConsoleDevice == "" {
					t.Error("ConsoleDevice should not be empty")
				}

				// Root device required
				if bc.RootDevice == "" {
					t.Error("RootDevice should not be empty")
				}

				// Validate cmdline contains console specification
				if !strings.Contains(bc.Cmdline, "console=") {
					t.Error("Cmdline should specify console")
				}

				// Validate cmdline contains root specification
				if !strings.Contains(bc.Cmdline, "root=") {
					t.Error("Cmdline should specify root device")
				}

				// Console in cmdline should match ConsoleDevice
				expectedConsole := fmt.Sprintf("console=%s", bc.ConsoleDevice)
				if !strings.Contains(bc.Cmdline, expectedConsole) {
					t.Errorf("Cmdline should contain %q, got %q", expectedConsole, bc.Cmdline)
				}

				// Root in cmdline should match RootDevice
				expectedRoot := fmt.Sprintf("root=%s", bc.RootDevice)
				if !strings.Contains(bc.Cmdline, expectedRoot) {
					t.Errorf("Cmdline should contain %q, got %q", expectedRoot, bc.Cmdline)
				}
			})
		}
	}
}

func TestBootConfigFilesystemTypes(t *testing.T) {
	// Verify filesystem types are consistent between BootConfig and SetupRequirements
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		t.Run(string(id), func(t *testing.T) {
			sr := p.SetupRequirements()
			if sr == nil {
				t.Fatal("SetupRequirements returned nil")
			}

			// Check all supported architectures
			for _, arch := range p.SupportedArchs() {
				bc := p.BootConfig(arch)
				if bc == nil {
					t.Fatalf("BootConfig(%q) returned nil", arch)
				}

				// If cmdline specifies rootfstype, it should match SetupRequirements.FSType
				if strings.Contains(bc.Cmdline, "rootfstype=") {
					expectedType := fmt.Sprintf("rootfstype=%s", sr.FSType)
					if !strings.Contains(bc.Cmdline, expectedType) && sr.FSType != "" {
						t.Errorf("Cmdline rootfstype should match SetupRequirements.FSType (%s), got %q", sr.FSType, bc.Cmdline)
					}
				}

				// RootFSType in BootConfig should match FSType in SetupRequirements
				if bc.RootFSType != "" && sr.FSType != "" && bc.RootFSType != sr.FSType {
					t.Errorf("BootConfig.RootFSType (%s) should match SetupRequirements.FSType (%s)", bc.RootFSType, sr.FSType)
				}
			}
		})
	}
}

func TestAllDistrosHaveConsistentConfig(t *testing.T) {
	// Verify all distros use hvc0 as console (VMTerminal standard)
	for _, id := range AllDistros() {
		p, err := Get(id)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", id, err)
		}

		for _, arch := range p.SupportedArchs() {
			t.Run(fmt.Sprintf("%s/%s", id, arch), func(t *testing.T) {
				bc := p.BootConfig(arch)
				if bc == nil {
					t.Fatal("BootConfig returned nil")
				}

				// All distros should use hvc0 for virtio console
				if bc.ConsoleDevice != "hvc0" {
					t.Errorf("ConsoleDevice = %q, want hvc0", bc.ConsoleDevice)
				}

				// All distros should use /dev/vda as root
				if bc.RootDevice != "/dev/vda" {
					t.Errorf("RootDevice = %q, want /dev/vda", bc.RootDevice)
				}
			})
		}
	}
}

func TestArchSupportPerDistro(t *testing.T) {
	tests := []struct {
		id       ID
		wantArch []Arch
	}{
		{Alpine, []Arch{ArchAMD64, ArchARM64}},
		{Ubuntu, []Arch{ArchAMD64, ArchARM64}},
		{ArchLinux, []Arch{ArchAMD64}}, // Arch only supports x86_64
		{Debian, []Arch{ArchAMD64, ArchARM64}},
		{Rocky, []Arch{ArchAMD64, ArchARM64}},
		{OpenSUSE, []Arch{ArchAMD64, ArchARM64}},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			p, err := Get(tt.id)
			if err != nil {
				t.Fatalf("Get(%q) failed: %v", tt.id, err)
			}

			archs := p.SupportedArchs()
			if len(archs) != len(tt.wantArch) {
				t.Errorf("SupportedArchs() = %v, want %v", archs, tt.wantArch)
			}

			for _, want := range tt.wantArch {
				if !p.SupportsArch(want) {
					t.Errorf("SupportsArch(%q) = false, want true", want)
				}
			}
		})
	}
}
