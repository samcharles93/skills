# Dynamic AST Pattern Rules with gocritic & ruleguard

`ruleguard` is a powerful extension integrated into `gocritic` that allows teams to write project-specific AST inspection rules using a Go-based DSL **without recompiling `golangci-lint`**.

---

## 1. Enabling ruleguard in golangci-lint v2

In `.golangci.yml` (v2 schema), enable `gocritic` in `linters.enable` and configure `ruleguard` under `linters.settings.gocritic.settings.ruleguard`:

```yaml
version: "2"

linters:
  enable:
    - gocritic
  settings:
    gocritic:
      settings:
        ruleguard:
          # Path or glob pattern pointing to ruleguard Go files
          rules: "rules.go"
          # Fail if rules contain DSL syntax or import errors
          failOn: dsl,import

formatters:
  enable:
    - gofmt
```

---

## 2. Writing Ruleguard Rules (`rules.go`)

Create a `rules.go` file (or `rules/*.go`) in your repository. Ruleguard rules are written in standard Go syntax using the `gorule` DSL package.

### Example `rules.go`

```go
//go:build ruleguard
// +build ruleguard

package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// ForbidDirectTimeNow guards against non-deterministic time calls in core domain code.
func ForbidDirectTimeNow(m dsl.Matcher) {
	m.Match(`time.Now()`).
		Where(m["_"].File().Imports("github.com/myorg/myapp/internal/domain")).
		Report("Do not call time.Now() directly in domain logic; inject a Clock interface or time provider.")
}

// EnforceWrappedErrors flags unwrapped fmt.Errorf calls when wrapping external errors.
func EnforceWrappedErrors(m dsl.Matcher) {
	m.Match(`fmt.Errorf($format, $err)`).
		Where(m["err"].Type.Implements("error") && !m["format"].Text.Contains("%w")).
		Report("Error passed to fmt.Errorf should use %w formatting directive instead of %v/%s.")
}

// AvoidRawContextBackground flags raw context.Background() inside request handlers.
func AvoidRawContextBackground(m dsl.Matcher) {
	m.Match(`context.Background()`).
		Where(m["_"].File().Name.Regexp(`.*_handler\.go`)).
		Report("Do not use context.Background() in HTTP/RPC handlers; use r.Context() or pass request context.")
}
```

---

## 3. Key Ruleguard Concepts

- **`m.Match(pattern)`**: Matches Go code expressions using standard Go syntax and `$var` placeholders.
- **`m.Where(condition)`**: Applies filters (e.g. type checks `Implements("error")`, file path filters, import checks).
- **`m.Report(msg)`**: Emits a custom lint error message reported directly by `golangci-lint`.
- **`m.Suggest(replacement)`**: Provides suggested code fixes.
