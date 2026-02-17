# skillex

A TUI for browsing and reading your installed [Claude Code](https://docs.anthropic.com/en/docs/claude-code) skills.

![demo](demo.gif)

## Features

- Discovers all installed skills from Claude Code plugins
- Split-pane layout with filterable skill list and rendered markdown preview
- Vim-style `hjkl` navigation

## Installation

### Homebrew

```
brew install smauermann/tap/skillex
```

### From source

```
go install github.com/smauermann/skillex@latest
```

## Usage

```
skillex
```

### Keybindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate list / scroll preview |
| `l` | Focus preview pane |
| `h` | Back to skill list |
| `/` | Filter skills |
| `q` | Quit |
