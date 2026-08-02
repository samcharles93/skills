---
name: go-development
description: This skill should be used when the user asks to write, review, refactor or debug Go code, or says "add golangci-lint", "set up linting", "fix lint errors", "improve code quality", "clean up code smells", "format this Go code", "run the quality gate", "add a linter config", or mentions gofmt, gofumpt, staticcheck, go vet, govulncheck or .golangci.yml. Covers the end-to-end development process, code quality gates, documentation, error handling, modernization and testing for Go projects.
---

# Go Development

## Overview

Comprehensive Go development guide combining Effective Go, the Google Style
Guide, Uber Go Style and modern Go 1.24+ patterns. It is self-contained: the
development process, the quality gate and the analysis scripts all ship inside
this skill, so nothing here depends on another skill being installed.

## When to Use

Use this skill for any Go work:

- Writing, reviewing or refactoring Go code
- Adding features or fixing bugs in a Go project
- Running the quality gate before calling a change done
- Setting up or fixing linting, formatting or vulnerability checks
- Learning or applying Go idioms and conventions

For golangci-lint configuration specifically, see the companion `golangci-lint`
skill. For building an MCP server or a HUMA API, see `go-mcp-creator` and
`huma-api-builder`.

## Core Principles

### Names (Effective Go + Google Style)

| Rule | Example | Avoid |
|------|---------|-------|
| Package names: short, lowercase, no underscores | `package bytes` | `package byte_utils` |
| MixedCaps for exported names | `MaxLength` | `MAX_LENGTH`, `max_length` |
| Omit Get prefix | `obj.Owner()` | `obj.GetOwner()` |
| Receivers: 1-2 letters | `(b *Buffer)` | `(buffer *Buffer)` |
| Initialisms uniform case | `URL`, `HTTP` | `Url`, `Http` |

```go
// Package bytes implements functions for the manipulation of byte slices.
package bytes

// Buffer is a variable-sized buffer of bytes with Read and Write methods.
type Buffer struct { ... }

// Reset resets the buffer to be empty.
func (b *Buffer) Reset() { ... }
```

### Error Handling (Effective Go + Google + Uber)

**Core Rules:**
1. Errors are values - check explicitly
2. Error strings: lowercase, no punctuation
3. Wrap with `%w` to allow unwrapping (place at end)
4. Use `errors.Is` and `errors.As` for inspection
5. **Handle Once** - log OR return, not both
6. Skip "failed to" prefixes in error messages

**Uber Style Additions:**
```go
// Bad: Log and return (handled twice)
if err := doSomething(); err != nil {
    log.Printf("Failed to do something: %v", err)
    return err
}

// Good: Handle once - return with context
if err := doSomething(); err != nil {
    return fmt.Errorf("do something: %w", err)
}

// Bad: "failed to" prefix
return fmt.Errorf("failed to open file: %w", err)

// Good: Clear action
return fmt.Errorf("open config: %w", err)
```

```go
// Good: wrap errors with context
if err := os.Open(file); err != nil {
    return fmt.Errorf("open config %s: %w", file, err)
}

// Good: check specific error types
if errors.Is(err, fs.ErrNotExist) {
    // handle missing file
}

// Good: extract typed errors
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    // use pathErr.Path
}
```

**Anti-patterns:**
```go
// Bad: panic for normal errors
if err != nil {
    panic(err)
}

// Bad: capitalized error strings
return errors.New("File not found")  // "file not found"

// Bad: string comparison
if err.Error() == "file not found"  // fragile
```

### Documentation (Effective Go + Google Style)

**Package Comments:**
```go
// Package bytes implements functions for the manipulation of byte slices.
// It is analogous to the facilities of the strings package.
package bytes
```

**Function Comments:**
```go
// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained. The return value n is the number of bytes read. If the
// buffer has no data to return, err is io.EOF (unless len(p) is zero);
// otherwise it is nil.
func (b *Buffer) Read(p []byte) (n int, err error) { ... }
```

**Rules:**
- Complete sentence, start with name being documented
- Explain what and why, not how
- Stand on its own (no surrounding context needed)
- Use line comments (`//`), not block comments

### Interfaces (Effective Go + Google Style)

**Principles:**
1. Define where consumed, not implemented
2. Keep small (1-2 methods ideally)
3. Return concrete types, accept interfaces
4. Name with -er suffix for single method

```go
// Good: interface defined in consuming package
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Good: implicit implementation
type myReader struct{}
func (r *myReader) Read(p []byte) (n int, err error) { ... }

// Good: compile-time check
var _ json.Marshaler = (*MyType)(nil)
```

### Concurrency (Effective Go)

**Core Principle:** Do not communicate by sharing memory; share memory by communicating.

```go
// Good: channels for synchronization
c := make(chan int)
go func() {
    doWork()
    c <- 1  // signal completion
}()
<-c  // wait

// Good: explicit goroutine lifecycle
go func() {
    defer wg.Done()
    // work with clear exit condition
}()
```

**Rules:**
- Make goroutine lifecycles explicit
- Prefer synchronous APIs
- Use `chan struct{}` for signals
- Close channels from sender only


## Development Process

Apply the same loop to every change. Each step produces the evidence the next
one relies on. Full detail in `references/development-process.md`.

1. **Understand** the existing behaviour and the invariant it protects. Read
   the code and its tests, not a summary of them. Verify any claim made by a
   prior report against the source before acting on it.
2. **Reproduce** the defect with a failing test *before* changing the
   implementation — and confirm it actually fails for the stated reason. For a
   performance defect, the reproduction is a recorded baseline measurement.
3. **Change** the smallest thing that addresses the cause rather than the
   symptom. Leave unrelated cleanup out of the commit.
4. **Prove** the fix: the reproduction passes, and — critically —
   *mutation-check* it by reverting the fix to confirm the test fails again. A
   test that passes either way has proved nothing.
5. **Gate**: run the full quality gate below, not just the changed package.
6. **Report** honestly: what was verified, what was skipped, what remains.

## Quality Gate

Run before completing any Go work. `scripts/lint-gate.sh` runs the whole
sequence and reports every failing stage rather than stopping at the first:

```bash
scripts/lint-gate.sh            # full gate
scripts/lint-gate.sh --quick    # skip -race and govulncheck during the loop
scripts/lint-gate.sh --fix      # apply formatters and safe fixes first
```

Equivalent steps, in order:

```bash
gofmt -l .                   # formatting (empty output = pass)
go build ./...
go vet ./...
golangci-lint run ./...      # the substantive gate — see below
staticcheck ./...
govulncheck ./...            # known vulnerabilities in dependencies
go test ./...
go test -race ./...          # mandatory for any concurrency change
```

Prefer a project's own gate (`make check`, `task check`) when one exists.

For deeper analysis, run the checkers bundled with this skill. Paths are
relative to the skill directory, so they resolve under any harness that loads
skills by directory:

```bash
python3 scripts/analysis/analyze_docs.py            # documentation completeness
python3 scripts/analysis/find_unhandled_errors.py   # swallowed errors
python3 scripts/analysis/check_modern_go.py         # pre-1.22 patterns
python3 scripts/analysis/check_tests.py             # test quality
```

## Code Quality with golangci-lint

golangci-lint replaces a scattering of individual tools with one curated,
version-controlled rule set. Use `assets/golangci.yml` as a starting `.golangci.yml`
— it enables correctness linters (errcheck, errorlint, bodyclose, noctx,
nilerr, forcetypeassert, contextcheck), structural limits (cyclop, gocognit,
funlen, nestif) and formatters (gofumpt, goimports, gci).

**Two rules keep a gate alive:**

- **Do not silence findings to lower the count.** Adding an exclusion because a
  linter flagged inconvenient code decides what the linter is allowed to say.
  Fix the code, or suppress the single site with `//nolint:linter // reason`
  stating the behaviour that justifies it.
- **Do not keep a rule that fires constantly on idiomatic code.** Noise trains
  readers to skim past the findings that matter. Curate the enabled set once,
  deliberately, then drive the count to zero.

Some findings must **not** be applied mechanically — they are correct in
general and wrong at that site. Applying `QF1008` to a type that embeds the
interface it delegates to produces infinite recursion; applying `nilerr` to a
`filepath.Walk` callback that deliberately skips unreadable entries aborts the
walk. Recognise these, suppress with a reason, and move on.

**Linting covers one build target at a time.** A plain run only sees the host
platform, so files behind `//go:build js`, `//go:build windows` and similar are
never checked unless run again with `GOOS`/`GOARCH` set.

Adopting the linter on an existing codebase, splitting findings into
auto-fixable and judgement-requiring groups, and the per-finding fix patterns
are in `references/linting.md`.

## Quick Reference: Modern Go (1.22-1.26)

```go
// Go 1.22: range over integers
for i := range n { ... }  // not: for i := 0; i < n; i++

// Go 1.21: min/max builtins
x = max(x, minVal)  // not: if x < minVal { x = minVal }

// Go 1.18+: any alias
func f(v any) { ... }  // not: func f(v interface{})

// Go 1.24: b.Loop() for benchmarks
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {  // not: for i := 0; i < b.N; i++
        doWork()
    }
}

// Go 1.25: WaitGroup.Go()
var wg sync.WaitGroup
wg.Go(processTask)  // not: wg.Add(1); go func() { ... }()
wg.Wait()
```

## Common Patterns

### Table-Driven Tests
```go
func TestAbs(t *testing.T) {
    tests := []struct {
        name     string
        input    float64
        expected float64
    }{
        {"negative", -1, 1},
        {"zero", 0, 0},
        {"positive", 1, 1},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := Abs(tt.input); got != tt.expected {
                t.Errorf("Abs(%v) = %v, want %v", tt.input, got, tt.expected)
            }
        })
    }
}
```

### Error Wrapping Chain
```go
// Each layer adds context
if err := db.Query(); err != nil {
    return fmt.Errorf("fetch user %d: %w", userID, err)
}

// Later, caller can check
if errors.Is(err, sql.ErrNoRows) {
    // user not found
}
```

### Functional Options
```go
type Server struct { ... }

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func WithLogger(l *slog.Logger) Option {
    return func(s *Server) { s.logger = l }
}

func NewServer(options ...Option) *Server {
    s := &Server{timeout: defaultTimeout}
    for _, opt := range options {
        opt(s)
    }
    return s
}

// Usage
srv := NewServer(WithTimeout(5*time.Second), WithLogger(log))
```

## Uber Style Conventions

| Pattern | Uber Style |
|---------|-----------|
| Global vars | `_defaultTimeout` (prefix with `_`) |
| Channel buffers | "One or none" - avoid arbitrary sizes |
| Enums | Start at 1 with `iota + 1` |
| Goroutines | Never fire-and-forget; none in `init()`; explicit exit condition |
| Time types | `time.Time` for instants, `time.Duration` for periods |
| Exit | Only in `main()`, delegate to `run()` |
| String conversion | Prefer `strconv` over `fmt` on hot paths |
| Test fields | `give` for input, `want` for expected |

Full conventions, including worker lifecycle and atomics, in
`references/uber-style.md`.

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
- [Google Go Best Practices](https://google.github.io/styleguide/go/best-practices)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) (see `references/uber-style.md`)

## Additional Resources

### Reference Files

- **`references/development-process.md`** - The full change loop, performance
  work, concurrency changes and test hygiene
- **`references/linting.md`** - golangci-lint config, adoption on an existing
  codebase, per-finding fixes, complexity thresholds
- **`references/testing.md`** - Test patterns and coverage
- **`references/modern-features.md`** - Go 1.22-1.26 idioms
- **`references/uber-style.md`** - Uber Go Style Guide conventions
- **`references/architecture.md`**, **`references/api-design.md`** - Structure
  and API surface decisions
- **`references/resilience.md`**, **`references/docker.md`**,
  **`references/makefile.md`** - Operational concerns
- **`references/fuzz-testing.md`**, **`references/mutation-testing.md`** -
  Deeper verification techniques

### Scripts

- **`scripts/lint-gate.sh`** - Runs the full quality gate, reporting every
  failing stage
- **`scripts/verify-go-project.sh`** - Validates project structure and tooling

### Assets

- **`assets/golangci.yml`** - Copy-ready `.golangci.yml`; change the gci prefix
  to the module path

## Related Skills

Bundled alongside this skill in the go-dev-toolkit plugin:

- **golangci-lint** - configuring `.golangci.yml`, the v2 schema, incremental adoption
- **go-mcp-creator** - MCP servers built with the official Go SDK
- **huma-api-builder** - REST APIs with HUMA v2
