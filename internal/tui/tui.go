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

// activationBanner returns a human-readable activation hint for the detail panel.
// The PaddingLeft(2) matches Glamour's default block margin so the banner
// aligns with the rendered markdown below it.
func activationBanner(style discovery.ActivationStyle) string {
	base := lipgloss.NewStyle().PaddingLeft(2)
	switch style {
	case discovery.ActivationDirective:
		return base.Foreground(directiveColor).Render(
			"Strong wording — Claude will almost always pick up this skill automatically.")
	case discovery.ActivationPassive:
		return base.Foreground(passiveColor).Render(
			"Weak wording — Claude may skip this skill. Use MUST/ALWAYS/NEVER in the description to improve activation.")
	default:
		return base.Foreground(neutralColor).Render(
			"No activation signals found in description. Add directive language like MUST, ALWAYS, or NEVER to ensure Claude invokes this skill.")
	}
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

// budgetLabel renders the description-budget meter with color that shifts from
// green → orange → red as the 16k limit approaches and is exceeded.
func budgetLabel(used int) string {
	var color lipgloss.Color
	switch {
	case used >= descBudgetLimit:
		color = lipgloss.Color("196") // red — over limit
	case used > descBudgetLimit*8/10:
		color = lipgloss.Color("214") // orange — approaching limit
	default:
		color = lipgloss.Color("35") // green — plenty of room
	}
	text := fmt.Sprintf("desc budget: %s/%dk", formatK(used), descBudgetLimit/1000)
	return lipgloss.NewStyle().Foreground(color).Background(lipgloss.Color("236")).Render(text)
}

func formatK(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
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
		// Content height = total - top border(1) - bottom border(1)
		innerHeight := contentHeight - 2

		m.list.SetSize(listContentWidth, innerHeight)

		if !m.ready {
			m.viewport = viewport.New(vpContentWidth, innerHeight)
			m.ready = true
			m = m.updateViewportContent()
		} else {
			m.viewport.Width = vpContentWidth
			m.viewport.Height = innerHeight
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
	// The banner is lipgloss-styled and must stay outside Glamour to
	// avoid ANSI escape codes being mangled by the markdown renderer.
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

	banner := activationBanner(selected.skill.ActivationStyle)
	m.viewport.SetContent(banner + "\n" + rendered)
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

	var leftText string
	if m.focusViewport {
		leftText = key("j/k") + " scroll  " + key("h") + " back to list  " + key("/") + " filter  " + key("q") + " quit"
	} else {
		leftText = key("j/k") + " navigate  " + key("l") + " read preview  " + key("/") + " filter  " + key("q") + " quit"
	}

	rightText := budgetLabel(totalDescChars(m.skills))

	// Inner content width: helpBarStyle has Padding(0,1) so subtract 2 from total.
	innerW := m.width - 2
	if innerW < 0 {
		innerW = 0
	}
	pad := innerW - lipgloss.Width(leftText) - lipgloss.Width(rightText)
	if pad < 1 {
		pad = 1
	}

	content := leftText + strings.Repeat(" ", pad) + rightText
	return helpBarStyle.Width(m.width).Render(content)
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	contentHeight := m.height - 1 // reserve for help bar
	listWidth := m.width / 3
	viewportWidth := m.width - listWidth

	// Panel inner height: total content area minus top border (1) + bottom border (1) + top/bottom padding from border
	panelHeight := contentHeight - 3

	// Border colors: focused pane gets accent, other is dim
	listBorderColor := focusedBorderColor
	vpBorderColor := blurredBorderColor
	if m.focusViewport {
		listBorderColor = blurredBorderColor
		vpBorderColor = focusedBorderColor
	}

	// Left pane: Skills list in bordered panel (pass total panel width)
	leftPane := renderPanel("Skills", m.list.View(), listWidth, panelHeight, listBorderColor)

	// Right pane: Viewport in bordered panel
	rightPane := renderPanel("SKILL.md", m.viewport.View(), viewportWidth, panelHeight, vpBorderColor)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, panes, m.helpBar())
}
