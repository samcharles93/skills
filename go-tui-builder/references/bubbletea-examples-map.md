# Bubble Tea Examples Map

Use upstream examples as the default source of truth for implementation patterns:

- `https://github.com/charmbracelet/bubbletea/tree/main/examples`

## How to Use

1. Map the request to one or more example folders below.
2. Read the selected `main.go` files first.
3. Port architecture and message/command flow, then adapt naming/state to project conventions.
4. Avoid copy-pasting styles and constants that are app-specific.

## Intent-to-Example Mapping

- Minimal app shell: `simple`
- Multi-view navigation: `views`, `composable-views`, `tabs`
- External event injection: `send-msg`, `realtime`
- Ordered async orchestration: `sequence`
- Alt screen and fullscreen: `altscreen-toggle`, `fullscreen`
- Resize-aware rendering: `window-size`, `table-resize`
- Form/input UX: `textinput`, `textinputs`, `textarea`, `credit-card-form`, `split-editors`
- Lists and tables: `list-default`, `list-fancy`, `list-simple`, `table`, `table-resize`, `paginator`
- Loading and progress: `spinner`, `spinners`, `progress-static`, `progress-animated`, `progress-download`
- Network and process integration: `http`, `exec`, `pipe`, `tui-daemon-combo`, `package-manager`
- File browsing: `file-picker`
- Mouse handling: `mouse`
- Quit protections: `prevent-quit`
- Input throttling: `debounce`
- Markdown and rich text rendering: `glamour`, `pager`
- Time-driven state: `timer`, `stopwatch`
- Help overlays and key hints: `help`
- Completion UX: `autocomplete`
- End-to-end chat-like flow: `chat`
- Terminal window title: `set-window-title`
- Focus/blur terminal events: `focus-blur`
- Process suspend and resume: `suspend`
- Extracting result value after exit: `result`
- Animated cell grid / canvas rendering: `cellbuffer`
- Animated character rendering / Harmonica springs: `eyes`, `cellbuffer`

## Adaptation Rules

- Preserve the example’s update-loop structure before introducing refactors.
- Convert hard-coded constants to configuration only after behavior matches.
- Keep example-specific message types, then rename them to domain language.
- If multiple examples apply, compose them in this order:
  - architecture (`views`/`composable-views`)
  - interaction (`textinput`/`table`/`mouse`)
  - async/control flow (`send-msg`/`sequence`/`realtime`)
  - polish (`help`/`glamour`/`spinner`)
- Confirm APIs against local dependency versions before finalizing.
