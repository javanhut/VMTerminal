// Package terminal provides terminal attachment functionality.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// ErrEscapeSequence is returned when the user triggers the escape sequence.
var ErrEscapeSequence = errors.New("escape sequence detected")

// Console wraps terminal operations for VM attachment.
type Console struct {
	stdin  *os.File
	stdout *os.File
	fd     int
}

// Current returns the current console.
func Current() *Console {
	return &Console{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		fd:     int(os.Stdin.Fd()),
	}
}

// IsTTY returns true if stdin is a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// SetRaw puts the terminal into raw mode and returns restore function.
func (c *Console) SetRaw() (func(), error) {
	oldState, err := term.MakeRaw(c.fd)
	if err != nil {
		return nil, err
	}
	return func() {
		term.Restore(c.fd, oldState)
	}, nil
}

// Size returns the current terminal size.
func (c *Console) Size() (width, height int, err error) {
	return term.GetSize(c.fd)
}

// Attach connects the terminal to VM I/O with bidirectional copy.
// Blocks until ctx is cancelled, escape sequence is detected, or an I/O stream closes.
// Returns ErrEscapeSequence if the user triggers the escape sequence (Ctrl+] twice).
func (c *Console) Attach(ctx context.Context, vmIn io.Writer, vmOut io.Reader) error {
	restore, err := c.SetRaw()
	if err != nil {
		return err
	}
	defer restore()

	// Print escape hint
	fmt.Fprintf(c.stdout, "Escape sequence: Ctrl+] Ctrl+] (press twice quickly to exit)\r\n")

	// Setup SIGWINCH handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Resize handler goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigCh:
				// Size is available via Size() - VM resize is implementation-specific
				// For now, just consume the signal to prevent blocking
			}
		}
	}()

	// Trigger initial resize check
	select {
	case sigCh <- syscall.SIGWINCH:
	default:
	}

	// Wrap stdin with escape sequence detector
	escapeReader := NewEscapeReader(c.stdin)

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Track if escape was triggered
	escaped := false
	var escapeMu sync.Mutex

	// stdin -> VM (with escape detection)
	go func() {
		defer wg.Done()
		_, copyErr := io.Copy(vmIn, escapeReader)
		// Check if we stopped due to escape sequence
		select {
		case <-escapeReader.Escaped():
			escapeMu.Lock()
			escaped = true
			escapeMu.Unlock()
		default:
			// Normal EOF or error
			_ = copyErr
		}
	}()

	// VM -> stdout
	go func() {
		defer wg.Done()
		io.Copy(c.stdout, vmOut)
	}()

	// Wait for context cancellation, escape, or I/O completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-escapeReader.Escaped():
		fmt.Fprintf(c.stdout, "\r\nEscape sequence detected, exiting...\r\n")
		return ErrEscapeSequence
	case <-done:
		escapeMu.Lock()
		wasEscaped := escaped
		escapeMu.Unlock()
		if wasEscaped {
			fmt.Fprintf(c.stdout, "\r\nEscape sequence detected, exiting...\r\n")
			return ErrEscapeSequence
		}
		return nil
	}
}
