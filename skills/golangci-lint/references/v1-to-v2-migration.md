# golangci-lint v1 to v2 Migration & Trap Table

golangci-lint v2 introduced a breaking overhaul of the configuration schema. Config files written using v1 syntax are invalid under v2 and will fail validation or execution.

Every v2 configuration file MUST specify `version: "2"` at the top level.

---

## The v1 → v2 Trap Table

The following table summarizes the structural breaking changes between v1 and v2.

| Field / Area | v1 Syntax (INVALID in v2) | v2 Syntax (REQUIRED) | Key Differences |
| :--- | :--- | :--- | :--- |
| **Schema Version** | Omitted or ignored | `version: "2"` | Top-level string field mandatory in v2. |
| **Linter Settings** | `linters-settings:` (top-level) | `linters.settings:` | Moved under the `linters` root block. |
| **Enable All Linters** | `linters.enable-all: true` | `linters.default: all` | `enable-all` replaced by `default: all \| standard \| fast \| none`. |
| **Formatters Location** | `gofmt`, `gofumpt`, `goimports`, `gci`, `golines`, `swaggo` in `linters.enable` | `formatters.enable:` and `formatters.settings:` | Formatters moved out of `linters` into top-level `formatters` block. |
| **Exclusion Rules** | `issues.exclude-rules:` | `linters.exclusions.rules:` | Exclusions moved from `issues` into `linters.exclusions`. |
| **Excluded Paths** | `run.skip-dirs:`, `run.skip-files:`, `issues.exclude-dirs:`, `issues.exclude-files:` | `linters.exclusions.paths:` | Path exclusions unified under `linters.exclusions.paths`. |
| **Default Exclusions** | `issues.use-default-rules: false` | `linters.exclusions.presets:` | Presets control default exclusion groups (`comments`, `std-error-handling`, etc.). |
| **Generated Files** | `run.skip-dirs-use-default:` | `linters.exclusions.generated:` | `strict`, `lax`, or `disable` modes. |
| **Text Output Settings** | `output.print-issued-lines:`, `output.print-linter-name:` | `output.formats.text.print-issued-lines:`, `output.formats.text.print-linter-name:` | Text output flags nested inside `output.formats.text`. |

---

## Detailed Structural Breakdown

### 1. Formatters moved out of `linters.enable`

In v1, formatting tools like `gofmt`, `gofumpt`, `goimports`, `gci`, `golines`, and `swaggo` were enabled inside `linters.enable`. In v2, they belong exclusively to the top-level `formatters` section:

```yaml
# INVALID v1 syntax
linters:
  enable:
    - gofumpt
    - gci

# VALID v2 syntax
formatters:
  enable:
    - gofumpt
    - gci
  settings:
    gci:
      sections:
        - standard
        - default
```

### 2. `linters-settings` -> `linters.settings`

```yaml
# INVALID v1 syntax
linters-settings:
  govet:
    shadow: true

# VALID v2 syntax
linters:
  settings:
    govet:
      enable:
        - shadow
```

### 3. `issues.exclude-rules` -> `linters.exclusions.rules`

```yaml
# INVALID v1 syntax
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck

# VALID v2 syntax
linters:
  exclusions:
    rules:
      - path: _test\.go
        linters:
          - errcheck
```

### 4. `output` Format nesting

```yaml
# INVALID v1 syntax
output:
  format: colored-line-number
  print-linter-name: true

# VALID v2 syntax
output:
  formats:
    text:
      print-linter-name: true
      colors: true
```

---

## Automated Migration with `golangci-lint migrate`

To convert an existing v1 configuration file to v2 automatically, use the `migrate` subcommand:

```bash
# Preview migration output
golangci-lint migrate

# Verify the migrated configuration
golangci-lint config verify
```

`golangci-lint migrate` reads `.golangci.yml` in the current directory (or `-c <path>`) and converts all v1 directives into valid v2 syntax.
