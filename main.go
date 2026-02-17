package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
	"github.com/smauermann/skillex/internal/discovery"
	"github.com/smauermann/skillex/internal/tui"
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

	// Detect terminal style BEFORE bubbletea takes over stdin.
	// glamour.WithAutoStyle() sends an OSC query with a 5s timeout,
	// which blocks inside a running bubbletea program.
	var styleOpt glamour.TermRendererOption
	if termenv.HasDarkBackground() {
		styleOpt = glamour.WithStylePath("dark")
	} else {
		styleOpt = glamour.WithStylePath("light")
	}

	p := tea.NewProgram(tui.New(skills, styleOpt), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
