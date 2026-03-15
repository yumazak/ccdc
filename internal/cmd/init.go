package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/devcontainer"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate .devcontainer/ccdc/ feature with Claude Code support",
	RunE:  runInit,
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
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Add \"./ccdc\": {} to features in your devcontainer.json")
	fmt.Println("  2. Set \"remoteUser\": \"ccdc\" in your devcontainer.json")
	fmt.Println("  3. devcontainer up --workspace-folder .")
	fmt.Println("  4. devcontainer exec --workspace-folder . bash")
	fmt.Println("  5. ccdc (alias for claude --dangerously-skip-permissions)")
	return nil
}
