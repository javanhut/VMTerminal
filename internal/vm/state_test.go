package vm

import "testing"

func TestStateString(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  string
	}{
		{"new", StateNew, "new"},
		{"ready", StateReady, "ready"},
		{"running", StateRunning, "running"},
		{"stopping", StateStopping, "stopping"},
		{"stopped", StateStopped, "stopped"},
		{"error", StateError, "error"},
		{"unknown/invalid", State(99), "unknown"},
		{"negative", State(-1), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestStateConstants(t *testing.T) {
	// Verify state constants have expected values (iota order)
	if StateNew != 0 {
		t.Errorf("StateNew = %d, want 0", StateNew)
	}
	if StateReady != 1 {
		t.Errorf("StateReady = %d, want 1", StateReady)
	}
	if StateRunning != 2 {
		t.Errorf("StateRunning = %d, want 2", StateRunning)
	}
	if StateStopping != 3 {
		t.Errorf("StateStopping = %d, want 3", StateStopping)
	}
	if StateStopped != 4 {
		t.Errorf("StateStopped = %d, want 4", StateStopped)
	}
	if StateError != 5 {
		t.Errorf("StateError = %d, want 5", StateError)
	}
}
