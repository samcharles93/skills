# Go Dev Toolkit

`go-dev-toolkit` is a Claude Code plugin bundling the Go skills and tooling I use
most often into one package.

This repository is the single source of truth. Nothing here is synced from a
harness skill directory, and no skill body contains a harness-specific path.
That is what lets the same content serve more than one agent.

## Included skills

| Skill | Covers |
| --- | --- |
| `go-development` | End-to-end development process, quality gate, idioms, error handling, testing, bundled analysis scripts |
| `golangci-lint` | `.golangci.yml` v2 schema, linter settings, incremental adoption |
| `go-mcp-creator` | MCP servers built with the official Go SDK |
| `huma-api-builder` | REST APIs with HUMA v2 |

## Bundled MCP servers

- `go-workspace`
  - location: `mcp-servers/go-workspace`
  - tools: `workspace_summary`, `list_packages`, `describe_package`, `changed_packages`
  - every tool accepts an optional `root` path, so workspace inspection can
    target the caller's repository explicitly

## Installing

### Claude Code

```bash
claude --plugin-dir /path/to/go-agent-skills
```

The manifest is `.claude-plugin/plugin.json`. The MCP server resolves through
`${CLAUDE_PLUGIN_ROOT}`, so it works regardless of the working directory.

### Pi

Pi discovers any directory containing a `SKILL.md` under `~/.agents/skills/`.
Symlink rather than copy, so there is only ever one copy on disk:

```bash
for s in go-development golangci-lint go-mcp-creator huma-api-builder; do
  ln -sfn "$PWD/skills/$s" "$HOME/.agents/skills/$s"
done
```

Pi can also read the directory directly by adding it to the `skills` array in
`~/.pi/agent/settings.json`. Use one mechanism, not both: Pi warns on name
collisions across locations and the first match wins, which makes duplicate
copies silently ambiguous.

## Design rules

These exist because earlier versions of this repo were deployed by copy to four
different harness roots, which drifted to three different contents under two
version strings, one of them edited in place without a version bump.

1. **No harness-specific paths in skill bodies.** A skill may reference its own
   bundled files by relative path, or another skill by name, never
   `~/.claude/...` or `~/.codex/...`. Relative paths resolve from the skill
   directory under every harness that loads skills by directory.
2. **Skills are self-contained.** A skill must not depend on another skill being
   installed. `go-development` bundles its own analysis scripts under
   `scripts/` for this reason.
3. **Deploy by symlink, not copy.** One copy on disk cannot drift.
4. **Keep the plugin coherent.** Go tooling only.

## Repository layout

```
.claude-plugin/plugin.json   Claude Code manifest
.codex-plugin/plugin.json    Codex manifest, retained but unmaintained
.mcp.json                    MCP server registration
hooks.json                   Valid but intentionally empty
skills/<name>/SKILL.md       One directory per skill
mcp-servers/go-workspace/    Go MCP server source
```

## History

`v1.3.0-multiharness` tags the tree as it stood before the Claude plugin
restructure, when the repo targeted Claude, Codex and Copilot simultaneously and
carried five further skills (`go-docs`, `go-error-analyzer`, `go-modernize`,
`go-test-validator`, `go-tui-builder`). The first four were merged into
`go-development/scripts/analysis/`; `go-tui-builder` was dropped as roughly 180
files of vendored upstream Bubble Tea examples.
