package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Show installation instructions",
	Long:  `Display instructions for setting vmterminal as your login shell.`,
	Run:   runInstall,
}

func runInstall(cmd *cobra.Command, args []string) {
	// Try to find the binary path
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "/usr/local/bin/vmterminal"
	} else {
		binaryPath, _ = filepath.Abs(binaryPath)
	}

	fmt.Printf(`VMTerminal Shell Installation
=============================

To use vmterminal as your login shell:

1. First, verify the binary location:
   which vmterminal
   Current location: %s

2. Add vmterminal to /etc/shells (requires root):
   sudo sh -c 'echo %s >> /etc/shells'

3. Set it as your default shell:
   chsh -s %s

4. Open a new terminal to use your Linux VM as your shell.

To revert:
   chsh -s /bin/bash  # or your preferred shell

Note: The first launch may take longer as it downloads the Linux kernel.

How it works:
- When a shell is set as login shell, it's invoked with argv[0] prefixed with '-'
- vmterminal detects this and automatically starts the VM shell
- The VM console becomes your terminal

`, binaryPath, binaryPath, binaryPath)
}
