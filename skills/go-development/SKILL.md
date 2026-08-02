---
name: go-development
description: This skill should be used when the user asks to write, review, refactor or debug Go code, or says "is this idiomatic Go", "review this Go code", "fix lint errors", "run the quality gate", "improve code quality", "clean up code smells", "format this Go code", "wrap this error", "add a table-driven test", or mentions gofmt, gofumpt, staticcheck, go vet or govulncheck. Covers Go naming, error handling, interface and concurrency idioms, the reproduce-fix-prove change loop, and the quality gate to run before calling a change done. For authoring a .golangci.yml specifically, use the golangci-lint skill.
---

# Go Development

## Overview

Go development guide combining Effective Go, the Google Style Guide, Uber Go
Style and modern Go 1.24+ patterns. The process and the gate come first because
they apply to every change; the style material below them is reference to consult
as needed.

## When to Use

Use this skill for any Go work:

- Writing, reviewing or refactoring Go code
- Adding features or fixing bugs in a Go project
- Running the quality gate before calling a change done
- Applying Go idioms for naming, errors, interfaces and concurrency

For authoring a `.golangci.yml`, use the companion `golangci-lint` skill. For an
MCP server or a HUMA API, use `go-mcp-creator` and `huma-api-builder`.

## Development Process

Apply this loop to every change. Each step produces the evidence the next one
relies on. Full detail in `references/development-process.md`.

1. **Understand** the existing behaviour and the invariant it protects. Read the
   code and its tests, not a summary of them. Verify any claim made by a prior
   report against the source before acting on it.
2. **Reproduce** the defect with a failing test *before* changing the
   implementation, and confirm it fails for the stated reason. For a performance
   defect, the reproduction is a recorded baseline measurement.
3. **Change** the smallest thing that addresses the cause rather than the
   symptom. Leave unrelated cleanup out of the commit.
4. **Prove** the fix: the reproduction passes, and critically, *mutation-check*
   it by reverting the fix to confirm the test fails again. A test that passes
   either way has proved nothing.
5. **Gate**: run the full quality gate below, not just the changed package.
6. **Report** honestly: what was verified, what was skipped, what remains.

## Quality Gate

Run before completing any Go work. `scripts/lint-gate.sh` runs the whole
sequence and reports every failing stage rather than stopping at the first.

Invoke it by absolute path. `SKILL_DIR` below stands for the directory holding
this SKILL.md; substitute the real path. A bare relative path resolves against
the working directory, which is the Go project rather than this skill:

```bash
"$SKILL_DIR/scripts/lint-gate.sh"            # full gate
"$SKILL_DIR/scripts/lint-gate.sh" --quick    # skip -race and govulncheck during the loop
"$SKILL_DIR/scripts/lint-gate.sh" --fix      # apply formatters and safe fixes first
```

Equivalent steps, in order:

```bash
gofmt -l .                   # formatting (empty output = pass)
go build ./...
go vet ./...
golangci-lint run ./...      # the substantive gate, see below
staticcheck ./...
govulncheck ./...            # known vulnerabilities in dependencies
go test ./...
go test -race ./...          # mandatory for any concurrency change
```

Prefer a project's own gate (`make check`, `task check`) when one exists.

**The gate only covers the host platform.** `go build`, `go vet` and `go test`
skip every file behind a build tag, so code under `//go:build js`,
`//go:build windows` and similar stays unchecked while the gate reports green.
Find the tagged files and re-run per target:

```bash
grep -rl '//go:build' --include='*.go' .
GOOS=js GOARCH=wasm go vet ./...
```

`-race` is not universally available either; `js/wasm` rejects it outright, so
note which build targets the race gate did not cover rather than implying it
covered everything.

For deeper analysis, run the checkers bundled with this skill. Each takes the
project directory as its first argument and defaults to `.`, so pass the target
explicitly rather than relying on the default. Add `--format json` when
aggregating findings instead of reading them:

```bash
python3 "$SKILL_DIR/scripts/analysis/analyze_docs.py" .          # documentation completeness
python3 "$SKILL_DIR/scripts/analysis/find_unhandled_errors.py" . # swallowed errors
python3 "$SKILL_DIR/scripts/analysis/check_modern_go.py" .       # pre-1.22 patterns
python3 "$SKILL_DIR/scripts/analysis/check_tests.py" .           # test quality
```

## Code Quality with golangci-lint

golangci-lint replaces a scattering of individual tools with one curated,
version-controlled rule set. Use `assets/golangci.yml` as a starting
`.golangci.yml`: it enables correctness linters (errcheck, errorlint, bodyclose,
noctx, nilerr, forcetypeassert, contextcheck), structural limits (cyclop,
gocognit, funlen, nestif) and formatters (gofumpt, goimports, gci).

**Two rules keep a gate alive:**

- **Do not silence findings to lower the count.** Adding an exclusion because a
  linter flagged inconvenient code decides what the linter is allowed to say.
  Fix the code, or suppress the single site with `//nolint:linter // reason`
  stating the behaviour that justifies it.
- **Do not keep a rule that fires constantly on idiomatic code.** Noise trains
  readers to skim past the findings that matter. Curate the enabled set once,
  deliberately, then drive the count to zero.

Some findings must **not** be applied mechanically: they are correct in general
and wrong at that site. Applying `QF1008` to a type that embeds the interface it
delegates to produces infinite recursion; applying `nilerr` to a `filepath.Walk`
callback that deliberately skips unreadable entries aborts the walk. Recognise
these, suppress with a reason, and move on.

Adopting the linter on an existing codebase, splitting findings into
auto-fixable and judgement-requiring groups, and the per-finding fix patterns
are in `references/linting.md`.

## Core Principles

### Names

| Rule | Example | Avoid |
|------|---------|-------|
| Package names: short, lowercase, no underscores | `package bytes` | `package byte_utils` |
| MixedCaps for exported names | `MaxLength` | `MAX_LENGTH`, `max_length` |
| Omit Get prefix | `obj.Owner()` | `obj.GetOwner()` |
| Receivers: 1-2 letters | `(b *Buffer)` | `(buffer *Buffer)` |
| Initialisms uniform case | `URL`, `HTTP` | `Url`, `Http` |

### Error Handling

1. Errors are values, check them explicitly
2. Error strings: lowercase, no punctuation, no "failed to" prefix
3. Wrap with `%w` to allow unwrapping, placed at the end
4. Inspect with `errors.Is` and `errors.As`, never string comparison
5. **Handle once**: log or return, not both

```go
// Bad: handled twice, and the message repeats what the caller already knows.
if err := doSomething(); err != nil {
    log.Printf("Failed to do something: %v", err)
    return err
}

// Good: one handling, naming the action that failed.
if err := os.Open(file); err != nil {
    return fmt.Errorf("open config %s: %w", file, err)
}

// Good: inspect by sentinel or type, never by message text.
if errors.Is(err, fs.ErrNotExist) { ... }

var pathErr *fs.PathError
if errors.As(err, &pathErr) { ... }
```

Avoid: `panic` for ordinary errors, capitalised error strings, and
`err.Error() == "..."` comparisons.

### Documentation

- Complete sentence starting with the name being documented
- Explain what and why, not how
- Stands alone without surrounding context
- Line comments (`//`), not block comments

```go
// Package bytes implements functions for the manipulation of byte slices.
// It is analogous to the facilities of the strings package.
package bytes

// Read reads the next len(p) bytes from the buffer or until the buffer is
// drained. The return value n is the number of bytes read. If the buffer has
// no data to return, err is io.EOF (unless len(p) is zero); otherwise it is nil.
func (b *Buffer) Read(p []byte) (n int, err error) { ... }
```

### Interfaces

1. Define where consumed, not where implemented
2. Keep small; one or two methods is the norm
3. Return concrete types, accept interfaces
4. Name single-method interfaces with an `-er` suffix
5. Assert conformance at compile time: `var _ json.Marshaler = (*MyType)(nil)`

A wide interface is occasionally the right call. A storage or backend
abstraction that genuinely has one implementation per environment may need a
dozen methods, and `interfacebloat` will flag it at eight. Take that as a prompt
to justify the width, not an instruction to split it.

### Values crossing an API boundary

Copying a struct does not copy the arrays behind its slice and map fields, so a
getter returning stored state hands the caller a live pointer into it. Either
return a deep copy, or document that the result must not be modified. Silently
sharing is how a caller mutating a returned payload corrupts committed data.

Related: `nil` and empty slices marshal differently, `null` versus `[]`. Two
implementations of one interface that disagree about which they return will make
clients behave differently for no visible reason.

### Concurrency

**Do not communicate by sharing memory; share memory by communicating.**

- Make goroutine lifecycles explicit, never fire-and-forget
- Prefer synchronous APIs
- Use `chan struct{}` for signals
- Close channels from the sender only

```go
// Explicit lifecycle: the caller can wait, and the exit condition is visible.
var wg sync.WaitGroup
wg.Go(func() { doWork(ctx) })
wg.Wait()
```

## Reference Material

Details deliberately kept out of this file:

| File | Covers |
|---|---|
| `references/development-process.md` | The full change loop, performance work, concurrency changes, test hygiene |
| `references/linting.md` | golangci-lint config, adoption on an existing codebase, per-finding fixes, complexity thresholds |
| `references/testing.md` | Test patterns including table-driven tests, coverage |
| `references/modern-features.md` | Go 1.22-1.26 idioms: range-over-int, `min`/`max`, `b.Loop()`, `WaitGroup.Go` |
| `references/uber-style.md` | Uber conventions: global var naming, channel sizing, enums, worker lifecycle, atomics |
| `references/api-design.md` | API surface decisions including functional options |
| `references/architecture.md` | Project structure |
| `references/resilience.md`, `references/docker.md`, `references/makefile.md` | Operational concerns |
| `references/fuzz-testing.md`, `references/mutation-testing.md` | Deeper verification techniques |

### Scripts

- `scripts/lint-gate.sh` runs the full quality gate, reporting every failing stage
- `scripts/verify-go-project.sh` validates project structure and tooling
- `scripts/analysis/*.py` are the four deeper checkers documented above

### Assets

- `assets/golangci.yml` is a copy-ready `.golangci.yml`; change the gci prefix
  to the module path

## External Reading

- [Effective Go](https://go.dev/doc/effective_go)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
- [Google Go Best Practices](https://google.github.io/styleguide/go/best-practices)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

## Related Skills

Bundled alongside this skill in the go-dev-toolkit plugin:

- **golangci-lint** for authoring `.golangci.yml`, the v2 schema, incremental adoption
- **go-mcp-creator** for MCP servers built with the official Go SDK
- **huma-api-builder** for REST APIs with HUMA v2
