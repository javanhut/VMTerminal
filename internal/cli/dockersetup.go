package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dockerSetupCmd = &cobra.Command{
	Use:   "docker-setup",
	Short: "Show Docker installation instructions",
	Long:  `Display instructions for installing and configuring Docker inside the VM.`,
	RunE:  runDockerSetup,
}

func init() {
	rootCmd.AddCommand(dockerSetupCmd)
}

func runDockerSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Docker Setup for VMTerminal")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Println("Docker enables containerized workloads inside your VMTerminal VM.")
	fmt.Println()
	fmt.Println("STEP 1: Start VM")
	fmt.Println("----------------")
	fmt.Println("  vmterminal run")
	fmt.Println()
	fmt.Println("STEP 2: Install Docker (in VM)")
	fmt.Println("------------------------------")
	fmt.Println("  # Install Docker and Docker Compose")
	fmt.Println("  apk add docker docker-cli-compose")
	fmt.Println()
	fmt.Println("STEP 3: Enable Docker Service (in VM)")
	fmt.Println("-------------------------------------")
	fmt.Println("  # Start Docker now")
	fmt.Println("  service docker start")
	fmt.Println()
	fmt.Println("  # Enable Docker on boot")
	fmt.Println("  rc-update add docker boot")
	fmt.Println()
	fmt.Println("STEP 4: Configure User Permissions (in VM)")
	fmt.Println("------------------------------------------")
	fmt.Println("  # Add your user to the docker group (if not root)")
	fmt.Println("  addgroup $USER docker")
	fmt.Println()
	fmt.Println("  # Log out and back in for group changes to take effect")
	fmt.Println()
	fmt.Println("STEP 5: Verify Installation (in VM)")
	fmt.Println("-----------------------------------")
	fmt.Println("  # Check Docker is running")
	fmt.Println("  docker info")
	fmt.Println()
	fmt.Println("  # Run a test container")
	fmt.Println("  docker run --rm hello-world")
	fmt.Println()
	fmt.Println("USING FROM HOST")
	fmt.Println("---------------")
	fmt.Println("Once Docker is set up and SSH is configured, you can run Docker")
	fmt.Println("commands from your host:")
	fmt.Println()
	fmt.Println("  vmterminal docker ps")
	fmt.Println("  vmterminal docker run -it alpine sh")
	fmt.Println("  vmterminal docker compose up -d")
	fmt.Println()
	fmt.Println("TROUBLESHOOTING")
	fmt.Println("---------------")
	fmt.Println("- If Docker fails to start: Check cgroups are mounted")
	fmt.Println("  mount | grep cgroup")
	fmt.Println("- If permission denied: Make sure user is in docker group")
	fmt.Println("- If network issues: Ensure VM networking is enabled")
	fmt.Println()

	return nil
}
