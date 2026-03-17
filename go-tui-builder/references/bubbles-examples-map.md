# Bubbles Examples Map

Use this map to locate concrete component implementations in `references/bubbles-examples/`.

## Directory Notes

Some directories in `bubbles-examples/` contain component library source code (not runnable examples). These are useful for understanding the component's internal API and message types:
- `cursor/` — `cursor.go`: Blinking cursor component source; useful when embedding in custom text inputs.
- `key/` — `key.go`: Key binding system source; shows `Binding`, `Matches`, and `NewBinding` internals.
- `runeutil/` — `runeutil.go`: Unicode/rune helpers used internally by text components.

The remaining directories contain runnable `main.go` examples.

## Intent-to-Component Mapping

### Text Input
- Single-line input field: `textinput/`
- Multi-line text editor: `textarea/`

### Selection and Navigation
- Scrollable, filterable list: `list/`
- File system picker: `filepicker/`
- Page navigation controls: `paginator/`

### Data Display
- Columnar table with selection: `table/`
- Scrollable content pane: `viewport/`

### Feedback and Status
- Animated spinner: `spinner/`
- Progress meter (animated/static): `progress/`

### Time
- Countdown timer: `timer/`
- Count-up stopwatch: `stopwatch/`

### Keybinding and Help
- Key binding definitions and matching: `key/` (library source)
- Contextual help bar rendering: `help/`

### Cursor
- Blinking cursor component: `cursor/` (library source)

## Usage Pattern

1. Find the matching component directory above.
2. If it has a `main.go`, read it to understand the component's Init/Update/View contract and required message types.
3. If it is library source (no `main.go`), read the `.go` file to understand exported types, constructor options, and messages the component emits.
4. Cross-reference with `bubbletea-examples-map.md` to find full-program examples that compose these components.

## Embedding Pattern

Every Bubbles component follows the same delegation pattern:

```go
// In model struct
input textinput.Model

// In Init
func (m model) Init() tea.Cmd {
    return textinput.Blink   // start cursor blink if needed
}

// In Update — always reassign, always collect the command
var cmd tea.Cmd
m.input, cmd = m.input.Update(msg)
return m, cmd

// In View
m.input.View()
```

The key invariant: always reassign the component model and always collect its returned command.
