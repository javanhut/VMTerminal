package distro

import (
	"errors"
	"testing"
)

func TestRegistryGet(t *testing.T) {
	tests := []struct {
		name    string
		id      ID
		wantErr bool
	}{
		{"alpine", Alpine, false},
		{"ubuntu", Ubuntu, false},
		{"arch", ArchLinux, false},
		{"debian", Debian, false},
		{"rocky", Rocky, false},
		{"opensuse", OpenSUSE, false},
		{"unknown", ID("unknown"), true},
		{"empty", ID(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := Get(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Errorf("Get(%q) returned nil provider", tt.id)
			}
			if !tt.wantErr && provider.ID() != tt.id {
				t.Errorf("Get(%q) returned provider with ID %q", tt.id, provider.ID())
			}
		})
	}
}

func TestIsRegistered(t *testing.T) {
	tests := []struct {
		name string
		id   ID
		want bool
	}{
		{"alpine registered", Alpine, true},
		{"ubuntu registered", Ubuntu, true},
		{"arch registered", ArchLinux, true},
		{"debian registered", Debian, true},
		{"rocky registered", Rocky, true},
		{"opensuse registered", OpenSUSE, true},
		{"unknown not registered", ID("unknown"), false},
		{"empty not registered", ID(""), false},
		{"random not registered", ID("random-distro"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRegistered(tt.id)
			if got != tt.want {
				t.Errorf("IsRegistered(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestDefaultID(t *testing.T) {
	id := DefaultID()
	if id != Alpine {
		t.Errorf("DefaultID() = %q, want %q", id, Alpine)
	}
}

func TestGetDefault(t *testing.T) {
	provider, err := GetDefault()
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if provider == nil {
		t.Fatal("GetDefault() returned nil provider")
	}
	if provider.ID() != Alpine {
		t.Errorf("GetDefault() returned provider with ID %q, want %q", provider.ID(), Alpine)
	}
	if provider.Name() == "" {
		t.Error("GetDefault() provider has empty name")
	}
}

func TestList(t *testing.T) {
	ids := List()

	// Should have at least 6 distros
	if len(ids) < 6 {
		t.Errorf("List() returned %d distros, want at least 6", len(ids))
	}

	// Check all expected distros are present
	expected := []ID{Alpine, Ubuntu, ArchLinux, Debian, Rocky, OpenSUSE}
	for _, exp := range expected {
		found := false
		for _, id := range ids {
			if id == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("List() missing expected distro %q", exp)
		}
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	ids := List()

	// Should have same count as List()
	if len(providers) != len(ids) {
		t.Errorf("ListProviders() returned %d, List() returned %d", len(providers), len(ids))
	}

	// Each provider should have valid ID
	for _, p := range providers {
		if p == nil {
			t.Error("ListProviders() contains nil provider")
			continue
		}
		if p.ID() == "" {
			t.Error("provider has empty ID")
		}
		if p.Name() == "" {
			t.Errorf("provider %q has empty name", p.ID())
		}
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ID
		wantErr bool
	}{
		{"alpine", "alpine", Alpine, false},
		{"ubuntu", "ubuntu", Ubuntu, false},
		{"arch", "arch", ArchLinux, false},
		{"debian", "debian", Debian, false},
		{"rocky", "rocky", Rocky, false},
		{"opensuse", "opensuse", OpenSUSE, false},
		{"unknown", "unknown", "", true},
		{"empty", "", "", true},
		{"invalid", "not-a-distro", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestErrUnknownDistro(t *testing.T) {
	// Get an error by requesting unknown distro
	_, err := Get(ID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for unknown distro")
	}

	// Check error type using errors.As
	var unknownErr *ErrUnknownDistro
	if !errors.As(err, &unknownErr) {
		t.Errorf("error should be *ErrUnknownDistro, got %T", err)
	}

	// Check error message contains the unknown ID
	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}

	// Error message should mention the unknown ID
	if unknownErr.ID != ID("nonexistent") {
		t.Errorf("ErrUnknownDistro.ID = %q, want %q", unknownErr.ID, "nonexistent")
	}
}

func TestProviderInterface(t *testing.T) {
	// Test that all registered providers implement the interface properly
	providers := ListProviders()

	for _, p := range providers {
		t.Run(string(p.ID()), func(t *testing.T) {
			// ID should be non-empty
			if p.ID() == "" {
				t.Error("ID() should not be empty")
			}

			// Name should be non-empty
			if p.Name() == "" {
				t.Error("Name() should not be empty")
			}

			// Version should be non-empty
			if p.Version() == "" {
				t.Error("Version() should not be empty")
			}

			// Should support at least one architecture
			archs := p.SupportedArchs()
			if len(archs) == 0 {
				t.Error("SupportedArchs() should return at least one arch")
			}

			// SupportsArch should be consistent with SupportedArchs
			for _, arch := range archs {
				if !p.SupportsArch(arch) {
					t.Errorf("SupportsArch(%q) = false, but arch is in SupportedArchs()", arch)
				}
			}
		})
	}
}
