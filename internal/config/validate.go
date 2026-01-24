package config

import (
	"fmt"
	"strings"

	"github.com/javanstorm/vmterminal/pkg/hypervisor"
)

// ValidationError represents a configuration issue.
type ValidationError struct {
	Field   string
	Message string
	Fatal   bool // true = can't proceed, false = will be ignored
}

// ValidateConfig checks configuration against platform capabilities.
// Returns a list of validation errors/warnings.
func ValidateConfig(state *State, caps hypervisor.Capabilities) []ValidationError {
	var errors []ValidationError

	// Check shared directories
	if len(state.SharedDirs) > 0 && !caps.SharedDirs {
		errors = append(errors, ValidationError{
			Field:   "SharedDirs",
			Message: "Shared directories not supported on this platform (Linux KVM lacks virtio-fs)",
			Fatal:   false, // Will be ignored, not fatal
		})
	}

	// Check networking
	if state.EnableNetwork && !caps.Networking {
		errors = append(errors, ValidationError{
			Field:   "EnableNetwork",
			Message: "Networking not supported on this platform (Linux KVM lacks virtio-net)",
			Fatal:   false,
		})
	}

	return errors
}

// FormatValidationErrors returns human-readable error summary.
func FormatValidationErrors(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Configuration warnings:\n")
	for _, e := range errors {
		prefix := "Warning"
		if e.Fatal {
			prefix = "Error"
		}
		fmt.Fprintf(&b, "  %s [%s]: %s\n", prefix, e.Field, e.Message)
	}
	return b.String()
}
