package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	list          list.Model
	viewport      viewport.Model
	skills        []discovery.Skill
	styleOpt      glamour.TermRendererOption
	renderer      *glamour.TermRenderer
	rendererWidth int
	width         int
	height        int
	ready         bool
}

// New creates a new TUI model from discovered skills.
func New(skills []discovery.Skill, styleOpt glamour.TermRendererOption) Model {
	items := make([]list.Item, len(skills))
	for i, s := range skills {
		items[i] = skillItem{skill: s}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Skills"
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
