# Production-Ready Worked Configs (v2 Schema)

All configurations in this reference use the mandatory `version: "2"` schema and have passed verification via `golangci-lint config verify`.

---

## 1. Go Library / Package (`library.golangci.yml`)

Tailored for standalone Go libraries with strict API standards, high test coverage, zero external dependencies, and thorough documentation requirements.

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  default: standard
  enable:
    - asasalint
    - bodyclose
    - canonicalheader
    - copyloopvar
    - dupword
    - durationcheck
    - errname
    - errorlint
    - exhaustive
    - gocheckcompilerdirectives
    - goconst
    - gocritic
    - godot
    - gosec
    - inamedparam
    - intrange
    - misspell
    - nilerr
    - noctx
    - nolintlint
    - perfsprint
    - revive
    - sloglint
    - staticcheck
    - testifylint
    - unconvert
    - unparam
    - usestdlibvars
    - usetesting
    - whitespace
  exclusions:
    rules:
      - path: _test\.go
        linters:
          - errcheck
          - gosec
          - noctx

formatters:
  enable:
    - gci
    - gofmt
    - gofumpt
  settings:
    gci:
      sections:
        - standard
        - default

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

output:
  formats:
    text:
      print-linter-name: true
```

---

## 2. Go Service / Daemon (`service.golangci.yml`)

Designed for backend services, APIs, microservices, and resident daemons. Includes database leak checks (`sqlclosecheck`, `rowserrcheck`), logging checks (`sloglint`, `loggercheck`), print-statement restrictions (`forbidigo`), and custom AST rules via `gocritic` + `ruleguard`.

```yaml
version: "2"

run:
  timeout: 10m
  tests: true

linters:
  default: standard
  enable:
    - bodyclose
    - canonicalheader
    - copyloopvar
    - dupword
    - durationcheck
    - errcheck
    - errname
    - errorlint
    - exhaustive
    - fatcontext
    - forbidigo
    - gocheckcompilerdirectives
    - goconst
    - gocritic
    - godot
    - gosec
    - govet
    - ineffassign
    - intrange
    - loggercheck
    - makezero
    - misspell
    - nilerr
    - nilnil
    - noctx
    - nolintlint
    - perfsprint
    - reassign
    - revive
    - rowserrcheck
    - sloglint
    - sqlclosecheck
    - staticcheck
    - testifylint
    - unconvert
    - unparam
    - unused
    - usestdlibvars
    - usetesting
    - whitespace
  settings:
    forbidigo:
      forbid:
        - pattern: ^fmt\.Print.*$
          msg: Do not use fmt.Print in service code; use log/slog or structured logger.
    gocritic:
      settings:
        ruleguard:
          rules: rules.go
          failOn: dsl,import
  exclusions:
    rules:
      - path: _test\.go
        linters:
          - errcheck
          - gosec
          - noctx
          - forbidigo

formatters:
  enable:
    - gci
    - gofumpt
  settings:
    gci:
      sections:
        - standard
        - default

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

output:
  formats:
    text:
      print-linter-name: true
```

---

## 3. Legacy Codebase Incremental Adoption (`legacy.golangci.yml`)

Designed for introducing `golangci-lint` to an existing codebase without blocking ongoing development or requiring massive upfront remediation.

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  default: standard
  enable:
    - copyloopvar
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
  exclusions:
    generated: lax
    rules:
      - path: _test\.go
        linters:
          - errcheck

formatters:
  enable:
    - gofmt

issues:
  # Gate only new code added relative to the merge-base against main branch
  new-from-merge-base: main
  whole-files: false
  # Do not cap issues so devs see all violations in new code
  max-issues-per-linter: 0
  max-same-issues: 0

output:
  formats:
    text:
      print-linter-name: true
```
