---
name: go-docs
description: Use when analyzing Go code for documentation completeness, adding doc comments, or ensuring documentation follows Effective Go and Google Style Guide standards. Triggers on phrases like "document the API", "add doc comments", "check documentation", or "complete docs".
---

# Go Documentation

## Overview

Analyze and complete Go documentation following Effective Go and Google Style Guide standards. Ensures package comments, function documentation, and exported identifiers follow Go conventions.

## When to Use

- Adding documentation to Go code
- Checking doc completeness before review
- Ensuring package comments exist
- Verifying doc comment style compliance

## Core Principles

### Doc Comment Standards

**From Effective Go:**
- Doc comments appear before the declaration with no intervening blank lines
- Use complete sentences starting with the name being documented
- Package comments precede the package clause

**From Google Style:**
- Comments explain "why", not "what" (code shows what)
- Documentation must stand on its own
- Allow self-describing code; avoid redundant comments

### Package Comments

```go
// Package bytes implements functions for the manipulation of byte slices.
// It is analogous to the facilities of the strings package.
package bytes
```

**Rules:**
- First sentence: `Package {name} ...` - complete sentence summary
- Include examples of major usage patterns
- Multiple files: one file should have package comment (usually doc.go)

**Package Comment Template:**
```go
// Package {{name}} {{one-sentence description}}.
//
// {{Longer description of package purpose and usage.}}
//
// {{Usage example:}}
//
//	{{code example showing typical usage}}
//
// {{Any important caveats or notes.}}
package {{name}}
```

### Function/Method Comments

```go
// Join combines any number of path elements into a single path,
// adding a separating slash if necessary.
func Join(elem ...string) string { ... }

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained. The return value n is the number of bytes read. If the
// buffer has no data to return, err is io.EOF (unless len(p) is zero);
// otherwise it is nil.
func (b *Buffer) Read(p []byte) (n int, err error) { ... }
```

**Rules:**
- Start with the function name
- Explain what it does, not how
- Document parameters and return values
- Keep to 1-2 sentences for simple functions

### Type Comments

```go
// Buffer is a variable-sized buffer of bytes with Read and Write methods.
// The zero value for Buffer is an empty buffer ready to use.
type Buffer struct { ... }

// Reader is the interface that wraps the basic Read method.
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

### Variable/Constant Comments

```go
// ErrNotExist indicates that a file or directory does not exist.
var ErrNotExist = errors.New("file does not exist")

// MaxInt is the maximum value an int can hold.
const MaxInt = int(^uint(0) >> 1)
```

## Documentation Patterns

### Complete Function Documentation

```go
// ProcessBatch processes a batch of items and returns the number processed
// and any error encountered. If ctx is cancelled, ProcessBatch returns
// immediately with the items processed so far.
//
// The items slice must not be modified during processing.
func ProcessBatch(ctx context.Context, items []Item) (int, error) { ... }
```

### Interface with Implementation Note

```go
// StringWriter is the interface that wraps the WriteString method.
//
// Note: Most implementations also implement io.Writer, but StringWriter
// is provided as an optimization for string-heavy operations.
type StringWriter interface {
    WriteString(s string) (n int, err error)
}
```

### Deprecated API Documentation

```go
// OldMethod performs some operation.
//
// Deprecated: Use NewMethod instead. OldMethod will be removed in v2.0.
func OldMethod() { ... }
```

## Common Mistakes

| Mistake | Correction |
|---------|------------|
| `// Returns the user name` | `// GetUser returns the user name.` |
| `// TODO: document this` | Write proper documentation |
| `// setName sets the name` | `// SetName sets the user name.` |
| `// package foo ...` | `// Package foo ...` (capital P) |
| Missing period at end | Add period for complete sentence |
| Explains implementation | Explains purpose |

## Commands

```bash
# Analyze documentation coverage
python3 ~/.claude/skills/go-docs/scripts/analyze_docs.py

# Analyze specific package
python3 ~/.claude/skills/go-docs/scripts/analyze_docs.py --package mypackage

# JSON output for CI integration
python3 ~/.claude/skills/go-docs/scripts/analyze_docs.py --format json

# Check via go doc
go doc mypackage
go doc mypackage.MyFunction
```

## Priority Order

When documenting a codebase:

1. **Package documentation** - Every package needs a doc comment
2. **Exported types** - Public structs, interfaces
3. **Constructor functions** - `New*`, `Create*` functions
4. **Core methods** - Methods on primary types
5. **Exported functions** - Other public functions
6. **Exported constants/variables** - With context

## References

- [Effective Go - Commentary](https://go.dev/doc/effective_go#commentary)
- [Google Go Style - Comment](https://google.github.io/styleguide/go/decisions#comment)
- [Go Doc Comments](https://go.dev/doc/comment)
