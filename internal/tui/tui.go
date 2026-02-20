package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/smauermann/skillex/internal/discovery"
)

// descBudgetLimit is the fallback character budget for all skill descriptions
// combined in Claude Code's available_skills system prompt section.
// The actual limit scales dynamically at 2% of the model's context window,
// with 16,000 chars as the documented fallback. Skills that don't fit are
// silently excluded with no warning.
// Source: https://github.com/anthropics/claude-code/issues/13099
const descBudgetLimit = 16_000

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			Padding(0, 1)

	focusedBorderColor = lipgloss.Color("62")
	blurredBorderColor = lipgloss.Color("240")

	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Bold(true)

	// Skill list item styles.
	normalTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	normalDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedDescStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	cursorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)

	// Activation tag colors.
	directiveColor = lipgloss.Color("35")  // green  — directive descriptions activate reliably
	passiveColor   = lipgloss.Color("214") // orange — passive descriptions often ignored
	neutralColor   = lipgloss.Color("242") // dim    — unclear / no description

	// analyticsLabelStyle is the left-column label in the analytics panel.
	analyticsLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Bold(true).
				Width(13)
)

// skillItem implements list.Item for a Skill.
type skillItem struct {
	skill discovery.Skill
}

func (i skillItem) Title() string       { return i.skill.Name }
func (i skillItem) Description() string { return i.skill.Plugin }
func (i skillItem) FilterValue() string { return i.skill.Name + " " + i.skill.Plugin }

// skillDelegate is a custom list.ItemDelegate that renders each skill with an
// activation-style tag next to the plugin name.
type skillDelegate struct{}

func (d skillDelegate) Height() int                             { return 2 }
func (d skillDelegate) Spacing() int                            { return 1 }
func (d skillDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d skillDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(skillItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	var prefix string
	var tStyle, dStyle lipgloss.Style
	if isSelected {
		prefix = cursorStyle.Render("> ")
		tStyle = selectedTitleStyle
		dStyle = selectedDescStyle
	} else {
		prefix = "  "
		tStyle = normalTitleStyle
		dStyle = normalDescStyle
	}

	tag := activationTag(si.skill.ActivationStyle)
	fmt.Fprintf(w, "%s%s\n  %s %s", prefix, tStyle.Render(si.skill.Name), dStyle.Render(si.skill.Plugin), tag)
}

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
		tag +
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" — ") +
		lipgloss.NewStyle().Foreground(adviceColor).Render(advice)

	skillChars := len(skill.Description)
	descLine := analyticsLabelStyle.Render("Description") +
		fmt.Sprintf("%d chars", skillChars)

	totalChars := totalDescChars(allSkills)
	totalPct := float64(totalChars) / float64(descBudgetLimit)

	budgetLine := analyticsLabelStyle.Render("Budget") +
		fmt.Sprintf("%d / %d chars", totalChars, descBudgetLimit)

	barWidth := width - 13 - 6 // label width - " NNN%" suffix
	if barWidth < 10 {
		barWidth = 10
	}
	barLine := strings.Repeat(" ", 13) +
		progressBar(totalPct, barWidth) +
		fmt.Sprintf(" %d%%", int(totalPct*100))

	var legend string
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	switch {
	case totalChars >= descBudgetLimit:
		legend = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(
			strings.Repeat(" ", 13) + "Over limit — skills are being silently excluded")
	case totalChars > descBudgetLimit*8/10:
		legend = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(
			strings.Repeat(" ", 13) + "Approaching limit — consider shortening descriptions")
	default:
		legend = dimStyle.Render(
			strings.Repeat(" ", 13) + "Within budget — all skill descriptions fit")
	}

	return lipgloss.JoinVertical(lipgloss.Left, activationLine, descLine, budgetLine, barLine, legend)
}

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
		// Only simple key: value lines are rendered; multi-line YAML
		// values and list items are skipped.
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

// totalDescChars returns the sum of description lengths across all skills.
// Claude Code silently stops loading skills when this total exceeds 15,000 chars.
func totalDescChars(skills []discovery.Skill) int {
	total := 0
	for _, s := range skills {
		total += len(s.Description)
	}
	return total
}

// progressBar renders a colored bar of filled and empty blocks.
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

	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("\u2588", filled))
	bar += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("\u2591", empty))
	return bar
}

// Model is the top-level Bubble Tea model.
type Model struct {
	list          list.Model
	viewport      viewport.Model
	skills        []discovery.Skill
	styleOpt      glamour.TermRendererOption
	renderer      *glamour.TermRenderer
	rendererWidth int
	width         int
	height        int
	ready         bool
	focusViewport bool
}

// New creates a new TUI model from discovered skills.
func New(skills []discovery.Skill, styleOpt glamour.TermRendererOption) Model {
	items := make([]list.Item, len(skills))
	for i, s := range skills {
		items[i] = skillItem{skill: s}
	}

	l := list.New(items, skillDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)

	return Model{
		list:     l,
		skills:   skills,
		styleOpt: styleOpt,
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
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "l":
			if !m.focusViewport {
				m.focusViewport = true
				return m, nil
			}
		case "h":
			if m.focusViewport {
				m.focusViewport = false
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		contentHeight := msg.Height - 1 // reserve 1 row for help bar
		listWidth := msg.Width / 3
		viewportWidth := msg.Width - listWidth

		// Content width = total panel width - borders(2) - padding(2)
		listContentWidth := listWidth - 4
		vpContentWidth := viewportWidth - 4

		// Analytics panel: 6 inner rows + top border(1) + bottom border(1) = 8 total rows
		analyticsHeight := 8

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
	}

	// Route key events to the focused pane only.
	if m.focusViewport {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		prevIndex := m.list.Index()

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		if m.list.Index() != prevIndex {
			m = m.updateViewportContent()
		}
	}

	// Always forward non-key messages (like WindowSizeMsg) to both.
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

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

	// Recreate renderer only when width changes.
	if m.renderer == nil || width != m.rendererWidth {
		r, err := glamour.NewTermRenderer(
			m.styleOpt,
			glamour.WithWordWrap(width),
		)
		if err != nil {
			m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
			return m
		}
		m.renderer = r
		m.rendererWidth = width
	}

	// Build the markdown content: frontmatter + separator + body.
	var md strings.Builder
	fm := renderFrontmatter(selected.skill.Frontmatter)
	if fm != "" {
		md.WriteString("---\n\n")
		md.WriteString(fm)
		md.WriteString("---\n\n")
	}
	md.WriteString(selected.skill.Content)

	rendered, err := m.renderer.Render(md.String())
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
		return m
	}

	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
	return m
}

// renderPanel draws a bordered panel with the title embedded in the top border line.
// totalWidth is the full desired width of the panel including borders.
func renderPanel(title string, content string, totalWidth, height int, borderColor lipgloss.Color) string {
	border := lipgloss.RoundedBorder()

	// lipgloss .Width() includes padding but excludes borders.
	bodyWidth := totalWidth - 2
	body := panelStyle.
		BorderForeground(borderColor).
		Width(bodyWidth).
		Height(height).
		Render(content)

	// Build the top border line to match body width exactly.
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(title)
	titleWidth := lipgloss.Width(titleStyled)
	colorStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Layout: ╭─ Title ─────...──╮
	// Chars: ╭(1) ─(1) space(1) title(N) space(1) fill(F) ╮(1) = totalWidth
	fillCount := totalWidth - titleWidth - 5
	if fillCount < 0 {
		fillCount = 0
	}

	topLine := colorStyle.Render(border.TopLeft+border.Top+" ") +
		titleStyled +
		colorStyle.Render(" "+strings.Repeat(border.Top, fillCount)+border.TopRight)

	return lipgloss.JoinVertical(lipgloss.Left, topLine, body)
}

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

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	contentHeight := m.height - 1 // reserve for help bar
	listWidth := m.width / 3
	viewportWidth := m.width - listWidth

	// Analytics panel: 6 inner rows
	analyticsInnerHeight := 6
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
