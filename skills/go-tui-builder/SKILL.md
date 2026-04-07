---
name: go-tui-builder
description: Expert guidance for building, debugging, and refactoring Go terminal UIs with Bubble Tea, Bubbles, Lip Gloss, promptkit, termenv, reflow, Glamour, Harmonica, and related Charmbracelet/muesli packages. Use when implementing new TUIs, structuring model/init/update/view loops, handling async commands and messages, composing reusable components, building interactive prompts (selection, text, confirmation), styling responsive layouts with ANSI-safe text handling, wiring keyboard/mouse/focus interactions, tuning performance, or adding reliable tests for Bubble Tea applications.
---

# Go TUI Builder

Build production-grade Go TUIs with Bubble Tea and the Charmbracelet ecosystem while preserving clear state flow, predictable updates, and maintainable component boundaries.

## Workflow

1. Inspect `go.mod`, current model hierarchy, and terminal assumptions (alt-screen, mouse mode, color profile, non-TTY fallback).
2. Decide the model structure:
   - Use a single model for simple single-screen tools.
   - Use a root model with child models for multi-screen apps.
   - Wrap reusable behavior in dedicated components.
   - Use promptkit for self-contained prompts that run before or alongside the main TUI.
3. Define message and command contracts before coding:
   - Represent domain events as explicit message types.
   - Keep command producers isolated from rendering logic.
   - Represent error and cancellation paths as messages.
4. Implement in this order:
   - Define state and key maps.
   - Implement `Init()` startup commands.
   - Implement `Update(msg tea.Msg)` for transitions and command emission.
   - Implement `View()` as a pure render function.
5. Validate behavior:
   - Add deterministic tests for update transitions.
   - Add snapshot/golden checks for critical render fragments.
   - Run race checks when introducing concurrent work.
6. Finalize UX:
   - Show focus, loading, empty, and error states.
   - Handle terminal resize from `tea.WindowSizeMsg`.
   - Handle focus/blur from `tea.FocusMsg`/`tea.BlurMsg` when using `tea.WithReportFocus()`.
   - Use reflow for ANSI-safe word-wrap and truncation before loading content into viewports.
   - Restore terminal state cleanly on quit.

## Non-Negotiable Rules

- Keep `View()` side-effect free.
- Keep `Update()` focused on state transitions; run I/O inside commands.
- Prefer typed messages over generic payloads.
- Track component focus explicitly.
- Recompute layout from current size and state, not fixed constants.
- Always reassign Bubbles component models and collect their returned commands.
- Preserve existing repository conventions unless they are broken.

## References

Load `references/charm-package-map.md` when selecting packages and primitives across Bubble Tea, Bubbles, Lip Gloss, promptkit, termenv, reflow, and companion libraries.

Load `references/bubbletea-recipes.md` for concrete implementation patterns: command/message architecture, child-model delegation, async work, viewport/list/table integration, focus/blur, suspend/resume, result extraction, promptkit integration, reflow text handling, and test loops.

Load `references/bubbletea-examples-map.md` to map user intent to official `bubbletea/examples/*` implementations before writing new code.

Load `references/bubbles-examples-map.md` to map user intent to Bubbles component examples and understand component embedding patterns.

Use the `references/bubbletea-examples` directory when you need concrete runnable implementations of patterns.
Use the `references/bubbles-examples` directory when you need component-level source and examples from the bubbles repo.

## Implementation Checklist

- Define `bubbles/key` maps and help output (`bubbles/help`) for each interaction context.
- Handle `tea.KeyMsg`, `tea.WindowSizeMsg`, and domain messages explicitly.
- Use `tea.Batch` for independent commands and `tea.Sequence` for dependent commands.
- Keep long-running work out of `Update()` and return completion messages.
- Make Lip Gloss layout width-aware and resize-safe.
- Use reflow (`wordwrap`, `truncate`) for ANSI-safe text handling before viewport ingestion.
- Provide narrow-terminal fallback rendering.
- Treat errors as visible UI states; do not silently drop them.
- Use promptkit for selection/text/confirmation prompts rather than building from scratch.

## Quality Gate

Run existing project checks first. If none exist, run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

For render regressions, prefer stable snapshot assertions for targeted sections instead of asserting full-frame output.
