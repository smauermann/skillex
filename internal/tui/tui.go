package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/smauermann/skillex/internal/discovery"
)

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

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
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

		// Each panel has: 1 top border + 1 bottom border + left/right border chars + padding
		// Inner content height = contentHeight - 3 (top border + bottom border + slack)
		listInnerWidth := listWidth - 4  // border (2) + padding (2)
		vpInnerWidth := viewportWidth - 4
		innerHeight := contentHeight - 3

		m.list.SetSize(listInnerWidth, innerHeight)

		if !m.ready {
			m.viewport = viewport.New(vpInnerWidth, innerHeight)
			m.ready = true
			m = m.updateViewportContent()
		} else {
			m.viewport.Width = vpInnerWidth
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

	rendered, err := m.renderer.Render(selected.skill.Content)
	if err != nil {
		m.viewport.SetContent(fmt.Sprintf("Render error: %v", err))
		return m
	}

	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
	return m
}

// renderPanel draws a bordered panel with the title embedded in the top border line.
func renderPanel(title string, content string, width, height int, borderColor lipgloss.Color) string {
	border := lipgloss.RoundedBorder()

	// Render the body with panelStyle (no top border).
	body := panelStyle.
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Render(content)

	// The rendered body width includes: left border (1) + left padding (1) + content (width) + right padding (1) + right border (1).
	totalWidth := lipgloss.Width(body)

	// Build the top border line with the title embedded.
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(borderColor).Render(title)
	titleVisualWidth := lipgloss.Width(titleStyled)

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Top line: ╭─ Title ─────...─╮
	// prefix = "╭─ " (3 chars visible), suffix = " ─╮" would be too much, let's do:
	// prefix: ╭─  (2 visible) + title + space (1) + fill dashes + ╮ (1)
	prefix := borderStyle.Render(border.TopLeft + border.Top + " ")
	suffix := borderStyle.Render(border.TopRight)

	// Visible width used: 3 (╭─ ) + titleVisualWidth + 1 (space after title) + 1 (╮) = titleVisualWidth + 5
	fillCount := totalWidth - titleVisualWidth - 5
	if fillCount < 0 {
		fillCount = 0
	}
	fill := borderStyle.Render(strings.Repeat(border.Top, fillCount) + " ")

	topLine := prefix + titleStyled + fill + suffix

	return lipgloss.JoinVertical(lipgloss.Left, topLine, body)
}

func (m Model) helpBar() string {
	key := helpKeyStyle.Render
	bar := helpBarStyle.Render

	var help string
	if m.focusViewport {
		help = bar(key("j/k")+" scroll  "+key("h")+" back to list  "+key("/")+" filter  "+key("q")+" quit")
	} else {
		help = bar(key("j/k")+" navigate  "+key("l")+" read preview  "+key("/")+" filter  "+key("q")+" quit")
	}

	// Pad to full width.
	return helpBarStyle.Width(m.width).Render(help)
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

	// Left pane: Skills list in bordered panel
	// panelStyle adds border (1+1) + padding (1+1) = 4 to width, so inner width = listWidth - 4
	leftPane := renderPanel("Skills", m.list.View(), listWidth-4, panelHeight, listBorderColor)

	// Right pane: Viewport in bordered panel
	rightPane := renderPanel("SKILL.md", m.viewport.View(), viewportWidth-4, panelHeight, vpBorderColor)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, panes, m.helpBar())
}
