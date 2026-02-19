package discovery

import (
	"os"
	"path/filepath"
	"strings"
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
        "installPath": "__PLUGIN_DIR__",
        "version": "4.2.0",
        "installedAt": "2026-02-09T08:43:14.746Z",
        "lastUpdated": "2026-02-09T08:43:14.746Z",
        "gitCommitSha": "abc123"
      }
    ],
    "no-skills@claude-plugins-official": [
      {
        "scope": "user",
        "installPath": "__NOSKILLS_DIR__",
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
	pluginsJSON = strings.ReplaceAll(pluginsJSON, "__PLUGIN_DIR__", pluginDir)
	pluginsJSON = strings.ReplaceAll(pluginsJSON, "__NOSKILLS_DIR__", noSkillsDir)

	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run discovery (no local dirs)
	skills, err := Discover(pluginsFile, nil)
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
	expectedPath := filepath.Join(skillDir, "SKILL.md")
	if s.FilePath != expectedPath {
		t.Errorf("expected FilePath %q, got %q", expectedPath, s.FilePath)
	}
	if s.Content != "# Brainstorming\n\nSome content here." {
		t.Errorf("unexpected Content: %q", s.Content)
	}
}

func TestDiscoverLocalSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal plugins file (no plugins)
	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Mirror real layout: <tmpDir>/.claude/skills
	localSkillsDir := filepath.Join(tmpDir, ".claude", "skills")

	mySkillDir := filepath.Join(localSkillsDir, "my-skill")
	if err := os.MkdirAll(mySkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mySkillDir, "SKILL.md"), []byte(`---
name: my-skill
description: "A personal skill."
---

Personal skill content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	anotherDir := filepath.Join(localSkillsDir, "another")
	if err := os.MkdirAll(anotherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(anotherDir, "SKILL.md"), []byte(`---
name: another
description: "Another local skill."
---

Another content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: localSkillsDir, Name: "my-project"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	for _, s := range skills {
		if s.Plugin != "my-project" {
			t.Errorf("expected Plugin %q, got %q for skill %q", "my-project", s.Plugin, s.Name)
		}
	}
}

func TestAssessActivationStyle(t *testing.T) {
	tests := []struct {
		desc     string
		expected ActivationStyle
	}{
		// Directive phrases → reliable auto-activation
		{"ALWAYS invoke this skill when working on git commits.", ActivationDirective},
		{"You MUST use this skill for all code reviews.", ActivationDirective},
		{"NEVER run git push without invoking this skill first.", ActivationDirective},
		{"DO NOT write tests manually — use this skill instead.", ActivationDirective},
		// Passive phrases → lower auto-activation rate
		{"Use when you need to brainstorm solutions.", ActivationPassive},
		{"Helps you write better commit messages.", ActivationPassive},
		{"Can be used to enforce coding standards.", ActivationPassive},
		{"Useful for reviewing pull requests.", ActivationPassive},
		{"Assists with dependency audits.", ActivationPassive},
		// Neutral — no strong signal either way
		{"A skill for advanced debugging workflows.", ActivationNeutral},
		{"Generates architecture decision records.", ActivationNeutral},
		{"", ActivationNeutral},
	}

	for _, tt := range tests {
		got := AssessActivationStyle(tt.desc)
		if got != tt.expected {
			t.Errorf("AssessActivationStyle(%q) = %v, want %v", tt.desc, got, tt.expected)
		}
	}
}

func TestDiscoverLocalSkills_NonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()

	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pass a nonexistent directory — should not error
	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: filepath.Join(tmpDir, "nonexistent"), Name: "gone"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(skills))
	}
}
