# Skill Analytics Panel

## Problem

Skill insights (activation strength, description budget) are scattered across three UI locations:

1. Colored activation tags in the list panel
2. Activation banner text inside the detail viewport
3. Budget meter in the help bar

This causes three issues: the banner crowds the markdown preview, the budget meter is easy to miss, and the information feels fragmented.

## Design

Split the right column vertically into two bordered panels:

- **Skill Analytics** (~20% height) — consolidated skill health metrics
- **SKILL.md** (~80% height) — markdown preview (unchanged, minus the banner)

### Layout

```
╭─ Skills ───────────╮╭─ Skill Analytics ─────────────────╮
│                     ││ Activation   directive  — Claude  │
│ > brainstorming     ││              will auto-activate   │
│   superpowers       ││ Budget       842 / 16.0k (5.3%)  │
│   directive         ││              ████████░░░░░ 57%    │
│                     │╰───────────────────────────────────╯
│   dispatching-agts  │╭─ SKILL.md ────────────────────────╮
│   superpowers       ││                                   │
│   passive           ││ **name:** brainstorming           │
│                     ││ **description:** You MUST use...  │
│   executing-plans   ││ ---                               │
│   superpowers       ││                                   │
│   passive           ││ # Brainstorming Ideas Into...     │
│                     ││                                   │
╰─────────────────────╯╰───────────────────────────────────╯
 j/k navigate  l read preview  / filter  q quit
```

### Analytics Panel Content

Two key-value rows:

**Row 1 — Activation:**
- Label: `Activation`
- Value: colored tag (`directive`/`passive`/`unknown`) + condensed advice text
- Colors: green (#35) for directive, orange (#214) for passive, dim (#242) for neutral

**Row 2 — Budget:**
- Label: `Budget`
- Value: `<this skill chars> / <total limit> (<percentage>)`
- Sub-row: visual progress bar using `█`/`░` + percentage label
- Colors: green when under 80%, orange 80-100%, red over 100%

### What Moves

| Element | From | To |
|---------|------|----|
| Activation strength + advice | Banner inside viewport | Analytics panel row 1 |
| Per-skill budget contribution | Not shown | Analytics panel row 2 (new) |
| Global budget meter | Help bar (right side) | Analytics panel row 2 |

### What Stays

- Activation tags in the list panel (useful for scanning)
- Frontmatter rendering at the top of SKILL.md
- Help bar (keybindings only, budget removed)

### Sizing

- Analytics panel: fixed ~4-5 inner rows
- SKILL.md panel: remaining height (still scrollable via viewport)
- Both panels share the same `renderPanel()` function and border styling
- Both panels share the right column width (`totalWidth - listWidth`)

### Placeholder

When no skill is selected, the analytics panel shows a neutral placeholder message.
