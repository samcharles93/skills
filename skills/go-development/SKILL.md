---
name: go-development
description: This skill should be used when the user asks to write, review, refactor or debug Go code, or says "is this idiomatic Go", "review this Go code", "fix lint errors", "run the quality gate", "improve code quality", "clean up code smells", "format this Go code", "wrap this error", "add a table-driven test", or mentions gofmt, gofumpt, staticcheck, go vet or govulncheck. Covers Go naming, error handling, interface and concurrency idioms, the reproduce-fix-prove change loop, and the quality gate to run before calling a change done. For authoring a .golangci.yml specifically, use the golangci-lint skill.
---

# Go Development

## Overview

Go development guide combining Effective Go, the Google Style Guide, Uber Go
Style and modern Go 1.24-1.27 patterns. The process and the gate come first because
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

Any failure met along the way gets the same treatment, not just the defect the
work started from. A compile error, an unexpected test failure or a panic
surfaced while implementing something else is a gap in coverage: write the test
that would have caught it, then fix. The exception is pure test-harness misuse
with no production counterpart, such as combining `t.Parallel()` with
`t.Setenv()`, which Go forbids outright. Fix those directly and say why no test
was added.

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
gofumpt -l .                 # formatting (empty output = pass)
go build ./...
go vet ./...
golangci-lint run ./...      # the substantive gate, see below
staticcheck ./...
govulncheck ./...            # known vulnerabilities in dependencies
go test ./...
go test -race ./...          # mandatory for any concurrency change
```

**Format with `gofumpt`, not `gofmt`.** gofumpt is a strict superset, so
`gofmt -l` reports a file clean that `gofumpt -l` still flags, and a gate built
on gofmt gives false green. Install it with `go install mvdan.cc/gofumpt@latest`
rather than falling back to gofmt silently. Use gofmt only where a project has
deliberately standardised on it.

Prefer a project's own gate (`make check`, `task check`) when one exists, and
run it rather than a hand-assembled subset. Check what it actually covers before
trusting it: a project gate commonly omits `-race`, in which case the race check
is a separate invocation such as `task test:race`.

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

**`./...` stops at the module boundary.** A repository holding more than one
`go.mod` leaves every sibling module untested by a run from the root, and the
output gives no hint that it skipped them. Find them first:

```bash
find . -name go.mod -not -path './vendor/*'
```

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

## Verification and test hygiene

**No test may depend on the network.** `go test ./...` must pass with no external
service reachable and no credentials present. A test that reaches a live service
is worse than no test: slow, non-deterministic, and it fails in CI. Gate live
checks behind an env var with `t.Skip` when unset, or keep them outside the
suite as a manual step whose real output you report. Prove the suite is offline
by running it with each dependency's address pointed at a dead endpoint such as
`http://127.0.0.1:1`. If it still passes, the claim holds.

**A hung test is the finding, not an obstacle.** Do not widen `-test.timeout`, add
retries, or raise a budget to make a hang disappear; that buries the defect. Read
the goroutine dump: it names the culprit, where reasoning about it produces
stories.

**`httptest.Server.Close()` deadlocks** when a handler blocks until the request
context is cancelled: `Close` waits for outstanding handlers, so the test blocks
in its own cleanup. Have the handler block on a channel the test owns and close
it *before* `Close()`.

**Check the module flag.** With `GOFLAGS=-mod=mod` set, any `go build` silently
rewrites `go.mod`, promoting indirect requirements to direct. Verify dependency
changes with `-mod=readonly` and diff the manifest against the merge base. A
legitimate promotion is fine, a moved version is not.

**`go build ./cmd/<name>` writes a `<name>` binary into the working directory.** Gitignore
it, and check `git status` for other stray build output.

**Verify an HTTP contract against the running router, not the route
registrations.** Reading nesting from source indentation is unreliable. Probe it:
a wrong-method request (`PUT`) returns **405** on a registered path and **404** on
an unregistered one. That needs no fixtures and settles structure decisively.

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

### Modern Go

Write for the module's `go` directive and use what it allows; the Go team's
blog posts at go.dev/blog are the authoritative guide to each feature's intended
use. Defaults worth knowing:

- Write the plain allocation pattern (`make` with a size hint, `append` from
  nil, return the slice). Since 1.25-1.26 the compiler keeps small non-escaping
  backing stores on the stack, so manual tricks rarely pay.
- Generic methods (1.27) organise code around a type; they never satisfy an
  interface method.
- Retire an exported API with `// Deprecated:` plus `//go:fix inline` on a
  forwarding implementation, and let `go fix` migrate callers.
- After raising the `go` directive, run `go fix ./...` until it stops changing
  anything, once per GOOS/GOARCH, and review the diff. See `references/go-fix.md`.
- For hangs and leaks in running services: `/debug/pprof/goroutineleak` (1.27)
  and `trace.FlightRecorder` (1.25).
- SIMD is experimental and churns between releases: read `references/simd.md`
  before touching `simd` or `simd/archsimd`.

### Dependencies and external APIs

Do not answer "what version", "what does this package export", "is it
maintained" or "is it vulnerable" from memory. Training data lags real releases
and API signatures. When the plugin's `go-workspace` MCP server is connected,
use its `pkgsite_*` tools:

| Question | Tool |
|---|---|
| Which package does X? | `pkgsite_search` (`symbol` to search by identifier) |
| Latest version; retracted or deprecated? | `pkgsite_versions` |
| Exact exported API at a version | `pkgsite_symbols`, or `pkgsite_package` with `doc: "md"` |
| Known vulnerabilities and the fix version | `pkgsite_vulns` with `version` |
| Who depends on it (adoption, change impact) | `pkgsite_imported_by` |

Pass `version` to match the project's `go.mod`, not the latest. An ambiguous
path errors with the candidate modules; retry with `module` set. Without the
server, `go list -m -versions <mod>`, `go doc <pkg>@<version>` (1.27) and
`govulncheck` answer the same questions locally.

### Concurrency

**Do not communicate by sharing memory; share memory by communicating.**

- Make goroutine lifecycles explicit, never fire-and-forget
- Prefer synchronous APIs
- Use `chan struct{}` for signals
- Close channels from the sender only

**Read a dependency's dispatch defaults before designing around them.** A
library that looks event-driven may run a single worker and call handlers
synchronously, which serialises everything downstream of it and makes control
commands undeliverable while a handler is busy. That failure looks like a
working design right up until it matters. Confirm from the source rather than
the README:

```bash
go list -m -f '{{.Dir}}' github.com/some/dep
```

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
| `references/modern-features.md` | Go 1.24-1.27: generic methods, generic interfaces, JSON v2, `//go:fix inline`, leak profiles, allocation changes |
| `references/go-fix.md` | Modernizer workflow: fixed-point runs, per-platform runs, analyzer list, `newexpr` |
| `references/simd.md` | `simd` and `simd/archsimd`: dispatch, feature checks, 1.26-1.27 renames, gate commands |
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
- [The Go Blog](https://go.dev/blog/all): design rationale and usage guidance for each new feature; check it when adopting one
- [Go release notes](https://go.dev/doc/devel/release)

## Related Skills

Bundled alongside this skill in the go-dev-toolkit plugin:

- **golangci-lint** for authoring `.golangci.yml`, the v2 schema, incremental adoption
- **go-mcp-creator** for MCP servers built with the official Go SDK
- **huma-api-builder** for REST APIs with HUMA v2
