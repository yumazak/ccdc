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
	if err := os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	ccdcDir := filepath.Join(projectDir, ".devcontainer", "ccdc")

	if err := generateFeature(ccdcDir, fakeHome); err != nil {
		t.Fatalf("generateFeature failed: %v", err)
	}

	t.Run("feature files exist", func(t *testing.T) {
		for _, name := range []string{"devcontainer-feature.json", "install.sh"} {
			p := filepath.Join(ccdcDir, name)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected %s to exist: %v", name, err)
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

		// We created CLAUDE.md, so expect 1 mount
		if len(mounts) != 1 {
			t.Errorf("expected 1 mount, got %d", len(mounts))
		}

		for _, m := range mounts {
			mount := m.(map[string]any)
			source := mount["source"].(string)
			if !strings.HasPrefix(source, "${localEnv:HOME}/.claude/") {
				t.Errorf("mount source should use ${localEnv:HOME}, got %s", source)
			}
		}
	})

	t.Run("install.sh content", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(ccdcDir, "install.sh"))
		if err != nil {
			t.Fatal(err)
		}
		script := string(data)

		if !strings.Contains(script, "--dangerously-skip-permissions") {
			t.Error("install.sh should contain --dangerously-skip-permissions")
		}
		if strings.Contains(script, "--plugin-dir") {
			t.Error("install.sh should not contain --plugin-dir")
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

	mounts, _ := feature["mounts"].([]any)
	if len(mounts) != 0 {
		t.Errorf("expected 0 mounts when no claude files exist, got %d", len(mounts))
	}
}
