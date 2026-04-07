# Bubble Tea Recipes

Use these patterns to keep state transitions predictable and component boundaries clean.

## Message and Command Pattern

```go
type loadDoneMsg struct {
	items []Item
	err   error
}

func loadItemsCmd(repo Repo) tea.Cmd {
	return func() tea.Msg {
		items, err := repo.Load()
		return loadDoneMsg{items: items, err: err}
	}
}
```

Use explicit message types per async workflow. Do not return raw errors without context.

## Root Model Skeleton

```go
type model struct {
	keys     keyMap
	loading  bool
	err      error
	items    []Item
	spinner  spinner.Model
	viewport viewport.Model
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadItemsCmd(repo),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = max(0, msg.Height-4)
	case loadDoneMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.items = msg.items
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}
```

## Child Model Delegation

Delegate by active route/screen and only let one child handle focused input.

```go
if m.route == routeList {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}
```

## Async Sequencing

- Use `tea.Batch(cmdA, cmdB)` when commands can run independently.
- Use `tea.Sequence(cmdA, cmdB)` when command B depends on command A output.
- Emit completion/error messages for every async branch.

## Resize and Layout

- Handle every `tea.WindowSizeMsg`.
- Recompute child widths/heights from current dimensions.
- Keep one source of truth for frame padding/margins to avoid drift.
- Clamp width/height to safe minimums before rendering.

## Focus and Keymaps

- Keep per-mode keymaps (`browse`, `edit`, `confirm`).
- Update help output when focus/mode changes.
- Keep `tab`/`shift+tab` behavior consistent across components.

## Focus/Blur Terminal Events

Use `tea.WithReportFocus()` to receive `tea.FocusMsg` and `tea.BlurMsg` when the terminal
window gains or loses focus. Useful for pausing animations or dimming the UI when unfocused.

```go
p := tea.NewProgram(model{focused: true}, tea.WithReportFocus())

// In Update:
case tea.FocusMsg:
    m.focused = true
case tea.BlurMsg:
    m.focused = false
```

See `references/bubbletea-examples/focus-blur/` for the complete pattern.

## Suspend and Resume

Use `tea.Suspend` to temporarily yield terminal control (e.g., to launch an editor).
The program receives `tea.ResumeMsg` when it regains the terminal.

```go
case tea.KeyMsg:
    if msg.String() == "ctrl+z" {
        return m, tea.Suspend
    }
case tea.ResumeMsg:
    m.suspending = false
    return m, nil
```

See `references/bubbletea-examples/suspend/` for the complete pattern.

## Extracting a Result After Exit

Run the program normally but type-assert the returned model to extract a value:

```go
finalModel, err := tea.NewProgram(initialModel).Run()
if err != nil {
    log.Fatal(err)
}
if m, ok := finalModel.(model); ok {
    fmt.Println("You chose:", m.choice)
}
```

See `references/bubbletea-examples/result/` for the complete pattern.

## Setting the Window Title

```go
func (m model) Init() tea.Cmd {
    return tea.SetWindowTitle("My App")
}
```

See `references/bubbletea-examples/set-window-title/` for the complete pattern.

## Error Handling

- Represent expected failures as dedicated messages.
- Store user-visible error state in the model.
- Render non-blocking errors inline unless task is unrecoverable.
- Reserve panic/fatal exits for truly impossible states.

## Reflow: ANSI-Safe Text Wrapping

Use `reflow` to pre-process content before loading it into a viewport or rendering it inline.
All reflow operations preserve ANSI escape sequences (colors, styles).

```go
import (
    "github.com/muesli/reflow/wordwrap"
    "github.com/muesli/reflow/truncate"
)

// Word-wrap styled content to fit a viewport
wrapped := wordwrap.String(styledContent, viewportWidth-2)
m.viewport.SetContent(wrapped)

// Truncate a line with an ellipsis, preserving ANSI
line := truncate.StringWithTail(rawLine, uint(maxWidth), "…")
```

## promptkit Integration

Use `promptkit` for interactive prompts that precede or are embedded in a Bubble Tea program.

**Standalone prompt (before the main TUI starts):**
```go
import "github.com/erikgeiser/promptkit/selection"

sp := selection.New("Choose environment:", []string{"dev", "staging", "prod"})
env, err := sp.RunPrompt()
if err != nil {
    log.Fatal(err)
}
// now start the main TUI with env
```

**Embedded as a sub-model (inside the main program):**
```go
import (
    "github.com/erikgeiser/promptkit/selection"
    "github.com/erikgeiser/promptkit/textinput"
    "github.com/erikgeiser/promptkit/confirmation"
)

// In model struct
type model struct {
    phase     phase
    picker    *selection.Model[string]
    nameInput *textinput.Model
    confirm   *confirmation.Model
}

// Delegate to the active sub-model
case phaseSelect:
    newPicker, cmd := m.picker.Update(msg)
    m.picker = newPicker.(*selection.Model[string])
    // Check completion
    if val, err := m.picker.Value(); err == nil {
        m.selected = val
        m.phase = phaseConfirm
    }
    return m, cmd
```

Notes:
- promptkit is pre-v1.0; verify API against installed version with `go doc`.
- Each promptkit model returns its value via a `.Value()` method after the user confirms.
- Text input supports `.Hidden(true)` for password/secret fields and `.Validate(fn)` for custom rules.

## Testing Loop

- Unit-test `Update` transitions with table-driven tests.
- Assert key behaviors: quit, focus switch, async completion, error path.
- Snapshot stable `View` fragments where formatting is critical.
- Run `go test -race ./...` when commands perform concurrent work.

## Debugging Checklist

- Confirm `Update` always returns the model in all switch branches.
- Confirm component `Update` return values are reassigned.
- Confirm long-running work is in commands, not inline in `Update`.
- Confirm `tea.WindowSizeMsg` reaches each size-aware component.
- Confirm quit path returns `tea.Quit` and cleans up external resources.
- Confirm `tea.WithReportFocus()` is set if the model handles `tea.FocusMsg`/`tea.BlurMsg`.
