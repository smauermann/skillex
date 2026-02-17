package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlugins(t *testing.T) {
	// Create a temp directory with a fake installed_plugins.json
	tmpDir := t.TempDir()

	pluginsJSON := `{
  "version": 2,
  "plugins": {
    "superpowers@claude-plugins-official": [
      {
        "scope": "user",
        "installPath": "PLACEHOLDER",
        "version": "4.2.0",
        "installedAt": "2026-02-09T08:43:14.746Z",
        "lastUpdated": "2026-02-09T08:43:14.746Z",
        "gitCommitSha": "abc123"
      }
    ],
    "no-skills@claude-plugins-official": [
      {
        "scope": "user",
        "installPath": "PLACEHOLDER_NOSKILLS",
        "version": "1.0.0",
        "installedAt": "2026-01-28T22:52:44.362Z",
        "lastUpdated": "2026-01-28T22:52:44.362Z",
        "gitCommitSha": "def456"
      }
    ]
  }
}`

	// Create fake plugin directory with a skill
	pluginDir := filepath.Join(tmpDir, "superpowers")
	skillDir := filepath.Join(pluginDir, "skills", "brainstorming")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	skillContent := `---
name: brainstorming
description: "Explores user intent before implementation."
---

# Brainstorming

Some content here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a plugin dir with no skills/ subdirectory
	noSkillsDir := filepath.Join(tmpDir, "no-skills")
	if err := os.MkdirAll(noSkillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Patch the JSON with real paths
	pluginsJSON = replaceAll(pluginsJSON, "PLACEHOLDER_NOSKILLS", noSkillsDir)
	pluginsJSON = replaceAll(pluginsJSON, "PLACEHOLDER", pluginDir)

	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run discovery
	skills, err := Discover(pluginsFile)
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	s := skills[0]
	if s.Name != "brainstorming" {
		t.Errorf("expected name 'brainstorming', got %q", s.Name)
	}
	if s.Description != "Explores user intent before implementation." {
		t.Errorf("unexpected description: %q", s.Description)
	}
	if s.Plugin != "superpowers" {
		t.Errorf("expected plugin 'superpowers', got %q", s.Plugin)
	}
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}
