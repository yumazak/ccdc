package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/proxy"
)

var dockerFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate sandboxed Claude Code environment with network proxy",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&dockerFlag, "docker", false, "Add Docker access via socket-proxy")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := proxy.GenerateSquidConf(cwd, nil); err != nil {
		return err
	}
	if err := proxy.GenerateProxyDockerfile(cwd); err != nil {
		return err
	}
	if err := proxy.GenerateProxyEntrypoint(cwd); err != nil {
		return err
	}
	if err := proxy.GenerateDevDockerfile(cwd, dockerFlag); err != nil {
		return err
	}
	if err := proxy.GenerateCompose(cwd, dockerFlag); err != nil {
		return err
	}

	fmt.Println("Created .devcontainer/proxy/ (Squid forward proxy + DNS)")
	fmt.Println("Created .devcontainer/dev/ (Claude Code)")
	if dockerFlag {
		fmt.Println("Created socket-proxy service (Docker access)")
	}
	fmt.Println("Created .devcontainer/docker-compose.proxy.yml")

	printNextSteps(dockerFlag)
	return nil
}

func printNextSteps(withDocker bool) {
	fmt.Println("")
	fmt.Println("Next steps:")
	step := 1
	fmt.Printf("  %d. Edit .devcontainer/proxy/squid.conf to add project-specific domains\n", step)
	step++
	fmt.Printf("  %d. export GITHUB_TOKEN=$(gh auth token)\n", step)
	step++
	if withDocker {
		fmt.Printf("  %d. Start your project services: docker compose up -d\n", step)
		step++
	}
	fmt.Printf("  %d. docker compose -f .devcontainer/docker-compose.proxy.yml up -d --build\n", step)
	step++
	fmt.Printf("  %d. docker compose -f .devcontainer/docker-compose.proxy.yml exec dev bash\n", step)
	step++
	fmt.Printf("  %d. ccdc (alias for claude --dangerously-skip-permissions)\n", step)
	if withDocker {
		step++
		fmt.Printf("  %d. docker compose -p <project> exec <service> <command>\n", step)
	}
}
