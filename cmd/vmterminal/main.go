// Package main is the entry point for VMTerminal.
package main

import (
	"fmt"
	"os"

	"github.com/javanstorm/vmterminal/internal/cli"
)

func main() {
	// Check if running as login shell (argv[0] starts with '-')
	if cli.IsLoginShell() {
		cli.RunShellMode()
		return
	}

	// Normal CLI execution
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
