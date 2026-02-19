# Activation Indicator UX Redesign

## Problem

The current colored dot (`●`) next to each skill in the list panel is not self-explanatory. Users have no way to know what the dot means without external documentation.

## Design

### List panel: colored tag label

Replace the bare `●` dot with a short colored word after the plugin name:

- **`directive`** (green) — description uses imperative language (MUST, ALWAYS, NEVER, DO NOT)
- **`passive`** (orange) — description uses soft language (Use when, Helps, Can be used, etc.)
- **`unknown`** (dim) — no recognized activation signals in description

Example:

```
> brainstorming
  superpowers  directive

  frontend-design
  frontend-design  passive

  some-skill
  local  unknown
```

### Detail panel: activation banner + frontmatter + body

Three sections prepended above the SKILL.md body content:

**1. Activation banner** — a single colored line:

- Directive (green): "Strong wording — Claude will almost always pick up this skill automatically."
- Passive (orange): "Weak wording — Claude may skip this skill. Use MUST/ALWAYS/NEVER in the description to improve activation."
- Neutral (dim): "No activation signals found in description. Add directive language like MUST, ALWAYS, or NEVER to ensure Claude invokes this skill."

**2. Frontmatter fields** — every field from the YAML frontmatter rendered as bold key/value pairs:

```
**name:** brainstorming
**description:** You MUST use this before any creative work...
```

All fields present in the frontmatter are shown, not just name and description.

**3. Horizontal rule, then body** — the rest of the SKILL.md content as before.

## Changes

### `discovery.go`

- Add `Frontmatter string` field to the `Skill` struct to store the raw YAML block
- `parseFrontmatter` returns the raw YAML string alongside the parsed struct
- `discoverSkillsInDir` stores it on each skill

### `discovery_test.go`

- Update tests to assert on the new `Frontmatter` field

### `tui.go`

- Rename `activationDot` to `activationTag`, render a colored word instead of a dot
- Update `skillDelegate.Render` to use the tag
- Update `updateViewportContent` to prepend activation banner, styled frontmatter fields, and horizontal rule before body content
