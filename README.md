# Go Dev Toolkit

`go-dev-toolkit` is a Claude Code plugin bundling the Go skills and tooling I use
most often into one package.

This repository is the single source of truth. Nothing here is synced from a
harness skill directory, and no skill body contains a harness-specific path.
That is what lets the same content serve more than one agent.

## Included skills

| Skill | Covers |
| --- | --- |
| `go-development` | Development process, quality gate, naming, error handling, interface and concurrency idioms, bundled analysis scripts |
| `golangci-lint` | `.golangci.yml` v2 schema, linter settings, incremental adoption |
| `go-ldap` | LDAP and Active Directory from Go: bind, search, authentication, passwords, groups |
| `go-mcp-creator` | MCP servers built with the official Go SDK |
| `huma-api-builder` | REST APIs with HUMA v2 |

Skills own their triggers without overlapping. `go-development` covers fixing
lint findings; `golangci-lint` covers authoring the config that produces them.

## Bundled MCP servers

`go-workspace`, at `mcp-servers/go-workspace`. Every tool accepts an optional
`root` path, so inspection can target the caller's repository explicitly.

| Tool | Does |
| --- | --- |
| `workspace_summary` | Overview of the module and its layout |
| `list_packages` | Enumerate packages |
| `describe_package` | Detail on one package |
| `changed_packages` | Packages touched relative to git state |
| `search_symbols` | Find symbols across the workspace |
| `analyze_interface_impl` | Which types implement an interface |
| `find_dependents` | Reverse dependency lookup |
| `find_deprecated_usage` | Uses of deprecated identifiers |
| `module_dependency_graph` | Module dependency graph |
| `run_tests_for_package` | **Runs `go test`, which executes code from the target repository** |

`run_tests_for_package` is the one to be deliberate about. Running a project's
tests runs that project's code, so approve it only for repositories you trust.
Note that `.claude/settings.local.json` in this repo pre-approves it and sets
`enableAllProjectMcpServers`, which suppresses the prompt while working here.
That file is untracked and does not ship.

The server is started with `go run`, so it compiles on each start and needs a Go
toolchain plus a warm module cache or network on first use. That is the price of
staying platform independent rather than shipping a prebuilt binary.

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
for s in go-development golangci-lint go-ldap go-mcp-creator huma-api-builder; do
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
