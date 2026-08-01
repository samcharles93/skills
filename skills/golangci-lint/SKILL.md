---
name: golangci-lint
description: Configure and manage golangci-lint v2 (version "2") configuration files (.golangci.yml, .golangci.yaml). Use when configuring Go linting, setting up linter settings, installing golangci-lint, avoiding go install pitfalls, migrating from v1 to v2 schema, setting up incremental lint adoption (new-from-merge-base), configuring formatters (gofumpt, gci), or setting up gocritic ruleguard AST rules.
---

# golangci-lint v2 Configuration & Management

This skill provides procedural guidance, schema enforcement, installation best practices, and verification workflows for configuring `golangci-lint` v2 (`version: "2"`).

---

## Core Rules & Non-Negotiable Requirements

### 1. Binary Installation Only (No `go install`)
**Never install `golangci-lint` via `go install`.** Official maintainers warn against `go install` because it compiles against local Go versions, ignores `replace` directives in `go.mod`, causes transitive dependency pollution, and is significantly slower.

- **Local**: Use `curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2` or package managers (`brew install golangci-lint`).
- **CI**: Use official GitHub Action `golangci/golangci-lint-action@v6` with pinned exact versions (`version: v2.12.2`).

*For full installation instructions across OSs and CI systems, see [references/installation.md](references/installation.md).*

---

### 2. v2 Schema Only
Every configuration file MUST specify `version: "2"` at the top level. **Emitting v1 syntax is a failure.**

| v1 Syntax (INVALID) | v2 Syntax (REQUIRED) |
| :--- | :--- |
| `linters-settings:` | `linters.settings:` |
| `issues.exclude-rules:` | `linters.exclusions.rules:` |
| `run.skip-dirs:` / `skip-files:` | `linters.exclusions.paths:` |
| `gofmt`, `gofumpt`, `gci` in `linters.enable` | `formatters.enable:` and `formatters.settings:` |
| `linters.enable-all: true` | `linters.default: all \| standard \| fast \| none` |
| `output.print-issued-lines:` | `output.formats.text.print-issued-lines:` |

*For full migration details and automated conversion, see [references/v1-to-v2-migration.md](references/v1-to-v2-migration.md).*

---

### 3. Mandatory Verification Gate
**Never output a configuration to the user without validating it first.** Every generated or edited configuration MUST pass:

```bash
golangci-lint config verify -c .golangci.yml
# Or run the bundled python script:
python3 scripts/verify_config.py .golangci.yml
```

If verification fails, fix the configuration and re-run verification before responding.

---

### 4. Runtime Linter Discovery
Linter availability and deprecation statuses shift across releases (e.g. `wsl` → `wsl_v5`, `gomodguard` → `gomodguard_v2`). **Never rely on a hardcoded list of linters.**

Derive active linter names and deprecations at runtime:

```bash
golangci-lint help linters --json
# Or run the bundled discovery script:
python3 scripts/get_linters.py [--deprecated] [--group standard|fast|all] [--search <term>]
```

*For details on linter JSON parsing and group strategies, see [references/linter-discovery.md](references/linter-discovery.md).*

---

## Standard Workflow for Creating / Updating Configs

1. **Verify Binary Installation**: Ensure `golangci-lint` is installed from prebuilt binary (`references/installation.md`).
2. **Discover Installed Linters**: Run `golangci-lint help linters --json` (or `scripts/get_linters.py`) to verify exact linter names and filter out deprecated linters.
3. **Select Linter Default**: Set `linters.default: standard` (recommended) or `none`.
4. **Configure Formatters**: Put code formatters (`gofumpt`, `gci`, `gofmt`, `golines`) inside the top-level `formatters` block.
5. **Define Exclusions & Test Rules**: Place path and linter exclusions inside `linters.exclusions.rules` and `linters.exclusions.paths`.
6. **Set Adoption Strategy**:
   - For existing/legacy codebases, use `issues.new-from-merge-base: main` to gate only new PR changes.
   - For greenfield codebases, set `issues.max-issues-per-linter: 0`.
7. **Integrate Custom AST Rules**: Configure `gocritic` with `ruleguard` (`linters.settings.gocritic.settings.ruleguard`) if custom project rules are required.
8. **Run Verification Gate**: Execute `golangci-lint config verify` on the final YAML file.

---

## Reference Guides & Worked Examples

- **[Installation Guide](references/installation.md)**: Local binary install, package managers, GitHub Actions, and why `go install` is officially discouraged.
- **[v1 to v2 Migration Guide](references/v1-to-v2-migration.md)**: Exhaustive trap table and `golangci-lint migrate` instructions.
- **[Linter Discovery Reference](references/linter-discovery.md)**: Querying linter JSON, deprecation tables, and linter groups.
- **[Incremental Adoption Guide](references/incremental-adoption.md)**: Gating new code with `new-from-merge-base`, test exclusions, and debt remediation strategies.
- **[Ruleguard & gocritic Guide](references/ruleguard-gocritic.md)**: Writing dynamic Go AST inspection rules in `.ruleguard.go` without recompiling.
- **[Production Worked Configs](references/worked-configs.md)**: Verified sample configs for Go Libraries, Backend Services/Daemons, and Incremental Legacy Adoption.
