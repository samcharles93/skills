---
name: go-modernize
description: Use when modernizing Go code to use Go 1.22+ features, updating old patterns, or converting legacy code. Triggers on phrases like "modernize", "update to Go 1.22", "use range N", "apply min/max", or "convert interface{}".
---

# Go Modernize

## Overview

Apply modern Go 1.22+ patterns beyond what `go fix` handles. Converts old-style patterns to modern idioms for cleaner, more efficient code.

## When to Use

- Updating code to Go 1.22+
- Finding old-style patterns
- Codebase modernization
- Learning new Go features

## Modern Patterns

### Range Over Integers (Go 1.22+)

```go
// Old
for i := 0; i < n; i++ { ... }

// Modern
for i := range n { ... }

// Old (inclusive)
for i := 0; i <= n; i++ { ... }

// Modern
for i := range n + 1 { ... }
```

### Min/Max Builtins (Go 1.21+)

```go
// Old
if x < min { x = min }

// Modern
x = max(x, min)

// Old
if x > max { x = max }

// Modern
x = min(x, max)
```

### Any Alias (Go 1.18+)

```go
// Old
func process(v interface{}) { ... }

// Modern
func process(v any) { ... }

// Old
var m = make(map[string]interface{})

// Modern
var m = make(map[string]any)
```

### Benchmark Loop (Go 1.24+)

```go
// Old
func BenchmarkFoo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        doWork()
    }
}

// Modern
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {
        doWork()
    }
}
```

### Slices Package (Go 1.21+)

```go
import "slices"

// Old
sort.Strings(names)

// Modern
slices.Sort(names)

// Old (manual contains)
func contains(slice []string, s string) bool {
    for _, v := range slice {
        if v == s { return true }
    }
    return false
}

// Modern
if slices.Contains(names, "alice") { ... }
```

### Maps Package (Go 1.21+)

```go
import "maps"

// Old (manual copy)
copy := make(map[string]int, len(orig))
for k, v := range orig {
    copy[k] = v
}

// Modern
copy := maps.Clone(orig)

// Old (manual keys)
var keys []string
for k := range m {
    keys = append(keys, k)
}

// Modern
keys := slices.Collect(maps.Keys(m))
```

### WaitGroup.Go (Go 1.25+)

```go
// Old
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    processTask()
}()
wg.Wait()

// Modern
var wg sync.WaitGroup
wg.Go(processTask)  // Go 1.25+
wg.Wait()
```

## Commands

```bash
# Check for modernization opportunities
python3 ~/.claude/skills/go-modernize/scripts/check_modern_go.py

# Preview go fix changes
go fix -diff ./...

# Apply go fix
go fix ./...

# JSON output
python3 ~/.claude/skills/go-modernize/scripts/check_modern_go.py --format json
```

## Transformation Table

| Old | Modern | Since |
|-----|--------|-------|
| `for i:=0; i<n; i++` | `for i := range n` | 1.22 |
| `for i:=0; i<=n; i++` | `for i := range n+1` | 1.22 |
| `if x<min { x=min }` | `x = max(x, min)` | 1.21 |
| `if x>max { x=max }` | `x = min(x, max)` | 1.21 |
| `interface{}` | `any` | 1.18 |
| `for i:=0; i<b.N; i++` | `for b.Loop()` | 1.24 |
| `sort.Strings` | `slices.Sort` | 1.21 |
| `wg.Add(1); go func` | `wg.Go(fn)` | 1.25 |

## Go Fix Integration

```bash
# Preview all changes
go fix -diff .

# Apply safe transformations
go fix .

# Common transformations:
# - Loop variable capture fixes
# - Deprecated API updates
# - Type alias modernizations
```

## Modern Features Reference

### Generic Functions (1.18+)

```go
// Generic container
type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T) { s[v] = struct{}{} }
func (s Set[T]) Contains(v T) bool { _, ok := s[v]; return ok }
```

### Generic Type Aliases (Go 1.24+)

```go
// Old
type StringSet = map[string]bool

// Modern
type Set[T comparable] = map[T]bool

// Usage
users := Set[string]{"alice": true}
```

### Context After Function (Go 1.21+)

```go
// Context is first result from functions that need it
func (*Codec).AppendEncode(dst, src []byte) ([]byte, error)
```

## References

- [Go 1.22 Release Notes](https://go.dev/doc/go1.22)
- [Go 1.21 Release Notes](https://go.dev/doc/go1.21)
- [Go 1.24 Release Notes](https://go.dev/doc/go1.24)
