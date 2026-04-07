---
name: go-test-validator
description: Use when validating Go tests for modern patterns, completeness, and best practices. Triggers on phrases like "check tests", "validate tests", "test patterns", "benchmark modernization", or "improve test coverage".
---

# Go Test Validator

## Overview

Validate Go tests for modern patterns and completeness following Google Style Guide and Effective Go conventions. Checks table-driven tests, benchmarks, subtests, and test quality.

## When to Use

- Checking test quality
- Modernizing test patterns
- Validating benchmark style
- Ensuring proper test structure
- Code review of test files

## Core Principles

### Google Style Testing Rules

- Leave testing to test functions (no assertion libraries)
- Mark helpers with `t.Helper()`
- Never call `t.Fatal` from spawned goroutines
- Use field names in table-driven tests
- Prefer real transports over mocking

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
            got := Abs(tt.input)
            if got != tt.expected {
                t.Errorf("Abs(%v) = %v, want %v", tt.input, got, tt.expected)
            }
        })
    }
}
```

### Benchmark Modern (Go 1.24+)

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

### Test Helper Functions

```go
func setupTest(t *testing.T) *Server {
    t.Helper()  // Mark as helper
    srv, err := NewServer()
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }
    t.Cleanup(func() { srv.Close() })
    return srv
}

func TestFeature(t *testing.T) {
    srv := setupTest(t)  // Failure reports caller's line
    // test with srv
}
```

## Test Patterns

### t.Error vs t.Fatal

```go
func TestProcess(t *testing.T) {
    data, err := loadData()
    if err != nil {
        t.Fatalf("setup failed: %v", err)  // Fatal: can't proceed
    }

    result := Process(data)
    if result.Count != 42 {
        t.Errorf("Count = %d, want 42", result.Count)  // Error: can check more
    }
    if result.Name != "test" {
        t.Errorf("Name = %q, want test", result.Name)  // Still runs
    }
}
```

### Goroutine Testing

```go
func TestConcurrent(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < num; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            if err := work(i); err != nil {
                t.Errorf("work(%d): %v", i, err)  // Not t.Fatal!
            }
        }(i)
    }
    wg.Wait()
}
```

### Test Naming

```go
// Good
func TestUser_Create
func TestUser_Delete
func TestUser_Update

// Good table-driven
func TestUser_CRUD

// Bad
func TestUserCreate
func TestUserDelete
```

## Commands

```bash
# Check test files
python3 ~/.claude/skills/go-test-validator/scripts/check_tests.py

# Run tests with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# JSON output
python3 ~/.claude/skills/go-test-validator/scripts/check_tests.py --format json
```

## Test Checklist

- [ ] Table-driven for multiple cases
- [ ] Named struct fields in test cases
- [ ] Subtests with `t.Run`
- [ ] Benchmarks use `b.Loop()` (Go 1.24+)
- [ ] Helpers marked with `t.Helper()`
- [ ] `t.Fatal` only in setup, not goroutines
- [ ] Clear error messages with context
- [ ] Parallel tests with `t.Parallel()`
- [ ] Test cleanup with `t.Cleanup()`

## Anti-Patterns

| Pattern | Why Bad | Fix |
|---------|---------|-----|
| `assertEquals(t, got, want)` | Not idiomatic Go | Explicit check |
| `t.Fatal` in goroutine | Test may hang | Use `t.Error` |
| `if got != want { t.Logf(...) }` | Doesn't fail | Use `t.Errorf` |
| `t.Parallel()` at top | May skip setup | After setup |

## References

- [Google Go Best Practices - Testing](https://google.github.io/styleguide/go/best-practices#testing)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
