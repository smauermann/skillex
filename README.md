# skillex

A TUI for browsing and reading your installed [Claude Code](https://docs.anthropic.com/en/docs/claude-code) skills.

![demo](demo.gif)

## Features

- Discovers all installed skills from Claude Code plugins as well as local skills
- Split-pane layout with filterable skill list and rendered markdown preview
- Vim-style `hjkl` navigation
- **Activation health indicators** — colored dot per skill shows whether its description is likely to auto-invoke
- **Description budget meter** — tracks total description length against the 16,000-character limit before skills silently stop loading

## Skill health indicators

### Activation style dot

Each skill in the list shows a `●` dot next to the plugin name:

| Dot | Meaning |
|-----|---------|
| ● (green) | **Directive** — description uses imperative language (`ALWAYS`, `MUST`, `NEVER`, `DO NOT`). Community benchmarks show ~98% auto-activation rate. |
| ● (dim) | **Neutral** — no strong signal either way. |
| ● (orange) | **Passive** — description uses descriptive language (`Use when`, `Helps`, `Can be used`, `Useful for`). Community benchmarks show ~69% auto-activation rate. |

Auto-invocation in Claude Code is unreliable by default. Testing by [Scott Spence across 200+ prompts](https://scottspence.com/posts/claude-code-skills-dont-auto-activate) found a ~50% baseline activation rate — essentially a coin flip. [Ivan Seleznov's 650-trial study](https://medium.com/@ivan.seleznov1/why-claude-code-skills-dont-activate-and-how-to-fix-it-86f679409af1) confirmed that **description wording is the primary lever**: passive descriptions scored as low as 69% while directive descriptions using `ALWAYS invoke...` / `DO NOT ... directly` reached 98–100%.

**Example rewrites:**

| Before (passive, orange dot) | After (directive, green dot) |
|------------------------------|------------------------------|
| `Helps you write commit messages.` | `ALWAYS invoke this skill when writing a git commit message.` |
| `Use when reviewing pull requests.` | `NEVER review a pull request without invoking this skill first.` |

### Description budget meter

The bottom bar shows `desc budget: X/16k` tracking the total length of all skill descriptions combined:

| Color | Meaning |
|-------|---------|
| Green | Under 12,800 chars (80%) — plenty of room |
| Orange | 12,800–15,999 chars — approaching the limit |
| Red | 16,000+ chars — over the limit |

Claude Code loads all skill descriptions into its system prompt at startup under an `available_skills` section. The budget for that section is **16,000 characters** (or 2% of the model's context window, whichever is larger). When the combined total of all skill descriptions exceeds the budget, skills are silently excluded — no error, no warning, they just stop appearing to Claude. The limit was first documented empirically in [GitHub issue #13099](https://github.com/anthropics/claude-code/issues/13099), where researchers found 42 of 63 installed skills invisible once the total crossed ~15,500 chars. It is now [officially documented](https://code.claude.com/docs/en/skills) in the Claude Code troubleshooting guide and can be raised by setting the `SLASH_COMMAND_TOOL_CHAR_BUDGET` environment variable.

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
