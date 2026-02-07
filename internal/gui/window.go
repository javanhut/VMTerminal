// Package gui provides the GUI terminal emulator window for VMTerminal.
package gui

import (
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
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
// onClose is called when the user closes the window.
// This function blocks until the window is closed.
func RunTerminal(vmIn io.Writer, vmOut io.Reader, title string, onClose func()) {
	a := app.New()
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(800, 600))

	t := fyneterm.New()
	w.SetContent(container.NewStack(t))

	w.SetCloseIntercept(func() {
		if onClose != nil {
			onClose()
		}
		w.Close()
	})

	go func() {
		_ = t.RunWithConnection(nopWriteCloser{vmIn}, vmOut)
	}()

	w.ShowAndRun()
}
