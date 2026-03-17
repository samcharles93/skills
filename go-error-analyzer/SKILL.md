---
name: go-error-analyzer
description: Use when checking Go code for error handling issues, finding discarded errors, or ensuring proper error propagation. Triggers on phrases like "check error handling", "find unhandled errors", "fix error ignores", or "error analysis".
---

# Go Error Analyzer

## Overview

Find and fix unhandled errors in Go code following Effective Go, Google Style, and Uber Go patterns. Validates error wrapping, inspection, and propagation.

## When to Use

- Checking for discarded errors (`_`)
- Ensuring proper error wrapping
- Finding unchecked errors
- Validating error string format
- Checking sentinel error usage

## Core Principles

### Error Handling Rules

**From Effective Go:**
- Errors are values, not exceptions
- Functions return `(value, error)` tuples
- Check errors explicitly

**From Google Style:**
- Error strings: lowercase, no punctuation
- Wrap with `%w` at end of message
- Never discard errors without consideration

### Error Return Pattern

```go
// Good: explicit error check
f, err := os.Open(filename)
if err != nil {
    return fmt.Errorf("open config: %w", err)
}
defer f.Close()

// Process with file
```

### Error Wrapping

```go
// Good: wrap with context, %w at end
if err := db.Query(); err != nil {
    return fmt.Errorf("fetch user %d: %w", userID, err)
}

// Good: %v for system boundaries (no unwrapping)
return nil, fmt.Errorf("remote call failed: %v", err)
```

### Error Inspection

```go
// Good: errors.Is for sentinel values
if errors.Is(err, fs.ErrNotExist) {
    // handle missing file
}

// Good: errors.As for typed errors
var pathErr *fs.PathError
if errors.As(err, &pathErr) {
    // use pathErr.Path
}
```

### Error String Format

```go
// Good: lowercase, no punctuation
return errors.New("file not found")

// Bad: capitalized
return errors.New("File not found")

// Bad: ending punctuation
return errors.New("file not found.")
```

## Error Handling Patterns

### Sentinel Errors

```go
// Define at package level
var (
    ErrNotFound    = errors.New("not found")
    ErrInvalid     = errors.New("invalid input")
    ErrDuplicate   = errors.New("duplicate entry")
)

// Check with errors.Is
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Custom Error Types

```go
// Define structured error type
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// Usage
return &NotFoundError{Resource: "user", ID: userID}
```

### Error Chain

```go
// Layer 1: database
if err := db.Query(...); err != nil {
    return fmt.Errorf("query failed: %w", err)
}

// Layer 2: service
if err := repo.FindUser(id); err != nil {
    return fmt.Errorf("get user %d: %w", id, err)
}

// Layer 3: handler
user, err := service.GetUser(id)
if err != nil {
    if errors.Is(err, ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    http.Error(w, err.Error(), 500)
}
```

## Commands

```bash
# Find all unhandled errors
python3 ~/.claude/skills/go-error-analyzer/scripts/find_unhandled_errors.py

# Check specific directory
python3 ~/.claude/skills/go-error-analyzer/scripts/find_unhandled_errors.py ./pkg

# JSON output
python3 ~/.claude/skills/go-error-analyzer/scripts/find_unhandled_errors.py --format json
```

## Severity Levels

| Level | Pattern | Action |
|-------|---------|--------|
| Critical | `_, _ = func()` | Always handle |
| Warning | `func()` without check | Check or document why safe |
| Info | Error not wrapped | Add context with `%w` |

## Common Patterns

### Must Functions (Limited Use)

```go
// OK only for constants at startup
var validPattern = regexp.MustCompile(`[a-z]+`)

// NOT OK for runtime values
func MustParseUser(s string) *User {  // Don't do this
    user, err := ParseUser(s)
    if err != nil {
        panic(err)
    }
    return user
}
```

### Error Handling in Goroutines

```go
// Good: error channel
type Result struct {
    Value string
    Err   error
}

results := make(chan Result)
go func() {
    val, err := process()
    results <- Result{Value: val, Err: err}
}()
res := <-results
if res.Err != nil {
    // handle error
}
```

## Anti-Patterns

| Pattern | Why Bad | Fix |
|---------|---------|-----|
| `_ = file.Close()` | Silent failure | Check or document |
| `err.Error() == "..."` | Fragile | `errors.Is` |
| `panic(err)` | Normal errors | Return error |
| `%w` at system boundary | Leaks internals | Use `%v` |
| `log.Error` + return | Duplicate | Return only |

## References

- [Effective Go - Errors](https://go.dev/doc/effective_go#errors)
- [Google Go Style - Error](https://google.github.io/styleguide/go/decisions#error)
- [Google Go Best Practices - Errors](https://google.github.io/styleguide/go/best-practices#error-handling)
