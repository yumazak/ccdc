package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/proxy"
)

var joyFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate sandboxed Claude Code environment with network proxy",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&joyFlag, "joy", false, "Add joy notification forwarding to host")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := proxy.GenerateEnforcer(cwd); err != nil {
		return err
	}
	if err := proxy.GenerateDevDockerfile(cwd, joyFlag); err != nil {
		return err
	}
	if err := proxy.GenerateMiseToml(cwd); err != nil {
		return err
	}
	if err := proxy.GenerateCompose(cwd, joyFlag); err != nil {
		return err
	}

	fmt.Println("Created .ccdc/proxy/enforcer.py (mitmproxy L7 policy)")
	fmt.Println("Created .ccdc/dev/ (Claude Code)")
	if joyFlag {
		fmt.Println("Created joy notification forwarding")
	}
	fmt.Println("Created .ccdc/compose.yaml")

	printNextSteps()
	return nil
}

func printNextSteps() {
	fmt.Println("")
	fmt.Println("Next steps:")
	step := 1
	fmt.Printf("  %d. Edit .ccdc/dev/.mise.toml to add project tools (node, ruby, etc.)\n", step)
	step++
	fmt.Printf("  %d. Edit .ccdc/proxy/enforcer.py to customize access rules\n", step)
	step++
	fmt.Printf("  %d. docker compose -f .ccdc/compose.yaml up -d --build\n", step)
	step++
	fmt.Printf("  %d. docker compose -f .ccdc/compose.yaml exec dev bash\n", step)
	step++
	fmt.Printf("  %d. gh auth login (first time only)\n", step)
	step++
	fmt.Printf("  %d. ccdc (alias for claude --dangerously-skip-permissions)\n", step)
}
