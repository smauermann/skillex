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

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pluginsFile := filepath.Join(homeDir, ".claude", "plugins", "installed_plugins.json")

	// Collect local skill directories that exist: home-level and project-level
	var localDirs []discovery.LocalSkillsDir
	if dir := filepath.Join(homeDir, ".claude", "skills"); isDir(dir) {
		localDirs = append(localDirs, discovery.LocalSkillsDir{Path: dir, Name: "local"})
	}
	if wd, err := os.Getwd(); err == nil && wd != homeDir {
		if dir := filepath.Join(wd, ".claude", "skills"); isDir(dir) {
			localDirs = append(localDirs, discovery.LocalSkillsDir{Path: dir, Name: filepath.Base(wd)})
		}
	}

	// Detect terminal style BEFORE bubbletea takes over stdin.
	var styleOpt glamour.TermRendererOption
	if termenv.HasDarkBackground() {
		styleOpt = glamour.WithStylePath("dark")
	} else {
		styleOpt = glamour.WithStylePath("light")
	}

	p := tea.NewProgram(tui.NewSplash(pluginsFile, localDirs, styleOpt), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
