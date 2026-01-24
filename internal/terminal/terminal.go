// Package terminal provides terminal attachment functionality.
package terminal

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

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
// Blocks until ctx is cancelled or an I/O stream closes.
func (c *Console) Attach(ctx context.Context, vmIn io.Writer, vmOut io.Reader) error {
	restore, err := c.SetRaw()
	if err != nil {
		return err
	}
	defer restore()

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

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// stdin -> VM
	go func() {
		defer wg.Done()
		io.Copy(vmIn, c.stdin)
	}()

	// VM -> stdout
	go func() {
		defer wg.Done()
		io.Copy(c.stdout, vmOut)
	}()

	// Wait for context cancellation or I/O completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
