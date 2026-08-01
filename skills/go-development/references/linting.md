# Go Linting and Code Quality

## Doctrine: a gate that can be argued with enforces nothing

A linter is an environmental constraint, not a suggestion. Two failure modes
destroy it, and both feel reasonable at the time:

**Silencing findings to make the number look good.** Adding an exclusion
because a linter flagged something inconvenient is deciding what the linter is
allowed to say. The finding count drops, the code does not improve, and the
next reader trusts a gate that is no longer checking anything. Fix the code, or
suppress the single site with a stated reason — never widen the rule set's
blind spot to cover a category.

**Keeping a gate so noisy it gets ignored.** A rule that fires hundreds of
times on idiomatic code trains everyone to skim past output, including the
three findings that matter. Curate the enabled set once, deliberately, then
drive the count to zero and hold it there.

The distinction is where the judgement is applied: choosing which linters to
enable is a design decision made once and reviewed; excluding a linter after it
flags real code is usually the beginning of a dead gate.

## When a finding must NOT be applied

Some findings are correct in general and wrong for the specific site. Applying
them mechanically introduces bugs. Recognise these before "fixing":

- **Suggested simplifications that change semantics.** `QF1008` ("could remove
  embedded field from selector") on a type that embeds the very interface it
  delegates to turns explicit delegation into infinite recursion.
- **`nilerr` on deliberate skip logic.** A `filepath.Walk` callback that
  returns `nil` on an unreadable entry is skipping it on purpose; returning the
  error aborts the whole walk.
- **Complexity limits on a genuinely irreducible algorithm.** Splitting a
  parser or a state machine to satisfy a line count can make it harder to read.

For these, suppress the single line or function with `//nolint:linter // reason`
and state the reason in terms of behaviour, not preference. A `//nolint`
without a reason is indistinguishable from giving up.

## golangci-lint v2 Configuration

golangci-lint v2 uses a new YAML structure. Here's a production-ready configuration:

```yaml
# .golangci.yml
version: "2"
run:
  tests: true

linters:
  default: none
  enable:
    # Bugs & Correctness (Critical)
    - govet           # Go vet checks
    - staticcheck     # Comprehensive static analysis
    - errcheck        # Unchecked errors
    - errorlint       # Error wrapping issues
    - bodyclose       # HTTP response body close
    - noctx           # HTTP requests without context
    - durationcheck   # Detects time.Second * time.Second bugs
    - nilerr          # Catches return nil when err != nil
    - nilnesserr      # Checks err != nil but returns different nil
    - fatcontext      # Detects nested contexts in loops
    - contextcheck    # Non-inherited context usage
    - copyloopvar     # Loop variable copy issues (Go 1.22+)
    - forcetypeassert # Unchecked type assertions (panic risk)
    - makezero        # Slice with non-zero initial length bugs

    # Security
    - gosec           # Security issues

    # Performance
    - prealloc        # Slice preallocation suggestions
    - unconvert       # Unnecessary type conversions
    - perfsprint      # Faster sprintf alternatives

    # Style & Maintainability
    - gocyclo         # Cyclomatic complexity
    - gocognit        # Cognitive complexity
    - funlen          # Function length limits
    - nestif          # Nested if statement depth
    - ineffassign     # Ineffective assignments
    - unused          # Unused code detection
    - misspell        # Spelling mistakes
    - revive          # Fast, configurable linter
    - gocritic        # Opinionated linter

    # Modernization (Go 1.21+)
    - intrange        # Use for range n
    - usestdlibvars   # Use http.StatusOK instead of 200
    - modernize       # Modern Go features

    # Testing Quality
    - thelper         # Test helpers should call t.Helper()
    - tparallel       # Correct t.Parallel() usage

  settings:
    gocyclo:
      min-complexity: 15
    gocognit:
      min-complexity: 30
    funlen:
      lines: 80
      statements: 50
    nestif:
      min-complexity: 4
    misspell:
      locale: US
    errcheck:
      check-type-assertions: true
      check-blank: false  # Allow explicit _ = err

  exclusions:
    generated: lax
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
    rules:
      # Exclude complexity checks in test files
      - linters:
          - gocyclo
          - gocognit
          - funlen
          - nestif
        path: _test\.go

      # Example: Exclude inherently complex functions
      # - linters:
      #     - gocyclo
      #     - gocognit
      #   path: parser\.go
      #   text: "(parse|complexFunction)"

formatters:
  enable:
    - gci       # Import grouping
    - gofumpt   # Stricter gofmt
  settings:
    gci:
      sections:
        - standard
        - default
        - prefix(github.com/your-org/your-project)
```


## Adopting golangci-lint on an existing codebase

A first run on a mature repository commonly reports hundreds of findings. Work
through it as a measured backlog, not a single sweep.

### 1. Establish the baseline

```bash
golangci-lint run ./... 2>&1 | tail -30    # show-stats prints per-linter counts
```

Record the total and the per-linter breakdown. This number is the thing being
driven to zero, so configure `max-issues-per-linter: 0` and
`max-same-issues: 0` — the defaults truncate output and make the count
untrackable.

### 2. Split findings by kind

Three groups, handled differently:

| Kind | Examples | Approach |
| --- | --- | --- |
| Formatters | gofmt, gofumpt, goimports, gci | Auto-fix in one commit: `golangci-lint fmt ./...` |
| Mechanical correctness | errcheck, bodyclose, noctx, errorlint | Fix in bulk; each is a local, provable change |
| Structural | gocognit, cyclop, funlen, interfacebloat | Needs judgement; schedule as real work |

Splitting production from test findings is usually more informative than
splitting by linter:

```bash
golangci-lint run ./... 2>/dev/null | grep -E '^[a-z].*\.go:' | \
  sed -E 's/^([^:]+):[0-9]+.*\(([a-z]+)\)$/\2 \1/' | \
  awk '{ t=($2 ~ /_test\.go$/)?"test":"PROD"; print $1"\t"t }' | sort | uniq -c | sort -rn
```

### 3. Fix formatters first, separately

Run the formatter as its own commit so the semantic diff that follows is
readable. When other work is in flight on specific files, hold those back
rather than forcing a merge conflict:

```bash
golangci-lint fmt ./...
git checkout -- path/being/edited/elsewhere.go
```

### 4. Fix tests, then production

Test findings are usually mechanical and are worth fixing rather than
excluding — they are the cheapest possible practice for the gate. Two classes
carry real value:

- **`noctx` in tests**: an HTTP call or subprocess without a context means a
  hung server hangs the whole package instead of failing one test. Bind them to
  `t.Context()`.
- **`bodyclose`**: includes the `*http.Response` returned by WebSocket
  handshakes (`websocket.Dialer.Dial`), which is almost always discarded.

Where many call sites share a shape, add one small helper per test package with
a signature matching the function being replaced, so the call sites change
minimally:

```go
func postJSON(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
    request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
    if err != nil {
        return nil, err
    }
    request.Header.Set("Content-Type", "application/json")
    return http.DefaultClient.Do(request)
}
```

### 5. Verify after every batch

Bulk edits — especially scripted ones — break code in ways the linter does not
see. Run `go build ./... && go vet ./... && go test ./...` after each batch. A
blanket `_ =` prefix, for instance, silently breaks any call returning more
than one value.

## Linting every build target

golangci-lint analyses **one build target at a time** and defaults to the host
platform. Files behind build constraints are invisible to a plain run:

```bash
golangci-lint run ./...                              # host only
GOOS=js GOARCH=wasm golangci-lint run ./...          # js/wasm sources
GOOS=windows golangci-lint run ./...                 # windows-only files
```

Any project with `//go:build js`, `//go:build windows` or similar needs a
second invocation in its Makefile, or those files are simply unlinted. Note
that a directory containing two package names (a `main.go` behind a build tag
alongside a package file) will fail the typecheck on the target where both are
visible; lint those packages explicitly by path.

## Verifying the config itself

```bash
golangci-lint config verify        # validate against the schema before committing
golangci-lint linters              # list enabled/disabled linters for this config
```

Run `config verify` after every edit. A typo in a linter name is silently
ignored by some versions, leaving a rule that was never enabled.

## Linter Selection Strategy

### By Category

| Category | Linters | Priority |
|----------|---------|----------|
| **Bugs** | govet, staticcheck, errcheck, nilerr | Critical |
| **Security** | gosec, bidichk | High |
| **Performance** | prealloc, unconvert, perfsprint | Medium |
| **Style** | gocyclo, funlen, revive, gocritic | Medium |
| **Modernization** | intrange, modernize, usestdlibvars | Low |

### Adding Exclusions Properly

When a linter flags inherently complex code that cannot be simplified:

```yaml
exclusions:
  rules:
    # Document WHY the exclusion is needed
    # Next() and Prev() have inherent complexity due to
    # multi-field time calculation with wraparound logic
    - linters:
        - gocognit
        - gocyclo
      path: spec\.go
      text: "(Next|Prev)"
```

**Best Practice**: Always add a comment explaining why the exclusion is justified.

## Common staticcheck/revive Fixes

### ST1005: Error String Formatting

Error strings should NOT be capitalized or end with punctuation:

```go
// BAD - Will trigger ST1005
return errors.New("H expressions require a hash key")
return fmt.Errorf("Invalid input: %s.", input)

// GOOD
return errors.New("h expressions require a hash key")
return fmt.Errorf("invalid input: %s", input)
```

**Rationale**: Error messages are often wrapped or concatenated. Lowercase prevents awkward capitalization like `"failed: Invalid input"`.

### ST1003: Naming Conventions

```go
// BAD
var serverId string    // Should be serverID
func GetUserId() {}    // Should be GetUserID
type HttpClient struct  // Should be HTTPClient

// GOOD
var serverID string
func GetUserID() {}
type HTTPClient struct
```

### gosec G104: Unhandled Errors

For functions that always return nil errors (like `hash.Hash.Write`):

```go
// BAD - gosec G104 warning
h := fnv.New64a()
h.Write([]byte(key))  // Error unhandled

// GOOD - Explicitly acknowledge the ignored return
h := fnv.New64a()
_, _ = h.Write([]byte(key))  // hash.Hash.Write never returns error
```

**When to use `_, _ =`**:
- `hash.Hash.Write()` - Never returns error per spec
- `bytes.Buffer.Write()` - Never returns error
- `strings.Builder.WriteString()` - Never returns error

### revive: Error Naming

```go
// BAD
var InvalidInput = errors.New("invalid input")  // Should start with Err
type ValidationFailed struct{}                  // Should end with Error

// GOOD
var ErrInvalidInput = errors.New("invalid input")
type ValidationError struct{}
```

## Running Linters

### Development Workflow

```bash
# Quick check during development
golangci-lint run --fast

# Full check before commit
golangci-lint run

# Check specific files
golangci-lint run ./pkg/...

# Auto-fix where possible
golangci-lint run --fix
```

### CI Configuration

```yaml
# GitHub Actions
- name: golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: v1.62
    args: --timeout 5m
```

### Pre-commit Hook (lefthook)

```yaml
# .lefthook.yml
pre-commit:
  parallel: true
  commands:
    lint:
      glob: "*.go"
      run: golangci-lint run --new-from-rev=HEAD~1
```

## Complexity Guidelines

| Metric | Threshold | Action if Exceeded |
|--------|-----------|-------------------|
| Cyclomatic (gocyclo) | 15 | Refactor or document why justified |
| Cognitive (gocognit) | 30 | Simplify or add exclusion with rationale |
| Function Length | 80 lines | Extract helper functions |
| Nesting Depth | 4 levels | Refactor with early returns |

### When Complexity is Justified

Some functions have inherent complexity that cannot be reduced without fragmenting the algorithm:

1. **Parser functions** - Multiple input formats require branching
2. **Time calculations** - Multi-field wraparound (year/month/day/hour/minute/second)
3. **State machines** - Multiple states and transitions

In these cases, add an exclusion with clear documentation.

## Makefile Integration

```makefile
.PHONY: lint lint-full lint-fix

# Quick lint for development
lint:
	golangci-lint run --fast

# Full lint for CI
lint-full:
	golangci-lint run --timeout 5m

# Auto-fix issues
lint-fix:
	golangci-lint run --fix

# Run specific linters only
lint-security:
	golangci-lint run -E gosec,bidichk

lint-bugs:
	golangci-lint run -E govet,staticcheck,errcheck,nilerr
```
