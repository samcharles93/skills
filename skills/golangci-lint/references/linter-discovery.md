# Dynamic Runtime Linter Discovery

Linter availability, deprecations, and configuration parameters shift across `golangci-lint` releases. **Never hardcode linter names or rely on static lists from memory.**

Always query the running `golangci-lint` binary directly to determine active linters, groups, and deprecation statuses.

---

## Querying Linters via JSON

Run `golangci-lint help linters --json` to inspect all linters registered in the installed binary.

```bash
golangci-lint help linters --json
```

Each linter entry in the JSON array contains the following fields:

```json
{
  "name": "gomodguard_v2",
  "description": "Allow and blocklist linter for direct Go module dependencies.",
  "groups": ["all", "fast"],
  "fast": true,
  "autoFix": false,
  "deprecated": false,
  "since": "v2.12.0",
  "originalURL": "https://github.com/ryancurrah/gomodguard"
}
```

### Key JSON Fields to Inspect

- **`name`**: Exact string identifier to use in `linters.enable` or `linters.disable`.
- **`deprecated`**: Boolean (`true` / `false`). **If `true`, DO NOT use in new configurations.** Look for the replacement linter (e.g. `wsl` -> `wsl_v5`, `gomodguard` -> `gomodguard_v2`).
- **`fast`**: Indicates whether the linter runs without full package type-checking (`fast: true`).
- **`groups`**: Included groups (`standard`, `fast`, `all`).
- **`autoFix`**: Indicates if the linter supports automatic code fixes via `--fix`.

---

## Filtering Deprecated Linters

You can use the bundled helper script to inspect active vs. deprecated linters:

```bash
# List deprecated linters in the current binary
python3 scripts/get_linters.py --deprecated

# Search for linters matching a query
python3 scripts/get_linters.py --search context

# List fast linters only
python3 scripts/get_linters.py --group fast
```

### Common Deprecations & Replacements (v2.12+)

| Deprecated Linter | Replacement Linter | Reason |
| :--- | :--- | :--- |
| `wsl` | `wsl_v5` | Upgraded to wsl v5 major release |
| `gomodguard` | `gomodguard_v2` | Upgraded to gomodguard v2 major release |

---

## Linter Group Strategies

In `golangci-lint` v2, linter defaults are set via `linters.default`:

- **`default: standard`** (Recommended): Enables the core curated set of reliable, low-false-positive linters (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`).
- **`default: fast`**: Enables fast AST-only linters (ideal for pre-commit hooks).
- **`default: none`**: Disables all linters by default; requires explicit enablement in `linters.enable`.
- **`default: all`**: Enables every linter (NOT recommended for CI without granular exclusions, as it enables noisy/opinionated linters).
