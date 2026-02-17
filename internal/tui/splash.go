package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/smauermann/skillex/internal/discovery"
)

const logo = `
     _____ __ __ _____ __    __    _____ __ __
    |   __|  |  |     |  |  |  |  |   __|  |  |
    |__   |    -|-   -|  |__|  |__|   __|-   -|
    |_____|__|__|_____|_____|_____|_____|__|__|
`

var (
	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	taglineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))
)

// skillsLoadedMsg is sent when skill discovery completes.
type skillsLoadedMsg struct {
	skills []discovery.Skill
	err    error
}

// SplashModel shows a splash screen while loading skills.
type SplashModel struct {
	spinner     spinner.Model
	pluginsFile string
	styleOpt    glamour.TermRendererOption
	width       int
	height      int
	done        bool
	skills      []discovery.Skill
	err         error
}

// NewSplash creates the splash screen model.
func NewSplash(pluginsFile string, styleOpt glamour.TermRendererOption) SplashModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return SplashModel{
		spinner:     s,
		pluginsFile: pluginsFile,
		styleOpt:    styleOpt,
	}
}

func (m SplashModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.discoverSkills(),
	)
}

func (m SplashModel) discoverSkills() tea.Cmd {
	return func() tea.Msg {
		skills, err := discovery.Discover(m.pluginsFile)
		return skillsLoadedMsg{skills: skills, err: err}
	}
}

func (m SplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case skillsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, nil
		}
		if len(msg.skills) == 0 {
			m.err = fmt.Errorf("no skills found")
			m.done = true
			return m, nil
		}
		// Transition to main TUI.
		mainModel := New(msg.skills, m.styleOpt)
		// Send a WindowSizeMsg so the main model initializes with the right size.
		return mainModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m SplashModel) View() string {
	if m.done && m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err)
	}

	logoRendered := logoStyle.Render(logo)
	tagline := taglineStyle.Render("Claude Code skill explorer")
	loading := fmt.Sprintf("\n  %s Discovering skills...", m.spinner.View())

	content := lipgloss.JoinVertical(lipgloss.Center,
		logoRendered,
		tagline,
		loading,
	)

	// Center on screen.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
