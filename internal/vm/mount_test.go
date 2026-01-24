package vm

import (
	"strings"
	"testing"
)

func TestMountHelperSingleShare(t *testing.T) {
	shares := map[string]string{
		"home": "/Users/test",
	}
	helper := NewMountHelper(shares)

	cmd := helper.GenerateMountCommand("home", "/mnt/host/home")
	expected := "mkdir -p /mnt/host/home && mount -t virtiofs home /mnt/host/home"
	if cmd != expected {
		t.Errorf("GenerateMountCommand() = %q, want %q", cmd, expected)
	}
}

func TestMountHelperMultipleShares(t *testing.T) {
	shares := map[string]string{
		"home":     "/Users/test",
		"projects": "/Users/test/Projects",
	}
	helper := NewMountHelper(shares)

	script := helper.GenerateMountScript("/mnt/host")

	// Verify script contains both mounts
	if !strings.Contains(script, "mount -t virtiofs home /mnt/host/home") {
		t.Error("Script missing home mount command")
	}
	if !strings.Contains(script, "mount -t virtiofs projects /mnt/host/projects") {
		t.Error("Script missing projects mount command")
	}

	// Verify header with virtiofs check
	if !strings.Contains(script, "grep -q virtiofs /proc/filesystems") {
		t.Error("Script missing virtiofs availability check")
	}

	// Verify mounts are in alphabetical order (home before projects)
	homeIdx := strings.Index(script, "# Mount home")
	projectsIdx := strings.Index(script, "# Mount projects")
	if homeIdx == -1 || projectsIdx == -1 {
		t.Error("Script missing mount comments")
	}
	if homeIdx > projectsIdx {
		t.Error("Shares not in alphabetical order")
	}
}

func TestMountHelperEmptyShares(t *testing.T) {
	helper := NewMountHelper(nil)

	script := helper.GenerateMountScript("/mnt/host")
	if script != "" {
		t.Errorf("GenerateMountScript() with no shares = %q, want empty string", script)
	}

	if helper.HasShares() {
		t.Error("HasShares() = true, want false")
	}
}

func TestMountHelperEmptyMap(t *testing.T) {
	helper := NewMountHelper(map[string]string{})

	script := helper.GenerateMountScript("/mnt/host")
	if script != "" {
		t.Errorf("GenerateMountScript() with empty map = %q, want empty string", script)
	}

	if helper.HasShares() {
		t.Error("HasShares() = true, want false")
	}
}

func TestMountHelperPathsWithSpaces(t *testing.T) {
	shares := map[string]string{
		"docs": "/Users/test/My Documents",
	}
	helper := NewMountHelper(shares)

	cmd := helper.GenerateMountCommand("docs", "/mnt/host/my docs")

	// Path with spaces should be quoted
	if !strings.Contains(cmd, "'/mnt/host/my docs'") {
		t.Errorf("Path with spaces not properly quoted: %s", cmd)
	}
}

func TestMountHelperPathsWithSingleQuotes(t *testing.T) {
	shares := map[string]string{
		"docs": "/Users/test/Bob's Files",
	}
	helper := NewMountHelper(shares)

	cmd := helper.GenerateMountCommand("docs", "/mnt/host/Bob's Files")

	// Single quotes within path should be escaped
	if !strings.Contains(cmd, "'\"'\"'") {
		t.Errorf("Single quote in path not properly escaped: %s", cmd)
	}
}

func TestMountHelperTags(t *testing.T) {
	shares := map[string]string{
		"zeta":   "/z",
		"alpha":  "/a",
		"middle": "/m",
	}
	helper := NewMountHelper(shares)

	tags := helper.Tags()

	// Verify sorted order
	expected := []string{"alpha", "middle", "zeta"}
	if len(tags) != len(expected) {
		t.Errorf("Tags() returned %d items, want %d", len(tags), len(expected))
	}
	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("Tags()[%d] = %q, want %q", i, tag, expected[i])
		}
	}
}

func TestMountHelperHasShares(t *testing.T) {
	tests := []struct {
		name   string
		shares map[string]string
		want   bool
	}{
		{"nil", nil, false},
		{"empty", map[string]string{}, false},
		{"one share", map[string]string{"home": "/home"}, true},
		{"multiple shares", map[string]string{"home": "/home", "work": "/work"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := NewMountHelper(tt.shares)
			if got := helper.HasShares(); got != tt.want {
				t.Errorf("HasShares() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMountHelperScriptStructure(t *testing.T) {
	shares := map[string]string{
		"data": "/data",
	}
	helper := NewMountHelper(shares)

	script := helper.GenerateMountScript("/mnt")

	// Verify script is a valid shell script
	if !strings.HasPrefix(script, "#!/bin/sh") {
		t.Error("Script should start with shebang")
	}

	// Verify skip-if-mounted logic
	if !strings.Contains(script, "mountpoint -q") {
		t.Error("Script missing already-mounted check")
	}

	// Verify error handling
	if !strings.Contains(script, "Failed to mount") {
		t.Error("Script missing mount failure message")
	}

	// Verify success message
	if !strings.Contains(script, "Mounted data at") {
		t.Error("Script missing mount success message")
	}
}

func TestQuotePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/simple/path", "/simple/path"},
		{"/path with spaces", "'/path with spaces'"},
		{"/path\twith\ttabs", "'/path\twith\ttabs'"},
		{"/path'quote", "'/path'\"'\"'quote'"},
		{"/path$var", "'/path$var'"},
		{"/path`cmd`", "'/path`cmd`'"},
		{"/path\\escape", "'/path\\escape'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := quotePath(tt.input)
			if got != tt.want {
				t.Errorf("quotePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
