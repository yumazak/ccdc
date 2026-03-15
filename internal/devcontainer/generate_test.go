package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	projectDir := t.TempDir()

	// Setup fake ~/.claude with some files
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use generateFeature and generatePlugin directly to control home dir
	ccdcDir := filepath.Join(projectDir, ".devcontainer", "ccdc")

	if err := generateFeature(ccdcDir, fakeHome); err != nil {
		t.Fatalf("generateFeature failed: %v", err)
	}
	if err := generatePlugin(ccdcDir); err != nil {
		t.Fatalf("generatePlugin failed: %v", err)
	}

	t.Run("feature files exist", func(t *testing.T) {
		for _, name := range []string{"devcontainer-feature.json", "install.sh"} {
			p := filepath.Join(ccdcDir, name)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected %s to exist: %v", name, err)
			}
		}
	})

	t.Run("plugin files exist", func(t *testing.T) {
		pluginDir := filepath.Join(ccdcDir, "ccdc-plugin")
		for _, rel := range []string{
			".claude-plugin/plugin.json",
			"hooks/notify.sh",
			"hooks/hooks.json",
		} {
			p := filepath.Join(pluginDir, rel)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected %s to exist: %v", rel, err)
			}
		}
	})

	t.Run("devcontainer-feature.json content", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "devcontainer-feature.json"))
		if err != nil {
			t.Fatal(err)
		}

		var feature map[string]any
		if err := json.Unmarshal(data, &feature); err != nil {
			t.Fatal(err)
		}

		if feature["id"] != "ccdc" {
			t.Errorf("expected id=ccdc, got %v", feature["id"])
		}

		mounts, ok := feature["mounts"].([]any)
		if !ok {
			t.Fatal("mounts is not an array")
		}

		// We created settings.json and CLAUDE.md, so expect 2 mounts
		if len(mounts) != 2 {
			t.Errorf("expected 2 mounts, got %d", len(mounts))
		}

		for _, m := range mounts {
			mount := m.(map[string]any)
			source := mount["source"].(string)
			if !strings.HasPrefix(source, "${localEnv:HOME}/.claude/") {
				t.Errorf("mount source should use ${localEnv:HOME}, got %s", source)
			}
		}
	})

	t.Run("install.sh contains plugin-dir", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "install.sh"))
		if err != nil {
			t.Fatal(err)
		}
		script := string(data)

		if !strings.Contains(script, "--dangerously-skip-permissions") {
			t.Error("install.sh should contain --dangerously-skip-permissions")
		}
		if !strings.Contains(script, "--plugin-dir .devcontainer/ccdc/ccdc-plugin") {
			t.Error("install.sh should contain --plugin-dir .devcontainer/ccdc/ccdc-plugin")
		}
		if !strings.Contains(script, "useradd") {
			t.Error("install.sh should create ccdc user")
		}
	})

	t.Run("install.sh is executable", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(ccdcDir, "install.sh"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o111 == 0 {
			t.Error("install.sh should be executable")
		}
	})

	t.Run("plugin.json content", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "ccdc-plugin", ".claude-plugin", "plugin.json"))
		if err != nil {
			t.Fatal(err)
		}
		var meta map[string]string
		if err := json.Unmarshal(data, &meta); err != nil {
			t.Fatal(err)
		}
		if meta["name"] != "ccdc-plugin" {
			t.Errorf("expected name=ccdc-plugin, got %s", meta["name"])
		}
	})

	t.Run("hooks.json content", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "ccdc-plugin", "hooks", "hooks.json"))
		if err != nil {
			t.Fatal(err)
		}

		var hooks struct {
			Hooks map[string][]struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"hooks"`
		}
		if err := json.Unmarshal(data, &hooks); err != nil {
			t.Fatal(err)
		}

		for _, event := range []string{"Stop", "Notification", "PermissionRequest"} {
			entries, ok := hooks.Hooks[event]
			if !ok {
				t.Errorf("missing hook event: %s", event)
				continue
			}
			if len(entries) == 0 || len(entries[0].Hooks) == 0 {
				t.Errorf("hook event %s has no hooks", event)
				continue
			}
			cmd := entries[0].Hooks[0].Command
			if !strings.Contains(cmd, "notify.sh") {
				t.Errorf("hook command should reference notify.sh, got %s", cmd)
			}
		}
	})

	t.Run("notify.sh content", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "ccdc-plugin", "hooks", "notify.sh"))
		if err != nil {
			t.Fatal(err)
		}
		script := string(data)
		if !strings.Contains(script, "host.docker.internal:${CCDC_PORT}") {
			t.Error("notify.sh should use CCDC_PORT env var for host.docker.internal")
		}
		if !strings.Contains(script, "CCDC_PORT:-5454") {
			t.Error("notify.sh should default CCDC_PORT to 5454")
		}
		if !strings.Contains(script, "|| true") {
			t.Error("notify.sh should suppress errors with || true")
		}
	})
}

func TestGenerate_AlreadyExists(t *testing.T) {
	projectDir := t.TempDir()
	ccdcDir := filepath.Join(projectDir, ".devcontainer", "ccdc")
	if err := os.MkdirAll(ccdcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ccdcDir, "devcontainer-feature.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Generate(projectDir)
	if err == nil {
		t.Fatal("expected error when already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestGenerate_NoClaudeFiles(t *testing.T) {
	projectDir := t.TempDir()

	// fakeHome with empty .claude dir (no files)
	fakeHome := t.TempDir()
	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ccdcDir := filepath.Join(projectDir, ".devcontainer", "ccdc")
	if err := generateFeature(ccdcDir, fakeHome); err != nil {
		t.Fatalf("generateFeature failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(ccdcDir, "devcontainer-feature.json"))
	if err != nil {
		t.Fatal(err)
	}

	var feature map[string]any
	if err := json.Unmarshal(data, &feature); err != nil {
		t.Fatal(err)
	}

	// mounts should be null/empty when no claude files exist
	mounts, _ := feature["mounts"].([]any)
	if len(mounts) != 0 {
		t.Errorf("expected 0 mounts when no claude files exist, got %d", len(mounts))
	}
}
