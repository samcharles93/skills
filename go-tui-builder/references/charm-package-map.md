# Charmbracelet Package Map

Use this map to choose the smallest package set that solves the requested UI behavior.

## Core Runtime

- `github.com/charmbracelet/bubbletea`: Program loop, model lifecycle, messages, commands, renderer integration.
- `github.com/charmbracelet/bubbles`: Reusable UI components and key/help primitives.
- `github.com/charmbracelet/lipgloss`: Styling, borders, spacing, alignment, adaptive color/layout formatting.

## Common Bubbles Components

- `bubbles/textinput`: Single-line editable inputs with unicode, pasting, and in-place scrolling.
- `bubbles/textarea`: Multi-line editors with unicode, pasting, and vertical scrolling.
- `bubbles/viewport`: Scrollable wrapped content with optional pager keybindings and mouse support.
- `bubbles/list`: Filterable/selectable list UI with pagination, fuzzy filter, spinner, and status messages.
- `bubbles/table`: Columnar data rendering with selection and vertical scrolling.
- `bubbles/progress`: Progress bars with optional Harmonica animation, solid/gradient fills.
- `bubbles/spinner`: Loading indicators; multiple built-in frame sets.
- `bubbles/paginator`: Paging controls (dot or numeric style); can be used for logic only.
- `bubbles/help`: Contextual key help rendering; single/multi-line modes with graceful truncation.
- `bubbles/key`: Key binding definitions and matching; powers help generation and remapping.
- `bubbles/cursor`: Blinking cursor component for use inside text inputs.
- `bubbles/timer` and `bubbles/stopwatch`: Time-driven displays with configurable update frequency.
- `bubbles/filepicker`: Filesystem navigation and selection with extension filtering.

## Interactive Prompts (promptkit)

- `github.com/erikgeiser/promptkit/selection`: Selection prompt with filtering and pagination over any slice type.
- `github.com/erikgeiser/promptkit/textinput`: Single-line text prompt with validation and hidden/password mode.
- `github.com/erikgeiser/promptkit/confirmation`: Binary yes/no confirmation prompt.

Use promptkit when:
- You need self-contained, styled prompts that run and return a value before the main TUI starts.
- You want to embed interactive prompts as Bubble Tea sub-models inside a larger application.
- You need password/secret input or custom per-item validation.

Note: promptkit is pre-v1.0; expect breaking changes in minor versions. Windows support is limited.

```go
// Standalone usage — runs prompt and returns chosen value
sp := selection.New("Pick one:", []string{"alpha", "beta", "gamma"})
sp.PageSize = 5
choice, err := sp.RunPrompt()

// Embedded in a Bubble Tea model
spModel := selection.NewModel(sp)
// spModel implements tea.Model; embed and delegate Update/View normally
```

## Styling and Rendering Helpers

### lipgloss
The primary styling layer. Use for all borders, padding, margins, alignment, and adaptive colors.
```go
style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FF75B7")).
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2)
```

### glamour
- `github.com/charmbracelet/glamour`: Terminal markdown rendering with configurable themes.
Use only when rendering markdown content; overkill for plain text.

### harmonica
- `github.com/charmbracelet/harmonica`: Spring physics animations for smooth state transitions.
Use only when animation materially improves UX (e.g., scrolling momentum, bounce effects).

### termenv
- `github.com/muesli/termenv`: Low-level terminal color/profile detection and ANSI styling.

Key capabilities:
- Detect color profile: `Ascii` → `ANSI` (16 colors) → `ANSI256` → `TrueColor`
- Query terminal background (light vs dark): `output.HasDarkBackground()`
- Automatic color downgrade across profiles
- Cursor control, screen management, mouse mode, bracketed paste, window title

```go
output := termenv.NewOutput(os.Stdout)
profile := output.ColorProfile()        // detect capability
dark := output.HasDarkBackground()

s := output.String("Hello").
    Foreground(output.Color("#abcdef")).
    Bold()
fmt.Println(s)
```

Lipgloss uses termenv internally. Use termenv directly only when you need lower-level control,
custom color conversion, or terminal state management outside Lipgloss's scope.

### reflow
- `github.com/muesli/reflow`: ANSI-aware text transformation; all operations preserve escape sequences.

Sub-packages:
- `reflow/wordwrap`: Break at word boundaries within a column limit.
- `reflow/wrap`: Hard wrap at exact character limit (may break mid-word).
- `reflow/indent`: Add consistent leading indentation.
- `reflow/dedent`: Strip common leading whitespace.
- `reflow/padding`: Right-pad strings to a fixed width.
- `reflow/truncate`: Truncate to a max width, preserving ANSI codes.
- `reflow/margin`: Add configurable horizontal margins.

```go
// Word-wrap a styled string to 60 chars
wrapped := wordwrap.String(styledText, 60)

// Truncate with ANSI safety
truncated := truncate.StringWithTail(line, uint(maxWidth), "…")
```

Reflow pairs naturally with `bubbles/viewport` to pre-process content before setting it as viewport content.

## Companion Charm Packages (Version-Dependent)

- `github.com/charmbracelet/log`: Structured logging aligned with terminal workflows.
- `github.com/charmbracelet/x/ansi`: ANSI utilities for low-level terminal rendering tasks.
- `github.com/charmbracelet/x/term`: Terminal capability and state helpers.

Only use `charmbracelet/x/*` modules when the repository already depends on them or when the task needs lower-level control not covered by Bubble Tea/Bubbles.

## Selection Rules

- Start with Bubble Tea + Lip Gloss; add Bubbles components incrementally.
- Prefer existing components over custom controls.
- Use promptkit for isolated prompts or embedded sub-model prompts.
- Reach for Glamour only when rendering markdown.
- Use reflow for ANSI-safe text wrapping/truncation when Lipgloss alone is insufficient.
- Use termenv directly only for color profile detection or terminal state operations.
- Use Harmonica only when animation materially improves UX.
- Add low-level ANSI/term packages only for renderer edge cases.

## Version Guardrails

- Inspect actual versions before using APIs:
```bash
go list -m all | rg 'charmbracelet|muesli|erikgeiser'
```
- Match code to the repository's existing package versions.
- If an API is uncertain, verify with:
```bash
go doc <module-or-package>
```
