# Uber Go Style Guide Reference

This document contains key patterns and conventions from the Uber Go Style Guide.

## Error Handling

The rules (handle once, lowercase without punctuation or a "failed to" prefix,
wrap with `%w`, inspect with `errors.Is`/`errors.As`) are the core principles in
SKILL.md. The Uber-specific nuance: at the top level of a service, log-and-
swallow is legitimate for non-critical cleanup, as long as it is logged once.
## Interface Design

Compile-time conformance checks (`var _ http.Handler = (*Handler)(nil)`) are a
core principle in SKILL.md.
### Interface Pointer Semantics

Avoid pointers to interfaces.

```go
// Bad: Pointer to interface
func process(r *io.Reader) { ... }

// Good: Interface value (can hold pointer)
func process(r io.Reader) { ... }
```

### Mutex Embedding

Don't embed mutexes; use named fields.

```go
// Bad: Embedded mutex
type Cache struct {
    sync.Mutex
    data map[string]string
}

// Good: Named field
type Cache struct {
    mu   sync.Mutex
    data map[string]string
}

// Usage: c.mu.Lock() - clear ownership
```

## Function Organization

### Grouping Order

```go
// 1. Type definition
type Server struct { ... }

// 2. Exported functions (after type, before methods)
func NewServer(addr string) *Server { ... }

// 3. Exported methods (grouped by receiver, rough call order)
func (s *Server) Start() error { ... }
func (s *Server) Stop() error { ... }
func (s *Server) Handle(req *Request) (*Response, error) { ... }

// 4. Unexported methods
func (s *Server) setup() error { ... }

// 5. Utility functions (at file end)
func validateAddress(addr string) error { ... }
```

## Import Organization

Two groups only: stdlib, then third-party.

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/some/package"
    "go.uber.org/zap"
)
```

## Testing (Uber Style)

Table-driven tests are the default test shape; see `testing.md`.
### Avoid Complex Logic in Tests

```go
// Bad: Complex conditional in test
for _, tt := range tests {
    got := Process(tt.input)
    if tt.expected == nil {
        if got != nil {
            t.Error("expected nil")
        }
    } else {
        if got == nil || got.Value != tt.expected.Value {
            t.Errorf("got %v, want %v", got, tt.expected)
        }
    }
}

// Good: Split into separate test functions
func TestProcess_NilInput(t *testing.T) { ... }
func TestProcess_ValidInput(t *testing.T) { ... }
```

### Parallel Test Safety

```go
func TestParallel(t *testing.T) {
    tests := []struct{ name string; input int }{
        {"small", 10},
        {"large", 1000},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Safe: since Go 1.22 each iteration has its own tt
            result := Process(tt.input)
            assert.Equal(t, tt.input*2, result)
        })
    }
}
```

## Uber-Specific Conventions

### Global Variables

Prefix unexported globals with `_` (except errors).

```go
// Good: Unexported global with _ prefix
var _defaultTimeout = 30 * time.Second
var _maxRetries = 3

// Exception: Errors use err prefix
var errClosed = errors.New("connection closed")

// Exported globals - no prefix
var DefaultConfig = Config{...}
```

### Type-Safe Atomics

Use `go.uber.org/atomic` instead of `sync/atomic`.

```go
// Bad: sync/atomic - easy to misuse
var counter int64
atomic.AddInt64(&counter, 1)

// Good: go.uber.org/atomic - type safe
var counter atomic.Int64
counter.Inc()
current := counter.Load()
```

### Channel Buffering

Channel sizes: "one or none".

```go
// Unbuffered: synchronous communication
ch := make(chan int)

// Size 1: for async signal, single value buffer
ch := make(chan int, 1)

// Bad: Arbitrary buffer size
ch := make(chan int, 10)  // Why 10?
```

### Enum Starting Values

Start enums at 1 with `iota + 1`, unless 0 has meaning.

```go
// Good: iota + 1 - invalid zero value forces explicit choice
type Status int

const (
    StatusUnknown Status = iota // 0 - invalid
    StatusPending               // 1
    StatusRunning               // 2
    StatusComplete              // 3
)

// Exception: When 0 is valid default
type LogLevel int

const (
    LogLevelInfo LogLevel = iota  // 0 - valid default
    LogLevelDebug
    LogLevelWarn
    LogLevelError
)
```

### Time Types

- `time.Time` for absolute instants
- `time.Duration` for relative periods

```go
// Bad: int for time
func wait(seconds int) { ... }

// Good: time.Duration
func wait(d time.Duration) { ... }

// Usage
wait(5 * time.Second)
```

### Goroutine Lifecycle

Never "fire and forget" goroutines.

```go
// Bad: Fire and forget
go process(data)

// Good: Lifecycle management
type Worker struct {
    done chan struct{}
    wg   sync.WaitGroup
}

func (w *Worker) Start() {
    w.wg.Add(1)
    go func() {
        defer w.wg.Done()
        for {
            select {
            case <-w.done:
                return
            case data := <-w.queue:
                process(data)
            }
        }
    }()
}

func (w *Worker) Stop() {
    close(w.done)
    w.wg.Wait()
}
```

### No Goroutines in init()

```go
// Bad: Goroutine in init
func init() {
    go cleanupOldFiles()  // Don't do this
}

// Good: Explicit startup
func StartCleaner(ctx context.Context) {
    go cleanupOldFiles(ctx)
}
```

### Exit Only in main()

```go
// Good: main delegates to run(), returns error
func main() {
    os.Exit(run())
}

func run() int {
    if err := setup(); err != nil {
        fmt.Fprintf(os.Stderr, "setup: %v\n", err)
        return 1
    }
    // ... do work
    return 0
}
```

### Slice/Map Boundaries

Copy at boundaries to prevent data races.

```go
// Bad: Returns internal slice directly
func (s *Server) GetHandlers() []Handler {
    return s.handlers  // Caller can modify internal state
}

// Good: Return copy
func (s *Server) GetHandlers() []Handler {
    handlers := make([]Handler, len(s.handlers))
    copy(handlers, s.handlers)
    return handlers
}

// Good: Or return read-only view
func (s *Server) RangeHandlers(fn func(Handler)) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, h := range s.handlers {
        fn(h)
    }
}
```

### String Conversion

Prefer `strconv` over `fmt` on hot paths.

```go
// Bad: fmt for simple conversions (slower)
port := fmt.Sprintf("%d", p)

// Good: strconv (faster)
port := strconv.Itoa(p)

// Good: strconv for formatting too
port := strconv.FormatInt(int64(p), 10)
```

### Map Initialization

Use `make()` instead of literals when empty.

```go
// Bad: Empty literal
m := map[string]int{}

// Good: make
m := make(map[string]int)

// Good: With capacity hint
m := make(map[string]int, expectedSize)
```

### Zero Values

```go
// Good: Use var for zero values
var p Point  // Zero value ready to use

// Bad: Explicit zero value
p := Point{}  // Unnecessary

// Good: &T{} for pointers
p := &Point{X: 1, Y: 2}

// Bad: new(T) for structs
p := new(Point)  // Less common in Uber style
```

### Reduce Nesting

Handle errors early, avoid unnecessary else.

```go
// Bad: Deep nesting
if err := check(); err == nil {
    if data, err := fetch(); err == nil {
        if valid(data) {
            // ... actual logic
        }
    }
}

// Good: Early returns
if err := check(); err != nil {
    return err
}
data, err := fetch()
if err != nil {
    return err
}
if !valid(data) {
    return fmt.Errorf("invalid data")
}
// ... actual logic
```

### Line Length

Soft limit of 99 characters. Break long lines appropriately.

```go
// Break after open paren for long function signatures
func NewClientWithConfig(
    endpoint string,
    timeout time.Duration,
    retries int,
) (*Client, error) {
    // ...
}

// Break chained calls
db.Model(&User{}).
    Where("active = ?", true).
    Order("created_at DESC").
    Find(&users)
```

Functional options for API configuration are covered in `api-design.md`.
## References

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
