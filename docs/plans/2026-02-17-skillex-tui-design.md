# Skillex TUI Design

## Overview

A Go TUI for browsing Claude Code skills. Split-pane layout with a filterable skill list on the left and rendered markdown preview on the right.

## Skill Discovery

- Parse `~/.claude/plugins/installed_plugins.json` to find installed plugins and their install paths
- Walk each plugin's `skills/*/SKILL.md` to find all skills
- Extract YAML frontmatter (`name`, `description`) from each `SKILL.md`
- Deduplicate by using the plugin entries from `installed_plugins.json` (canonical source)

## Layout

```
+----------------------+--------------------------------+
|  Skills              |  SKILL.md (rendered)           |
|                      |                                |
| > brainstorming      |  # Brainstorming               |
|   dispatching-agents |                                |
|   executing-plans    |  Help turn ideas into fully    |
|   ...                |  formed designs and specs...   |
+----------------------+--------------------------------+
```

- Left pane: `bubbles/list` with built-in filtering
- Right pane: `bubbles/viewport` with glamour-rendered markdown
- Layout managed by `lipgloss`, responsive to terminal width

## Keybindings

- `j/k` or arrows: navigate skill list
- `enter` or `l`: select skill, show preview
- `/`: filter skills
- `q` or `ctrl+c`: quit

## Libraries

- `charmbracelet/bubbletea` — TUI framework (Elm architecture)
- `charmbracelet/bubbles` — list and viewport components
- `charmbracelet/lipgloss` — layout and styling
- `charmbracelet/glamour` — markdown rendering
- `gopkg.in/yaml.v3` — YAML frontmatter parsing

## Data Flow

1. Startup: read `~/.claude/plugins/installed_plugins.json`
2. For each plugin, resolve `installPath/skills/*/SKILL.md`
3. Parse YAML frontmatter from each SKILL.md
4. Populate list with skill name, description, plugin source
5. On cursor move, render selected skill's full SKILL.md in viewport
