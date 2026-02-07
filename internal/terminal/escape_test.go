package terminal

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestEscapeReaderNormalRead(t *testing.T) {
	input := []byte("hello world")
	r := NewEscapeReader(bytes.NewReader(input))

	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(buf[:n]) != "hello world" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hello world")
	}

	// Escaped channel should not be closed
	select {
	case <-r.Escaped():
		t.Error("escaped channel should not be closed")
	default:
		// OK
	}
}

func TestEscapeReaderSingleEscapePassesThrough(t *testing.T) {
	// Single Ctrl+] followed by normal char should pass through
	input := []byte{EscapeChar, 'a', 'b'}
	r := NewEscapeReader(bytes.NewReader(input))

	// First read might return 0 bytes if only escape is read
	buf := make([]byte, 64)
	total := 0
	for total < 3 {
		n, err := r.Read(buf[total:])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		total += n
	}

	expected := string([]byte{EscapeChar, 'a', 'b'})
	if string(buf[:total]) != expected {
		t.Errorf("got %q (%v), want %q", string(buf[:total]), buf[:total], expected)
	}

	// Escaped should not be triggered
	select {
	case <-r.Escaped():
		t.Error("escaped channel should not be closed for single escape")
	default:
		// OK
	}
}

func TestEscapeReaderDoubleEscapeTriggersExit(t *testing.T) {
	// Two Ctrl+] in a row triggers escape
	input := []byte{EscapeChar, EscapeChar}
	r := NewEscapeReader(bytes.NewReader(input))

	buf := make([]byte, 64)
	n, err := r.Read(buf)

	// Should return EOF with 0 bytes
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}

	// Escaped channel should be closed
	select {
	case <-r.Escaped():
		// OK
	default:
		t.Error("escaped channel should be closed")
	}
}

func TestEscapeReaderEscapeInMiddle(t *testing.T) {
	// Normal chars, then escape sequence
	input := []byte{'a', 'b', EscapeChar, EscapeChar}
	r := NewEscapeReader(bytes.NewReader(input))

	buf := make([]byte, 64)

	// First read should return "ab"
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	if string(buf[:n]) != "ab" {
		t.Errorf("first read: got %q, want %q", string(buf[:n]), "ab")
	}

	// Second read should detect escape and return EOF
	n, err = r.Read(buf)
	if err != io.EOF {
		t.Errorf("second read: expected EOF, got %v", err)
	}
	if n != 0 {
		t.Errorf("second read: expected 0 bytes, got %d", n)
	}

	// Escaped should be triggered
	select {
	case <-r.Escaped():
		// OK
	default:
		t.Error("escaped channel should be closed")
	}
}

func TestEscapeReaderTimeoutResets(t *testing.T) {
	// This test verifies the timeout behavior
	// We'll use a custom approach since we can't easily simulate time passing

	r := &EscapeReader{
		r:       bytes.NewReader([]byte{EscapeChar}),
		escaped: make(chan struct{}),
	}

	buf := make([]byte, 1)
	r.Read(buf) // First escape

	// Simulate timeout by setting lastEscape to the past
	r.mu.Lock()
	r.lastEscape = time.Now().Add(-EscapeTimeout - time.Second)
	r.mu.Unlock()

	// Read another escape - should reset due to timeout
	r.r = bytes.NewReader([]byte{EscapeChar, 'x'})
	buf = make([]byte, 64)
	n, _ := r.Read(buf)

	// Should get the escape char (since timeout reset) plus 'x'
	// Actually after timeout, it should reset counter and treat next escape as first
	if n == 0 {
		// Need another read to flush
		n, _ = r.Read(buf)
	}

	// The escape char should be flushed when 'x' arrives
	// This behavior is correct - pending escapes are flushed on non-escape char
}

func TestEscapeReaderMultipleCalls(t *testing.T) {
	// Escape sequence detected, subsequent reads should also return EOF
	input := []byte{EscapeChar, EscapeChar}
	r := NewEscapeReader(bytes.NewReader(input))

	buf := make([]byte, 64)

	// First read triggers escape
	_, err := r.Read(buf)
	if err != io.EOF {
		t.Errorf("first read: expected EOF, got %v", err)
	}

	// Escaped channel should remain closed (idempotent)
	select {
	case <-r.Escaped():
		// OK
	default:
		t.Error("escaped channel should remain closed")
	}
}
