package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var podmanSetupCmd = &cobra.Command{
	Use:   "podman-setup",
	Short: "Show Podman installation instructions",
	Long:  `Display instructions for installing and configuring Podman inside the VM.`,
	RunE:  runPodmanSetup,
}

func init() {
	rootCmd.AddCommand(podmanSetupCmd)
}

func runPodmanSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Podman Setup for VMTerminal")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Println("Podman is a daemonless container engine that can run as root or rootless.")
	fmt.Println()
	fmt.Println("STEP 1: Start VM")
	fmt.Println("----------------")
	fmt.Println("  vmterminal run")
	fmt.Println()
	fmt.Println("STEP 2: Install Podman (in VM)")
	fmt.Println("------------------------------")
	fmt.Println("  # Install Podman and Podman Compose")
	fmt.Println("  apk add podman podman-compose")
	fmt.Println()
	fmt.Println("STEP 3: Configure Cgroups (in VM)")
	fmt.Println("---------------------------------")
	fmt.Println("  # Podman needs cgroups v2 for rootless mode")
	fmt.Println("  # Check cgroup version:")
	fmt.Println("  cat /sys/fs/cgroup/cgroup.controllers")
	fmt.Println()
	fmt.Println("  # If empty or missing, you may need to use rootful mode")
	fmt.Println()
	fmt.Println("STEP 4: Configure Storage (in VM)")
	fmt.Println("---------------------------------")
	fmt.Println("  # For rootless podman, configure storage")
	fmt.Println("  mkdir -p ~/.config/containers")
	fmt.Println()
	fmt.Println("  # Create storage.conf if needed:")
	fmt.Println("  cat > ~/.config/containers/storage.conf << 'EOF'")
	fmt.Println("  [storage]")
	fmt.Println("  driver = \"overlay\"")
	fmt.Println("  EOF")
	fmt.Println()
	fmt.Println("STEP 5: Verify Installation (in VM)")
	fmt.Println("-----------------------------------")
	fmt.Println("  # Check Podman version")
	fmt.Println("  podman --version")
	fmt.Println()
	fmt.Println("  # Run a test container")
	fmt.Println("  podman run --rm docker.io/library/alpine echo 'Hello from Podman!'")
	fmt.Println()
	fmt.Println("USING FROM HOST")
	fmt.Println("---------------")
	fmt.Println("Once Podman is set up and SSH is configured, you can run Podman")
	fmt.Println("commands from your host:")
	fmt.Println()
	fmt.Println("  vmterminal podman ps")
	fmt.Println("  vmterminal podman run -it alpine sh")
	fmt.Println("  vmterminal podman-compose up -d")
	fmt.Println()
	fmt.Println("ROOTLESS VS ROOTFUL")
	fmt.Println("-------------------")
	fmt.Println("- Rootless: More secure, runs as normal user, needs user namespaces")
	fmt.Println("- Rootful: Run as root for full access, simpler setup")
	fmt.Println()
	fmt.Println("For rootless, you may need to configure /etc/subuid and /etc/subgid")
	fmt.Println()
	fmt.Println("TROUBLESHOOTING")
	fmt.Println("---------------")
	fmt.Println("- If 'permission denied': Try running as root or configure rootless")
	fmt.Println("- If 'overlay not supported': Use vfs driver in storage.conf")
	fmt.Println("- If network issues: Check /etc/cni/net.d/ configuration")
	fmt.Println()

	return nil
}
