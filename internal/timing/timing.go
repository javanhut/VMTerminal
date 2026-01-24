// Package timing provides simple phase timing for startup performance measurement.
package timing

import (
	"fmt"
	"io"
	"time"
)

// Timer tracks durations of named phases.
type Timer struct {
	start  time.Time
	phases []Phase
}

// Phase represents a timed phase with name and duration.
type Phase struct {
	Name     string
	Duration time.Duration
}

// New creates a new Timer starting from now.
func New() *Timer {
	return &Timer{start: time.Now()}
}

// Mark records a named phase ending now.
// Duration is time since last mark (or since start if first mark).
func (t *Timer) Mark(name string) {
	now := time.Now()
	var duration time.Duration
	if len(t.phases) == 0 {
		duration = now.Sub(t.start)
	} else {
		duration = now.Sub(t.start) - t.totalDuration()
	}
	t.phases = append(t.phases, Phase{Name: name, Duration: duration})
}

// Total returns the total elapsed time since timer creation.
func (t *Timer) Total() time.Duration {
	return time.Since(t.start)
}

// Phases returns all recorded phases.
func (t *Timer) Phases() []Phase {
	return t.phases
}

// Report prints a timing report to the given writer.
func (t *Timer) Report(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "=== Startup Timing ===")
	for _, p := range t.phases {
		fmt.Fprintf(w, "  %-20s %s\n", p.Name+":", formatDuration(p.Duration))
	}
	fmt.Fprintf(w, "  %-20s %s\n", "TOTAL:", formatDuration(t.Total()))
	fmt.Fprintln(w, "======================")
}

// totalDuration returns the sum of all phase durations.
func (t *Timer) totalDuration() time.Duration {
	var total time.Duration
	for _, p := range t.phases {
		total += p.Duration
	}
	return total
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
