package timing

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTimerMark(t *testing.T) {
	timer := New()

	// Sleep to ensure measurable duration
	time.Sleep(10 * time.Millisecond)
	timer.Mark("phase1")

	time.Sleep(15 * time.Millisecond)
	timer.Mark("phase2")

	phases := timer.Phases()
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}

	// Phase 1 should be ~10ms
	if phases[0].Name != "phase1" {
		t.Errorf("expected phase1, got %s", phases[0].Name)
	}
	if phases[0].Duration < 10*time.Millisecond {
		t.Errorf("phase1 duration too short: %v", phases[0].Duration)
	}

	// Phase 2 should be ~15ms
	if phases[1].Name != "phase2" {
		t.Errorf("expected phase2, got %s", phases[1].Name)
	}
	if phases[1].Duration < 15*time.Millisecond {
		t.Errorf("phase2 duration too short: %v", phases[1].Duration)
	}
}

func TestTimerTotal(t *testing.T) {
	timer := New()

	time.Sleep(10 * time.Millisecond)
	timer.Mark("phase1")

	total := timer.Total()
	if total < 10*time.Millisecond {
		t.Errorf("total too short: %v", total)
	}
}

func TestTimerReport(t *testing.T) {
	timer := New()

	time.Sleep(10 * time.Millisecond)
	timer.Mark("config")

	time.Sleep(10 * time.Millisecond)
	timer.Mark("startup")

	var buf bytes.Buffer
	timer.Report(&buf)

	output := buf.String()

	// Check report contains expected sections
	if !strings.Contains(output, "Startup Timing") {
		t.Error("report missing header")
	}
	if !strings.Contains(output, "config:") {
		t.Error("report missing config phase")
	}
	if !strings.Contains(output, "startup:") {
		t.Error("report missing startup phase")
	}
	if !strings.Contains(output, "TOTAL:") {
		t.Error("report missing total")
	}
}

func TestTimerEmpty(t *testing.T) {
	timer := New()

	// No marks - should still work
	phases := timer.Phases()
	if len(phases) != 0 {
		t.Errorf("expected 0 phases, got %d", len(phases))
	}

	// Total should still return a duration
	total := timer.Total()
	if total < 0 {
		t.Error("total should be positive")
	}

	// Report with no phases should not panic
	var buf bytes.Buffer
	timer.Report(&buf)
	output := buf.String()
	if !strings.Contains(output, "TOTAL:") {
		t.Error("empty report should still have total")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{500 * time.Microsecond, "500µs"},
		{50 * time.Millisecond, "50ms"},
		{1500 * time.Millisecond, "1.50s"},
		{2 * time.Second, "2.00s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.d)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.d, result, tt.expected)
		}
	}
}
