package devcontainer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Generate creates .devcontainer/ccdc/ feature.
func Generate(projectDir string) error {
	dcDir := filepath.Join(projectDir, ".devcontainer")
	ccdcDir := filepath.Join(dcDir, "ccdc")

	if _, err := os.Stat(filepath.Join(ccdcDir, "devcontainer-feature.json")); err == nil {
		return fmt.Errorf(".devcontainer/ccdc/ already exists, skipping")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Generate the feature
	if err := generateFeature(ccdcDir, home); err != nil {
		return fmt.Errorf("failed to generate feature: %w", err)
	}

	return nil
}

func generateFeature(ccdcDir, home string) error {
	if err := os.MkdirAll(ccdcDir, 0o755); err != nil {
		return err
	}

	claudeDir := filepath.Join(home, ".claude")

	type mount struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type   string `json:"type"`
	}

	// Mount only necessary files/directories from ~/.claude
	mountTargets := []string{
		"CLAUDE.md",
		"agents",
		"commands",
		"skills",
		"projects",
	}

	var mounts []mount
	for _, name := range mountTargets {
		p := filepath.Join(claudeDir, name)

		if _, err := os.Lstat(p); err != nil {
			continue
		}

		mounts = append(mounts, mount{
			Source: "${localEnv:HOME}/.claude/" + name,
			Target: "/etc/claude/" + name,
			Type:   "bind",
		})
	}

	type featureJSON struct {
		ID               string  `json:"id"`
		Version          string  `json:"version"`
		Name             string  `json:"name"`
		Description      string  `json:"description"`
		Mounts           []mount `json:"mounts"`
		PostStartCommand string  `json:"postStartCommand"`
	}

	feature := featureJSON{
		ID:               "ccdc",
		Version:          "1.0.0",
		Name:             "Claude Code Dev Container",
		Description:      "Installs Claude Code with --dangerously-skip-permissions support",
		Mounts:           mounts,
		PostStartCommand: `mkdir -p ~/.claude && for item in /etc/claude/*; do [ -e "$item" ] && cp -r "$item" ~/.claude/$(basename "$item"); done`,
	}

	if err := writeJSON(filepath.Join(ccdcDir, "devcontainer-feature.json"), feature); err != nil {
		return err
	}

	// install.sh
	installScript := `#!/bin/bash
set -e

CCDC_USER="ccdc"
CCDC_HOME="/home/${CCDC_USER}"

# Create ccdc user if it doesn't exist
if ! id "${CCDC_USER}" &>/dev/null; then
  useradd -m -s /bin/bash "${CCDC_USER}"
fi

# Install Claude Code for the ccdc user
su - "${CCDC_USER}" -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Resolve claude binary path
CLAUDE_BIN="${CCDC_HOME}/.local/bin/claude"

# Create ccdc command (claude --dangerously-skip-permissions wrapper)
cat > /usr/local/bin/ccdc << SCRIPT
#!/bin/sh
exec ${CLAUDE_BIN} --dangerously-skip-permissions "\$@"
SCRIPT
chmod +x /usr/local/bin/ccdc
`

	return os.WriteFile(filepath.Join(ccdcDir, "install.sh"), []byte(installScript), 0o755)
}

func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
