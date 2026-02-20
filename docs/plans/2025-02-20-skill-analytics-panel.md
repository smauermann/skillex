# Skill Analytics Panel Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the scattered activation banner and budget meter with a dedicated "Skill Analytics" panel in the right column above the markdown preview.

**Architecture:** Split the right column into two stacked bordered panels: a fixed-height analytics panel (top, ~4-5 inner rows) showing activation strength and budget metrics, and the existing SKILL.md viewport (bottom, remaining height). Remove the activation banner from inside the viewport and the budget meter from the help bar.

**Tech Stack:** Go, Bubble Tea, lipgloss, bubbles (viewport, list)

---

### Task 1: Add `progressBar` helper function

**Files:**
- Modify: `internal/tui/tui.go` (add function after `formatK` at line 182)
- Create: `internal/tui/tui_test.go` (new test file)

**Step 1: Write the failing test**

Create `internal/tui/tui_test.go`:

```go
package tui

import "testing"

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		fraction float64
		width    int
		wantFull int
	}{
		{"empty", 0.0, 20, 0},
		{"half", 0.5, 20, 10},
		{"full", 1.0, 20, 20},
		{"over", 1.5, 20, 20},
		{"small width", 0.5, 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := progressBar(tt.fraction, tt.width)
			// Just verify length and non-empty (ANSI codes make exact matching fragile)
			if bar == "" {
				t.Error("expected non-empty progress bar")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestProgressBar -v`
Expected: FAIL — `progressBar` undefined

**Step 3: Write minimal implementation**

Add to `internal/tui/tui.go` after the `formatK` function (after line 182):

```go
// progressBar renders a colored bar of filled (█) and empty (░) blocks.
func progressBar(fraction float64, width int) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction * float64(width))
	empty := width - filled

	var color lipgloss.Color
	switch {
	case fraction >= 1:
		color = lipgloss.Color("196") // red
	case fraction > 0.8:
		color = lipgloss.Color("214") // orange
	default:
		color = lipgloss.Color("35") // green
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	bar += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("░", empty))
	return bar
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestProgressBar -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/tui.go internal/tui/tui_test.go
git commit -m "feat: add progressBar helper for budget visualization"
```

---

### Task 2: Add `renderAnalyticsPanel` function

**Files:**
- Modify: `internal/tui/tui.go` (add new function)
- Modify: `internal/tui/tui_test.go` (add test)

**Step 1: Write the failing test**

Add to `internal/tui/tui_test.go`:

```go
func TestRenderAnalyticsPanel(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "skill-a", Description: "ALWAYS use this skill.", ActivationStyle: discovery.ActivationDirective},
		{Name: "skill-b", Description: "Helps with stuff.", ActivationStyle: discovery.ActivationPassive},
	}

	result := renderAnalyticsPanel(skills[0], skills, 60)
	if result == "" {
		t.Fatal("expected non-empty analytics panel content")
	}
	if !strings.Contains(result, "Activation") {
		t.Error("expected 'Activation' label")
	}
	if !strings.Contains(result, "Budget") {
		t.Error("expected 'Budget' label")
	}
}
```

Add `"strings"` and `"github.com/smauermann/skillex/internal/discovery"` to the test file imports.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRenderAnalyticsPanel -v`
Expected: FAIL — `renderAnalyticsPanel` undefined

**Step 3: Write minimal implementation**

Add to `internal/tui/tui.go` (replace the existing `activationBanner` function around line 112-125):

```go
// analyticsLabelStyle is the left-column label in the analytics panel.
var analyticsLabelStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("243")).
	Bold(true).
	Width(13)

// activationAdvice returns a short one-line explanation for the activation level.
func activationAdvice(style discovery.ActivationStyle) string {
	switch style {
	case discovery.ActivationDirective:
		return "Claude will almost always auto-activate this skill"
	case discovery.ActivationPassive:
		return "Claude may skip this skill — use MUST/ALWAYS/NEVER"
	default:
		return "No activation signals — add directive language"
	}
}

// renderAnalyticsPanel builds the inner content of the Skill Analytics panel.
func renderAnalyticsPanel(skill discovery.Skill, allSkills []discovery.Skill, width int) string {
	tag := activationTag(skill.ActivationStyle)
	advice := activationAdvice(skill.ActivationStyle)

	var adviceColor lipgloss.Color
	switch skill.ActivationStyle {
	case discovery.ActivationDirective:
		adviceColor = directiveColor
	case discovery.ActivationPassive:
		adviceColor = passiveColor
	default:
		adviceColor = neutralColor
	}

	activationLine := analyticsLabelStyle.Render("Activation") +
		tag + "  " +
		lipgloss.NewStyle().Foreground(adviceColor).Render(advice)

	totalChars := totalDescChars(allSkills)
	skillChars := len(skill.Description)
	pct := float64(0)
	if totalChars > 0 {
		pct = float64(skillChars) / float64(descBudgetLimit) * 100
	}
	totalPct := float64(totalChars) / float64(descBudgetLimit)

	budgetLine := analyticsLabelStyle.Render("Budget") +
		fmt.Sprintf("%d / %dk (%.1f%%)", skillChars, descBudgetLimit/1000, pct)

	barWidth := width - 13 - 6 // label width - " NNN%" suffix
	if barWidth < 10 {
		barWidth = 10
	}
	barLine := strings.Repeat(" ", 13) +
		progressBar(totalPct, barWidth) +
		fmt.Sprintf(" %d%%", int(totalPct*100))

	return lipgloss.JoinVertical(lipgloss.Left, activationLine, budgetLine, barLine)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestRenderAnalyticsPanel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/tui.go internal/tui/tui_test.go
git commit -m "feat: add renderAnalyticsPanel with activation and budget rows"
```

---

### Task 3: Integrate analytics panel into the layout

This is the main layout change — split the right column into analytics panel (top) + viewport (bottom).

**Files:**
- Modify: `internal/tui/tui.go` — `View()` method (lines 410-439) and `WindowSizeMsg` handler (lines 247-271)

**Step 1: Update `WindowSizeMsg` handler to account for analytics panel height**

In `internal/tui/tui.go`, update the `tea.WindowSizeMsg` case (around line 247). The analytics panel needs a fixed height. Replace the `WindowSizeMsg` case:

```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		contentHeight := msg.Height - 1 // reserve 1 row for help bar
		listWidth := msg.Width / 3
		viewportWidth := msg.Width - listWidth

		// Content width = total panel width - borders(2) - padding(2)
		listContentWidth := listWidth - 4
		vpContentWidth := viewportWidth - 4

		// Analytics panel: 4 inner rows + top border(1) + bottom border(1) = 6 total rows
		analyticsHeight := 6

		// List panel: full content height minus borders
		listInnerHeight := contentHeight - 2

		// Viewport panel: remaining height after analytics panel and its borders
		vpInnerHeight := contentHeight - analyticsHeight - 2

		m.list.SetSize(listContentWidth, listInnerHeight)

		if !m.ready {
			m.viewport = viewport.New(vpContentWidth, vpInnerHeight)
			m.ready = true
			m = m.updateViewportContent()
		} else {
			m.viewport.Width = vpContentWidth
			m.viewport.Height = vpInnerHeight
		}
```

**Step 2: Update `View()` to render three panels**

Replace the `View()` method:

```go
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	contentHeight := m.height - 1 // reserve for help bar
	listWidth := m.width / 3
	viewportWidth := m.width - listWidth

	// Analytics panel: 4 inner rows
	analyticsInnerHeight := 4
	analyticsHeight := analyticsInnerHeight + 2 // + borders

	// List panel: full content height - borders
	listPanelHeight := contentHeight - 3

	// Viewport panel: remaining
	vpPanelHeight := contentHeight - analyticsHeight - 3

	// Border colors: focused pane gets accent, other is dim
	listBorderColor := focusedBorderColor
	vpBorderColor := blurredBorderColor
	if m.focusViewport {
		listBorderColor = blurredBorderColor
		vpBorderColor = focusedBorderColor
	}

	// Left pane: Skills list
	leftPane := renderPanel("Skills", m.list.View(), listWidth, listPanelHeight, listBorderColor)

	// Right pane top: Skill Analytics
	var analyticsContent string
	if selected, ok := m.list.SelectedItem().(skillItem); ok {
		analyticsContent = renderAnalyticsPanel(selected.skill, m.skills, viewportWidth-4)
	} else {
		analyticsContent = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("No skill selected.")
	}
	analyticsPane := renderPanel("Skill Analytics", analyticsContent, viewportWidth, analyticsInnerHeight, vpBorderColor)

	// Right pane bottom: SKILL.md viewport
	vpPane := renderPanel("SKILL.md", m.viewport.View(), viewportWidth, vpPanelHeight, vpBorderColor)

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, analyticsPane, vpPane)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightColumn)

	return lipgloss.JoinVertical(lipgloss.Left, panes, m.helpBar())
}
```

**Step 3: Run the app to verify layout**

Run: `go run .`
Expected: Three bordered panels visible — Skills (left), Skill Analytics (top-right), SKILL.md (bottom-right). Analytics shows activation tag + advice and budget info.

**Step 4: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat: integrate analytics panel into right-column layout"
```

---

### Task 4: Remove activation banner from viewport content

The activation banner is now in the analytics panel, so remove it from `updateViewportContent`.

**Files:**
- Modify: `internal/tui/tui.go` — `updateViewportContent()` method (lines 302-350)

**Step 1: Remove banner from viewport content**

In `updateViewportContent`, remove the banner lines. Replace the end of the function (lines 346-348):

```go
	// Before (remove these lines):
	// banner := activationBanner(selected.skill.ActivationStyle)
	// m.viewport.SetContent(banner + "\n" + rendered)

	// After:
	m.viewport.SetContent(rendered)
```

**Step 2: Delete the `activationBanner` function**

Remove the `activationBanner` function entirely (lines 109-125). It's replaced by `activationAdvice` + the analytics panel.

**Step 3: Run the app to verify**

Run: `go run .`
Expected: SKILL.md panel starts with frontmatter directly — no activation banner text. Activation info is only in the analytics panel.

**Step 4: Run tests**

Run: `go test ./...`
Expected: All tests pass (no tests reference `activationBanner`)

**Step 5: Commit**

```bash
git add internal/tui/tui.go
git commit -m "refactor: remove activation banner from viewport content"
```

---

### Task 5: Remove budget meter from help bar

The budget is now in the analytics panel, so simplify the help bar.

**Files:**
- Modify: `internal/tui/tui.go` — `helpBar()` method (lines 384-408)

**Step 1: Simplify helpBar**

Replace the `helpBar` method:

```go
func (m Model) helpBar() string {
	key := helpKeyStyle.Render

	var content string
	if m.focusViewport {
		content = key("j/k") + " scroll  " + key("h") + " back to list  " + key("/") + " filter  " + key("q") + " quit"
	} else {
		content = key("j/k") + " navigate  " + key("l") + " read preview  " + key("/") + " filter  " + key("q") + " quit"
	}

	return helpBarStyle.Width(m.width).Render(content)
}
```

**Step 2: Delete the `budgetLabel` function**

Remove the `budgetLabel` function (lines 163-175). It's replaced by the budget row in the analytics panel.

Note: Keep `totalDescChars` and `formatK` — they're still used by `renderAnalyticsPanel`.

**Step 3: Run the app to verify**

Run: `go run .`
Expected: Help bar shows only keybindings. Budget info appears only in the analytics panel.

**Step 4: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 5: Commit**

```bash
git add internal/tui/tui.go
git commit -m "refactor: remove budget meter from help bar"
```

---

### Task 6: Update demo GIF

Per project rules, update the demo after TUI visual changes.

**Files:**
- Modify: `demo.tape` (if sizing adjustments needed)
- Generate: `demo.gif`

**Step 1: Run VHS to generate new demo**

Run: `vhs demo.tape`
Expected: `demo.gif` regenerated with the new three-panel layout

**Step 2: Commit**

```bash
git add demo.gif
git commit -m "chore: update demo with skill analytics panel"
```
