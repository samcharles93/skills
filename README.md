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

`go-workspace`, at `mcp-servers/go-workspace`. It has two groups of tools.

**Local workspace.** Every tool accepts an optional `root` path, so inspection
can target the caller's repository explicitly.

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

**pkg.go.dev.** Read-only wrappers over the [pkg.go.dev API](https://pkg.go.dev/v1/api)
(`https://pkg.go.dev/v1`). They need network access and no credentials.

| Tool | Does |
| --- | --- |
| `pkgsite_search` | Search packages, or symbols by name |
| `pkgsite_package` | Package metadata at a version, optionally docs and imports |
| `pkgsite_module` | Module metadata, `go.mod` contents, optionally README |
| `pkgsite_versions` | All versions, with deprecation and retraction status |
| `pkgsite_packages` | Packages in a module |
| `pkgsite_symbols` | Exported symbols of a package |
| `pkgsite_imported_by` | Importers outside the module |
| `pkgsite_vulns` | Known vulnerabilities and fixed versions |

List tools take `limit` (default 20, max 100), `token` for the next page, and
`filter`, a boolean Go expression over the item's fields such as
`kind == "Type"`. Set `PKGSITE_API_URL` to point them at another pkgsite
instance. The API allows 45 requests per second per IP block.

`run_tests_for_package` is the one to be deliberate about. Running a project's
tests runs that project's code, so approve it only for repositories you trust.
Note that `.claude/settings.local.json` in this repo pre-approves it and sets
`enableAllProjectMcpServers`, which suppresses the prompt while working here.
That file is untracked and does not ship.

Prebuilt binaries ship in `mcp-servers/go-workspace/bin/` for linux and darwin
(amd64, arm64) and windows (amd64). `run.sh` picks the one for the host and falls
back to `go run` only where none exists, which needs a Go toolchain and network
on first start. On Windows it runs through Git Bash's `sh`.

**After changing the server, run `mcp-servers/go-workspace/build.sh` and commit
the binaries with the source.** The plugin runs whatever is in `bin/`, so stale
binaries silently serve old behaviour.

## Installation

The plugin updates itself and includes the MCP server. Copying the skill
directories gives you editable files that you update by hand, without the server.
Pick one per agent: installing both gives you every skill twice.

<details open>
<summary><strong>Claude Code</strong> (plugin, includes the MCP server)</summary>

In Claude Code:

```
/plugin install go-dev-toolkit --marketplace samcharles93/skills
```

Or from a shell:

```bash
claude plugin marketplace add samcharles93/skills
claude plugin install go-dev-toolkit@samcharles93
```

Update with `claude plugin marketplace update samcharles93`. The `go-workspace`
server starts from a prebuilt binary, so it needs no Go toolchain.

</details>

<details>
<summary><strong>Any other agent, or editable files</strong> (skills only, manual updates)</summary>

```bash
npx skills@latest add samcharles93/skills -a <agent>  # cursor, codex, opencode, pi, ...; omit -a to choose
```

Update with `npx skills@latest update`. To use the MCP server with another
agent, register `sh <clone>/mcp-servers/go-workspace/run.sh` as a stdio server.

</details>

<details>
<summary><strong>From a local clone</strong> (development)</summary>

```bash
git clone https://github.com/samcharles93/skills
claude --plugin-dir ./skills
```

For Pi, symlink rather than copy so there is only one copy on disk:

```bash
for s in go-development golangci-lint go-ldap go-mcp-creator huma-api-builder; do
  ln -sfn "$PWD/skills/skills/$s" "$HOME/.agents/skills/$s"
done
```

Pi can instead read the directory from the `skills` array in
`~/.pi/agent/settings.json`. Use one mechanism: Pi warns on name collisions and
the first match wins.

</details>

### Requirements

- The skills' quality gate expects `gofumpt`, `golangci-lint`, `staticcheck` and
  `govulncheck` on `PATH`.
- The `pkgsite_*` tools need network access to `pkg.go.dev`; no credentials.
- On Windows the MCP server runs through Git Bash's `sh`.

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
