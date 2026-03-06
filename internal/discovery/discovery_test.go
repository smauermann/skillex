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
	if !strings.Contains(s.Frontmatter, "name: brainstorming") {
		t.Errorf("expected Frontmatter to contain 'name: brainstorming', got %q", s.Frontmatter)
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

func TestSkillFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()

	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	localDir := filepath.Join(tmpDir, "skills")
	skillDir := filepath.Join(localDir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: my-skill
description: "A test skill."
license: MIT
---

Body content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: localDir, Name: "test"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	s := skills[0]
	if s.Frontmatter == "" {
		t.Fatal("expected non-empty Frontmatter")
	}
	if !strings.Contains(s.Frontmatter, "license: MIT") {
		t.Errorf("expected Frontmatter to contain 'license: MIT', got %q", s.Frontmatter)
	}
	if !strings.Contains(s.Frontmatter, "name: my-skill") {
		t.Errorf("expected Frontmatter to contain 'name: my-skill', got %q", s.Frontmatter)
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

func TestDiscoverDisabledSkill(t *testing.T) {
	tmpDir := t.TempDir()

	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	localDir := filepath.Join(tmpDir, "skills")
	skillDir := filepath.Join(localDir, "disabled-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md.disabled"), []byte(`---
name: disabled-skill
description: "A disabled skill."
---

Disabled content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: localDir, Name: "test"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	s := skills[0]
	if s.Enabled {
		t.Error("expected Enabled=false for SKILL.md.disabled")
	}
	if s.Name != "disabled-skill" {
		t.Errorf("expected name 'disabled-skill', got %q", s.Name)
	}
	if !strings.HasSuffix(s.FilePath, "SKILL.md.disabled") {
		t.Errorf("expected FilePath to end with SKILL.md.disabled, got %q", s.FilePath)
	}
}

func TestDiscoverPrefersEnabledWhenBothExist(t *testing.T) {
	tmpDir := t.TempDir()

	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	localDir := filepath.Join(tmpDir, "skills")
	skillDir := filepath.Join(localDir, "both-exist")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := []byte("---\nname: both-exist\ndescription: \"test\"\n---\nBody.\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md.disabled"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: localDir, Name: "test"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if !skills[0].Enabled {
		t.Error("expected Enabled=true when both SKILL.md and SKILL.md.disabled exist")
	}
	if !strings.HasSuffix(skills[0].FilePath, "SKILL.md") || strings.HasSuffix(skills[0].FilePath, ".disabled") {
		t.Errorf("expected FilePath to point to SKILL.md, got %q", skills[0].FilePath)
	}
}

func TestToggleSkillRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("---\nname: my-skill\n---\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	skill := Skill{
		Name:     "my-skill",
		FilePath: skillFile,
		Enabled:  true,
	}

	// Disable
	if err := ToggleSkill(&skill); err != nil {
		t.Fatalf("ToggleSkill (disable) error: %v", err)
	}
	if skill.Enabled {
		t.Error("expected Enabled=false after disable")
	}
	if !strings.HasSuffix(skill.FilePath, ".disabled") {
		t.Errorf("expected .disabled suffix, got %q", skill.FilePath)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md.disabled")); err != nil {
		t.Error("expected SKILL.md.disabled to exist on disk")
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("expected SKILL.md to not exist on disk")
	}

	// Re-enable
	if err := ToggleSkill(&skill); err != nil {
		t.Fatalf("ToggleSkill (enable) error: %v", err)
	}
	if !skill.Enabled {
		t.Error("expected Enabled=true after re-enable")
	}
	if strings.HasSuffix(skill.FilePath, ".disabled") {
		t.Errorf("expected no .disabled suffix, got %q", skill.FilePath)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Error("expected SKILL.md to exist on disk")
	}
}

func TestEnabledSkillHasEnabledTrue(t *testing.T) {
	tmpDir := t.TempDir()

	pluginsJSON := `{"version": 2, "plugins": {}}`
	pluginsFile := filepath.Join(tmpDir, "installed_plugins.json")
	if err := os.WriteFile(pluginsFile, []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	localDir := filepath.Join(tmpDir, "skills")
	skillDir := filepath.Join(localDir, "enabled-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: enabled-skill
description: "An enabled skill."
---

Content.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, err := Discover(pluginsFile, []LocalSkillsDir{
		{Path: localDir, Name: "test"},
	})
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if !skills[0].Enabled {
		t.Error("expected Enabled=true for SKILL.md")
	}
}
