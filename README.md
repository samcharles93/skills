# Go Dev Toolkit

`go-dev-toolkit` is a local Codex plugin that bundles the Go-focused skills I use most often into one reusable package.

## Included skills

- `go-development`
- `go-docs`
- `go-error-analyzer`
- `go-modernize`
- `go-test-validator`
- `go-tui-builder`
- `huma-api-builder`
- `go-mcp-creator`

## Purpose

This plugin packages a coherent Go workflow rather than a random skill collection:

- general Go development and review
- documentation and error analysis
- modernization and test quality
- terminal UI work
- HUMA API development
- MCP server creation in Go

## Packaging Notes

- The canonical Codex manifest lives at `.codex-plugin/plugin.json`.
- `plugin.json` at the repo root is kept in sync as a compatibility manifest for existing local workflows.
- `hooks.json` and `.mcp.json` are intentionally valid but empty by default.
- `go-mcp-creator` is synced from `~/skills/go-mcp-creator`.
- `huma-api-builder` now uses its canonical skill name instead of the older `go-huma-api-builder` directory name.

## Maintenance Rules

- Treat `~/skills` as the source of truth for the bundled Go skills unless this plugin intentionally diverges.
- If a bundled skill is updated materially, sync the plugin copy and verify the plugin still has a coherent boundary.
- Avoid adding unrelated non-Go skills to this plugin.
