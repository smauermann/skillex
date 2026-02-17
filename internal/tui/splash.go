package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/smauermann/skillex/internal/discovery"
)

const logo = `
███████╗ ██╗  ██╗ ██╗ ██╗      ██╗      ███████╗ ██╗  ██╗
██╔════╝ ██║ ██╔╝ ██║ ██║      ██║      ██╔════╝ ╚██╗██╔╝
███████╗ █████╔╝  ██║ ██║      ██║      █████╗    ╚███╔╝
╚════██║ ██╔═██╗  ██║ ██║      ██║      ██╔══╝    ██╔██╗
███████║ ██║  ██╗ ██║ ███████╗ ███████╗ ███████╗ ██╔╝ ██╗
╚══════╝ ╚═╝  ╚═╝ ╚═╝ ╚══════╝ ╚══════╝ ╚══════╝ ╚═╝  ╚═╝
`

var (
	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
)

// skillsLoadedMsg is sent when skill discovery completes.
type skillsLoadedMsg struct {
	skills []discovery.Skill
	err    error
}

// SplashModel shows a splash screen until the user presses Enter.
type SplashModel struct {
	pluginsFile  string
	localDirs    []discovery.LocalSkillsDir
	styleOpt     glamour.TermRendererOption
	width        int
	height       int
	err          error
	skills       []discovery.Skill
	skillsLoaded bool
}

// NewSplash creates the splash screen model.
func NewSplash(pluginsFile string, localDirs []discovery.LocalSkillsDir, styleOpt glamour.TermRendererOption) SplashModel {
	return SplashModel{
		pluginsFile: pluginsFile,
		localDirs:   localDirs,
		styleOpt:    styleOpt,
	}
}

func (m SplashModel) Init() tea.Cmd {
	return m.discoverSkills()
}

func (m SplashModel) discoverSkills() tea.Cmd {
	return func() tea.Msg {
		skills, err := discovery.Discover(m.pluginsFile, m.localDirs)
		return skillsLoadedMsg{skills: skills, err: err}
	}
}

func (m SplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.skillsLoaded && len(m.skills) > 0 {
				mainModel := New(m.skills, m.styleOpt)
				return mainModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case skillsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if len(msg.skills) == 0 {
			m.err = fmt.Errorf("no skills found")
			return m, nil
		}
		m.skills = msg.skills
		m.skillsLoaded = true
	}

	return m, nil
}

func (m SplashModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err)
	}

	logoRendered := logoStyle.Render(logo)

	desc := descStyle.Render("Browse and read your installed Claude Code skills.")

	var prompt string
	if m.skillsLoaded {
		prompt = promptStyle.Render(fmt.Sprintf(
			"Found %d skills. Press Enter to continue.",
			len(m.skills),
		))
	} else {
		prompt = promptStyle.Render("Loading skills...")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		logoRendered,
		"",
		desc,
		"",
		prompt,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
