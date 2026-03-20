package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/devcontainer"
	"github.com/yumazak/ccdc/internal/proxy"
)

var (
	proxyFlag bool
	dindFlag  bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate .devcontainer/ccdc/ feature with Claude Code support",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&proxyFlag, "proxy", false, "Generate proxy (Caddy forward proxy) configuration")
	initCmd.Flags().BoolVar(&dindFlag, "dind", false, "Use Docker-in-Docker for the dev container (use with --proxy)")
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
	if err := proxy.GenerateDevDockerfile(cwd, dindFlag); err != nil {
		return err
	}
	if dindFlag {
		if err := proxy.GenerateDevEntrypoint(cwd); err != nil {
			return err
		}
	}
	if err := proxy.GenerateCompose(cwd, dindFlag); err != nil {
		return err
	}

	fmt.Println("Created .devcontainer/proxy/ (Caddy forward proxy + DNS)")
	if dindFlag {
		fmt.Println("Created .devcontainer/dev/ (Docker-in-Docker + Claude Code)")
	} else {
		fmt.Println("Created .devcontainer/dev/ (Claude Code)")
	}
	fmt.Println("Created .devcontainer/docker-compose.proxy.yml")

	printNextSteps(true, dindFlag)
	return nil
}

func printNextSteps(withProxy, withDind bool) {
	fmt.Println("")
	fmt.Println("Next steps:")
	step := 1
	if withProxy {
		fmt.Printf("  %d. Edit .devcontainer/proxy/allowlist.txt to add project-specific domains\n", step)
		step++
		fmt.Printf("  %d. export GITHUB_TOKEN=$(gh auth token)\n", step)
		step++
		fmt.Printf("  %d. docker compose -f .devcontainer/docker-compose.proxy.yml up -d --build\n", step)
		step++
		fmt.Printf("  %d. docker compose -f .devcontainer/docker-compose.proxy.yml exec -u ccdc dev bash\n", step)
		step++
		fmt.Printf("  %d. ccdc (alias for claude --dangerously-skip-permissions)\n", step)
		if withDind {
			step++
			fmt.Printf("  %d. docker compose up (inside the container, for your project services)\n", step)
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
