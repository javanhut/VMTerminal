package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/javanstorm/vmterminal/internal/terminal"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach to VM console",
	Long:  `Attach to the running VM's console. Press Ctrl+] to detach.`,
	RunE:  runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	// For now, attach command is a placeholder
	// Full implementation would need to connect to an existing running VM
	// This requires a daemon/socket pattern which is out of scope for this plan
	fmt.Println("Attach command not yet implemented for standalone use.")
	fmt.Println("Use 'vmterminal run' to start and attach to a VM.")
	return nil
}

// AttachToConsole attaches the current terminal to VM console I/O.
// This is called by the run command after starting the VM.
// Returns when VM exits, context is cancelled, or escape sequence (Ctrl+]) is pressed.
func AttachToConsole(ctx context.Context, vmIn io.Writer, vmOut io.Reader) error {
	console := terminal.Current()

	// Create a context that we can cancel on escape sequence
	attachCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-attachCtx.Done():
		}
	}()

	// Use the terminal package's Attach for bidirectional I/O
	// The terminal package handles raw mode and SIGWINCH
	return console.Attach(attachCtx, vmIn, vmOut)
}
