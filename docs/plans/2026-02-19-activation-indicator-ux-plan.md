# Activation Indicator UX Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the opaque colored dot in the skill list with self-explanatory labels and add activation context + full frontmatter to the detail panel.

**Architecture:** Three changes: (1) `parseFrontmatter` returns the raw YAML string and the `Skill` struct stores it, (2) list delegate renders a colored word tag instead of a dot, (3) detail panel prepends an activation banner and styled frontmatter fields above the body.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Glamour

---

### Task 1: Add raw frontmatter to Skill struct and parseFrontmatter

**Files:**
- Modify: `internal/discovery/discovery.go:48-55` (Skill struct)
- Modify: `internal/discovery/discovery.go:74-111` (discoverSkillsInDir)
- Modify: `internal/discovery/discovery.go:154-176` (parseFrontmatter)

**Step 1: Write the failing test**

Add to `internal/discovery/discovery_test.go` after `TestLoadPlugins`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery/ -run TestSkillFrontmatter -v`
Expected: FAIL — `Skill` has no `Frontmatter` field.

**Step 3: Implement the changes**

In `internal/discovery/discovery.go`:

1. Add `Frontmatter string` to the `Skill` struct (after `Content`).

2. Change `parseFrontmatter` signature to return four values — add the raw YAML string:

```go
func parseFrontmatter(content []byte) (frontmatter, string, string, error) {
	var fm frontmatter

	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return fm, "", string(content), nil
	}

	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return fm, "", string(content), nil
	}

	yamlBlock := rest[:idx]
	body := rest[idx+4:]

	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return fm, "", string(content), err
	}

	return fm, string(bytes.TrimSpace(yamlBlock)), string(bytes.TrimSpace(body)), nil
}
```

3. Update `discoverSkillsInDir` to capture and store the raw YAML:

```go
fm, rawFM, body, err := parseFrontmatter(content)
if err != nil {
	continue
}

// ...

skills = append(skills, Skill{
	Name:            name,
	Description:     fm.Description,
	Plugin:          pluginName,
	FilePath:        skillFile,
	Content:         body,
	Frontmatter:     rawFM,
	ActivationStyle: AssessActivationStyle(fm.Description),
})
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/ -v`
Expected: ALL PASS (including existing tests — they still work because `parseFrontmatter` now returns 4 values and all call sites are updated).

**Step 5: Commit**

```bash
git add internal/discovery/discovery.go internal/discovery/discovery_test.go
git commit -m "feat: store raw frontmatter YAML on Skill struct"
```

---

### Task 2: Replace activation dot with colored tag label in list

**Files:**
- Modify: `internal/tui/tui.go:50-54` (color comment)
- Modify: `internal/tui/tui.go:65-107` (delegate + activationDot)

**Step 1: Rename `activationDot` to `activationTag` and render a word**

Replace the `activationDot` function (lines 97-107) with:

```go
// activationTag returns a colored word indicating auto-activation reliability.
func activationTag(style discovery.ActivationStyle) string {
	switch style {
	case discovery.ActivationDirective:
		return lipgloss.NewStyle().Foreground(directiveColor).Render("directive")
	case discovery.ActivationPassive:
		return lipgloss.NewStyle().Foreground(passiveColor).Render("passive")
	default:
		return lipgloss.NewStyle().Foreground(neutralColor).Render("unknown")
	}
}
```

**Step 2: Update the Render method to use `activationTag`**

Change line 93-94 from:

```go
dot := activationDot(si.skill.ActivationStyle)
fmt.Fprintf(w, "%s%s\n  %s %s", prefix, tStyle.Render(si.skill.Name), dStyle.Render(si.skill.Plugin), dot)
```

to:

```go
tag := activationTag(si.skill.ActivationStyle)
fmt.Fprintf(w, "%s%s\n  %s %s", prefix, tStyle.Render(si.skill.Name), dStyle.Render(si.skill.Plugin), tag)
```

**Step 3: Update the comment on `skillDelegate`**

Change line 66 comment from "activation-style indicator dot" to "activation-style tag".

Update the color var comment (line 50) from "Activation indicator dot colors." to "Activation tag colors."

**Step 4: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat: replace activation dot with colored tag label"
```

---

### Task 3: Add activation banner and frontmatter to detail panel

**Files:**
- Modify: `internal/tui/tui.go:260-295` (updateViewportContent)

**Step 1: Add `activationBanner` function**

Add this function after `activationTag`:

```go
// activationBanner returns a human-readable activation hint for the detail panel.
func activationBanner(style discovery.ActivationStyle) string {
	switch style {
	case discovery.ActivationDirective:
		return lipgloss.NewStyle().Foreground(directiveColor).Render(
			"Strong wording — Claude will almost always pick up this skill automatically.")
	case discovery.ActivationPassive:
		return lipgloss.NewStyle().Foreground(passiveColor).Render(
			"Weak wording — Claude may skip this skill. Use MUST/ALWAYS/NEVER in the description to improve activation.")
	default:
		return lipgloss.NewStyle().Foreground(neutralColor).Render(
			"No activation signals found in description. Add directive language like MUST, ALWAYS, or NEVER to ensure Claude invokes this skill.")
	}
}
```

**Step 2: Add `renderFrontmatter` function**

Add this function after `activationBanner`:

```go
// renderFrontmatter converts raw YAML frontmatter into styled bold key/value markdown lines.
func renderFrontmatter(raw string) string {
	if raw == "" {
		return ""
	}
	var buf strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			// Strip surrounding quotes from YAML values.
			val = strings.Trim(val, "\"'")
			buf.WriteString(fmt.Sprintf("**%s:** %s\n\n", key, val))
		}
	}
	return buf.String()
}
```

**Step 3: Update `updateViewportContent` to prepend banner + frontmatter**

Replace the rendering section (lines 286-293) with:

```go
	// Build the detail content: banner + frontmatter + separator + body.
	var detail strings.Builder
	detail.WriteString(activationBanner(selected.skill.ActivationStyle))
	detail.WriteString("\n\n")
	fm := renderFrontmatter(selected.skill.Frontmatter)
	if fm != "" {
		detail.WriteString(fm)
		detail.WriteString("---\n\n")
	}
	detail.WriteString(selected.skill.Content)

	rendered, err := m.renderer.Render(detail.String())
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
		return m
	}

	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
	return m
```

**Step 4: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

**Step 5: Manual smoke test**

Run: `go run . 2>/dev/null` (or however the binary is invoked)
Verify:
- List shows colored `directive`/`passive`/`unknown` tags instead of dots
- Detail panel shows activation banner at top
- Frontmatter fields rendered as bold key/value pairs
- Horizontal rule separates frontmatter from body
- Scrolling still works

**Step 6: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat: add activation banner and frontmatter to detail panel"
```

---

### Task 4: Update existing test assertions for Frontmatter field

**Files:**
- Modify: `internal/discovery/discovery_test.go`

**Step 1: Add Frontmatter assertion to TestLoadPlugins**

After line 101 (`s.Content` check), add:

```go
if !strings.Contains(s.Frontmatter, "name: brainstorming") {
	t.Errorf("expected Frontmatter to contain 'name: brainstorming', got %q", s.Frontmatter)
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add internal/discovery/discovery_test.go
git commit -m "test: add frontmatter assertions to existing tests"
```
