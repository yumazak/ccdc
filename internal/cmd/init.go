package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/devcontainer"
	"github.com/yumazak/ccdc/internal/proxy"
)

var (
	proxyFlag  bool
	dockerFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate .devcontainer/ccdc/ feature with Claude Code support",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&proxyFlag, "proxy", false, "Generate proxy (Caddy forward proxy) configuration")
	initCmd.Flags().BoolVar(&dockerFlag, "docker", false, "Add Docker access via socket-proxy (use with --proxy)")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := devcontainer.Generate(cwd); err != nil {
		return err
	}

	fmt.Println("Created .devcontainer/ccdc/ feature")

	if !proxyFlag {
		printNextSteps(false, false)
		return nil
	}

	if err := proxy.GenerateCaddyfile(cwd, nil); err != nil {
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

	fmt.Println("Created .devcontainer/proxy/ (Caddy forward proxy + DNS)")
	fmt.Println("Created .devcontainer/dev/ (Claude Code)")
	if dockerFlag {
		fmt.Println("Created socket-proxy service (Docker access)")
	}
	fmt.Println("Created .devcontainer/docker-compose.proxy.yml")

	printNextSteps(true, dockerFlag)
	return nil
}

func printNextSteps(withProxy, withDocker bool) {
	fmt.Println("")
	fmt.Println("Next steps:")
	step := 1
	if withProxy {
		fmt.Printf("  %d. Edit .devcontainer/proxy/Caddyfile to add project-specific domains\n", step)
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
			fmt.Printf("  %d. docker compose exec <service> <command> (e.g. docker compose exec web bundle exec rspec)\n", step)
		}
	} else {
		fmt.Printf("  %d. Add \"./ccdc\": {} to features in your devcontainer.json\n", step)
		step++
		fmt.Printf("  %d. Set \"remoteUser\": \"ccdc\" in your devcontainer.json\n", step)
		step++
		fmt.Printf("  %d. devcontainer up --workspace-folder .\n", step)
		step++
		fmt.Printf("  %d. devcontainer exec --workspace-folder . bash\n", step)
		step++
		fmt.Printf("  %d. ccdc (alias for claude --dangerously-skip-permissions)\n", step)
	}
}
