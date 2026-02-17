package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
	"github.com/smauermann/skillex/internal/tui"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pluginsFile := filepath.Join(homeDir, ".claude", "plugins", "installed_plugins.json")

	// Detect terminal style BEFORE bubbletea takes over stdin.
	var styleOpt glamour.TermRendererOption
	if termenv.HasDarkBackground() {
		styleOpt = glamour.WithStylePath("dark")
	} else {
		styleOpt = glamour.WithStylePath("light")
	}

	p := tea.NewProgram(tui.NewSplash(pluginsFile, styleOpt), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
