// Package gui provides the GUI terminal emulator window for VMTerminal.
package gui

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	fyneterm "github.com/fyne-io/terminal"
)

// nopWriteCloser wraps an io.Writer with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// RunTerminal opens a GUI window with a terminal emulator connected to the VM.
// vmIn is the writer to send input to the VM (keyboard -> VM).
// vmOut is the reader to receive output from the VM (VM -> display).
// title is the window title.
// onClose is called when the user closes the window or the VM connection ends.
// This function blocks until the window is closed.
func RunTerminal(vmIn io.Writer, vmOut io.Reader, title string, onClose func()) {
	a := app.New()
	w := a.NewWindow(title)
	w.SetPadded(false)
	w.Resize(fyne.NewSize(800, 600))

	t := fyneterm.New()
	w.SetContent(t)

	w.SetCloseIntercept(func() {
		if onClose != nil {
			onClose()
		}
		a.Quit()
	})

	// Handle signals: first SIGINT/SIGTERM does graceful close, second forces exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		// First signal: graceful shutdown
		if onClose != nil {
			onClose()
		}
		a.Quit()

		// Second signal: force exit
		<-sigCh
		os.Exit(1)
	}()

	// When VM connection ends, close the window
	go func() {
		_ = t.RunWithConnection(nopWriteCloser{vmIn}, vmOut)
		if onClose != nil {
			onClose()
		}
		a.Quit()
	}()

	w.Show()
	w.Canvas().Focus(t)
	a.Run()
}
