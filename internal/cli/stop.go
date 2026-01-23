package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running VM",
	Long:  `Stop the running VM gracefully. Currently only works when VM is running in foreground.`,
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	// For now, VMs run in foreground and are stopped via Ctrl+C
	// This command is a placeholder for when we add daemon mode
	fmt.Println("VM is not running in daemon mode.")
	fmt.Println("Use Ctrl+C to stop a foreground VM, or run 'vmterminal run' to start one.")
	return nil
}
