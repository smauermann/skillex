# Skillex TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go TUI that lists all installed Claude Code skills and shows rendered markdown preview in a split pane.

**Architecture:** Single-binary Bubble Tea app. Skill discovery reads `~/.claude/plugins/installed_plugins.json` and walks each plugin's `skills/` directory. Split-pane layout with a filterable list on the left and a glamour-rendered markdown viewport on the right.

**Tech Stack:** Go, bubbletea, bubbles (list + viewport), lipgloss, glamour, gopkg.in/yaml.v3

---

### Task 1: Initialize Go module and install dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Step 1: Initialize module**

Run: `go mod init github.com/smauermann/skillex`

**Step 2: Install dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/glamour@latest
go get gopkg.in/yaml.v3@latest
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: init go module with dependencies"
```

---

### Task 2: Skill discovery — data types and JSON parsing

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/discovery_test.go`

**Step 1: Write the failing test for parsing installed_plugins.json**

```go
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
	// simple helper to avoid importing strings in test
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery/ -v`
Expected: FAIL — `Discover` not defined

**Step 3: Write the implementation**

```go
package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a single discovered Claude Code skill.
type Skill struct {
	Name        string
	Description string
	Plugin      string
	FilePath    string
	Content     string
}

type installedPlugins struct {
	Version int                          `json:"version"`
	Plugins map[string][]pluginInstance  `json:"plugins"`
}

type pluginInstance struct {
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Discover reads installed_plugins.json and finds all skills.
func Discover(pluginsFile string) ([]Skill, error) {
	data, err := os.ReadFile(pluginsFile)
	if err != nil {
		return nil, fmt.Errorf("reading plugins file: %w", err)
	}

	var installed installedPlugins
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil, fmt.Errorf("parsing plugins file: %w", err)
	}

	var skills []Skill
	for key, instances := range installed.Plugins {
		if len(instances) == 0 {
			continue
		}
		// Use first instance (most recently registered)
		inst := instances[0]

		// Extract plugin name: "superpowers@claude-plugins-official" -> "superpowers"
		pluginName := key
		if idx := strings.Index(key, "@"); idx != -1 {
			pluginName = key[:idx]
		}

		skillsDir := filepath.Join(inst.InstallPath, "skills")
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			// Plugin has no skills/ directory — skip
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			content, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			fm, body := parseFrontmatter(content)

			name := fm.Name
			if name == "" {
				name = entry.Name()
			}

			skills = append(skills, Skill{
				Name:        name,
				Description: fm.Description,
				Plugin:      pluginName,
				FilePath:    skillFile,
				Content:     body,
			})
		}
	}

	return skills, nil
}

// parseFrontmatter extracts YAML frontmatter from markdown content.
func parseFrontmatter(content []byte) (frontmatter, string) {
	var fm frontmatter

	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return fm, string(content)
	}

	// Find closing ---
	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return fm, string(content)
	}

	yamlBlock := rest[:idx]
	body := rest[idx+4:] // skip \n---

	_ = yaml.Unmarshal(yamlBlock, &fm)

	return fm, string(bytes.TrimSpace(body))
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat: add skill discovery from installed_plugins.json"
```

---

### Task 3: TUI model — split pane with list and viewport

**Files:**
- Create: `internal/tui/tui.go`

**Step 1: Write the TUI model**

```go
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/smauermann/skillex/internal/discovery"
)

var (
	listStyle = lipgloss.NewStyle().
			Padding(1, 2)

	viewportStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderLeft(true).
			BorderForeground(lipgloss.Color("62"))
)

// skillItem implements list.Item for a Skill.
type skillItem struct {
	skill discovery.Skill
}

func (i skillItem) Title() string       { return i.skill.Name }
func (i skillItem) Description() string { return i.skill.Plugin }
func (i skillItem) FilterValue() string { return i.skill.Name + " " + i.skill.Plugin }

// Model is the top-level Bubble Tea model.
type Model struct {
	list     list.Model
	viewport viewport.Model
	skills   []discovery.Skill
	width    int
	height   int
	ready    bool
}

// New creates a new TUI model from discovered skills.
func New(skills []discovery.Skill) Model {
	items := make([]list.Item, len(skills))
	for i, s := range skills {
		items[i] = skillItem{skill: s}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Skills"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)

	return Model{
		list:   l,
		skills: skills,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		listWidth := msg.Width / 3
		viewportWidth := msg.Width - listWidth - 4 // account for border + padding

		m.list.SetSize(listWidth, msg.Height)

		if !m.ready {
			m.viewport = viewport.New(viewportWidth, msg.Height-4)
			m.ready = true
			m = m.updateViewportContent()
		} else {
			m.viewport.Width = viewportWidth
			m.viewport.Height = msg.Height - 4
		}
	}

	// Track selection before update
	prevIndex := m.list.Index()

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport content if selection changed
	if m.list.Index() != prevIndex {
		m = m.updateViewportContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateViewportContent() Model {
	selected, ok := m.list.SelectedItem().(skillItem)
	if !ok {
		m.viewport.SetContent("No skill selected.")
		return m
	}

	width := m.viewport.Width - 2
	if width < 20 {
		width = 20
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
		return m
	}

	rendered, err := renderer.Render(selected.skill.Content)
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
		return m
	}

	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
	return m
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	listWidth := m.width / 3
	viewportWidth := m.width - listWidth

	listView := listStyle.Width(listWidth).Height(m.height).Render(m.list.View())
	vpView := viewportStyle.Width(viewportWidth - 4).Height(m.height - 2).Render(m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, vpView)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/tui/`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/tui/
git commit -m "feat: add split-pane TUI with list and markdown viewport"
```

---

### Task 4: Main entry point — wire it all together

**Files:**
- Create: `main.go`

**Step 1: Write main.go**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/smauermann/skillex/internal/tui"
	"github.com/smauermann/skillex/internal/discovery"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pluginsFile := filepath.Join(homeDir, ".claude", "plugins", "installed_plugins.json")

	skills, err := discovery.Discover(pluginsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering skills: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found.")
		os.Exit(0)
	}

	p := tea.NewProgram(tui.New(skills), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Build and smoke test**

Run: `go build -o skillex . && ./skillex`
Expected: TUI launches with skill list on left, rendered markdown on right. Navigate with j/k, filter with /, quit with q.

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: add main entry point wiring discovery to TUI"
```

---

### Task 5: Add .gitignore and final cleanup

**Files:**
- Create: `.gitignore`

**Step 1: Create .gitignore**

```
skillex
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add gitignore"
```
