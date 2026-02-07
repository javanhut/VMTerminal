// Package terminal provides terminal attachment functionality.
package terminal

import (
	"io"
	"sync"
	"time"
)

const (
	// EscapeChar is Ctrl+] (0x1D).
	EscapeChar = 0x1D

	// EscapeCount is the number of consecutive escape chars needed.
	EscapeCount = 2

	// EscapeTimeout is the maximum time between escape key presses.
	EscapeTimeout = 500 * time.Millisecond
)

// EscapeReader wraps an io.Reader and detects escape sequences.
// When EscapeCount consecutive EscapeChar bytes are read within EscapeTimeout,
// it signals via the Escaped channel and returns io.EOF.
type EscapeReader struct {
	r           io.Reader
	escaped     chan struct{}
	escapedOnce sync.Once

	mu          sync.Mutex
	escapeCount int
	lastEscape  time.Time
}

// NewEscapeReader creates an EscapeReader wrapping the given reader.
func NewEscapeReader(r io.Reader) *EscapeReader {
	return &EscapeReader{
		r:       r,
		escaped: make(chan struct{}),
	}
}

// Escaped returns a channel that is closed when the escape sequence is detected.
func (e *EscapeReader) Escaped() <-chan struct{} {
	return e.escaped
}

// Read reads from the underlying reader, detecting escape sequences.
// When the escape sequence is detected, it closes the Escaped channel
// and returns io.EOF.
func (e *EscapeReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if n == 0 {
		return n, err
	}

	// Process each byte looking for escape sequence
	writeIdx := 0
	for i := 0; i < n; i++ {
		b := p[i]

		if b == EscapeChar {
			e.mu.Lock()
			now := time.Now()

			// Check if within timeout window
			if e.escapeCount > 0 && now.Sub(e.lastEscape) > EscapeTimeout {
				// Timeout expired, reset counter
				e.escapeCount = 0
			}

			e.escapeCount++
			e.lastEscape = now

			if e.escapeCount >= EscapeCount {
				// Escape sequence detected!
				e.mu.Unlock()
				e.escapedOnce.Do(func() {
					close(e.escaped)
				})
				// Return what we have so far (excluding escape chars) plus EOF
				if writeIdx > 0 {
					return writeIdx, nil
				}
				return 0, io.EOF
			}
			e.mu.Unlock()
			// Don't write this escape char yet - might be part of sequence
			continue
		}

		// Non-escape char - flush any pending escape chars first
		e.mu.Lock()
		pendingEscapes := e.escapeCount
		e.escapeCount = 0
		e.mu.Unlock()

		// Write any pending escape chars that weren't part of a sequence
		for j := 0; j < pendingEscapes; j++ {
			if writeIdx < len(p) {
				p[writeIdx] = EscapeChar
				writeIdx++
			}
		}

		// Write the current byte
		if writeIdx < len(p) {
			p[writeIdx] = b
			writeIdx++
		}
	}

	// If we processed all bytes and have pending escapes at the end,
	// we need to decide what to do. For now, we hold them in case
	// the next read completes the sequence.
	// The bytes are already in the escapeCount, they'll be flushed
	// on the next non-escape char.

	if writeIdx == 0 && n > 0 {
		// All bytes were escape chars being held - return 0 bytes read
		// but no error so caller retries
		return 0, nil
	}

	return writeIdx, err
}
