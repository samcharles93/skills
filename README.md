# Go Dev Toolkit

`go-dev-toolkit` is a local Codex plugin that bundles the Go-focused skills I use most often into one reusable package.

## Bundled MCP servers

- `go-workspace`
  - location: `mcp-servers/go-workspace`
  - tools: `workspace_summary`, `list_packages`, `describe_package`, `changed_packages`
  - note: every tool accepts an optional `root` path so workspace inspection can target the caller's repository explicitly

## Included skills

- `go-development`
- `golangci-lint`
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
- code quality gates and linter configuration
- documentation and error analysis
- modernization and test quality
- terminal UI work
- HUMA API development
- MCP server creation in Go
- Go workspace inspection and package discovery

## Packaging Notes

- The canonical Codex manifest lives at `.codex-plugin/plugin.json`.
- `plugin.json` at the repo root is kept in sync as a compatibility manifest for existing local workflows.
- `hooks.json` is intentionally valid but empty by default.
- `.mcp.json` now bundles the local `go-workspace` server via `go -C ./mcp-servers/go-workspace run .`.
- `go-mcp-creator` is synced from `~/skills/go-mcp-creator`.
- `golangci-lint` is synced from `~/.agents/skills/golangci-lint`, which is its
  source of truth rather than `~/skills`.
- `huma-api-builder` now uses its canonical skill name instead of the older `go-huma-api-builder` directory name.

## Maintenance Rules

- Treat `~/skills` as the source of truth for the bundled Go skills unless this plugin intentionally diverges.
- If a bundled skill is updated materially, sync the plugin copy and verify the plugin still has a coherent boundary.
- Avoid adding unrelated non-Go skills to this plugin.
