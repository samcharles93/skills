# Uber Go Style Guide Reference

This document contains key patterns and conventions from the Uber Go Style Guide.

## Error Handling (Uber Style)

### Handle Errors Once

Handle an error in only one place. Either log it OR return it, not both.

```go
// Bad: Error handled twice
if err := doSomething(); err != nil {
    log.Printf("Failed to do something: %v", err)  // Log
    return err  // AND return
}

// Good: Return error to let caller decide
if err := doSomething(); err != nil {
    return fmt.Errorf("do something: %w", err)
}

// Good: Log and swallow at appropriate level
if err := cleanup(); err != nil {
    logger.Warn("cleanup failed", zap.Error(err))
    // Don't return - cleanup failure is non-critical
}
```

### Error Prefix Conventions

Skip "failed to" prefixes. Context should add value.

```go
// Bad: Redundant "failed"
return fmt.Errorf("failed to open file: %w", err)

// Good: Clear context
return fmt.Errorf("open config: %w", err)

// Bad: Generic "error"
return fmt.Errorf("error getting user: %w", err)

// Good: Specific action
return fmt.Errorf("get user %d: %w", userID, err)
```

### Sentinel Errors

Export errors with `Err` prefix, custom types with `Error` suffix.

```go
// Sentinel errors
var (
    ErrNotFound    = errors.New("not found")
    ErrInvalidID   = errors.New("invalid ID")
)

// Custom error types
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}
```

### Error Inspection

Always use `errors.Is` or `errors.As`.

```go
// Good: errors.Is
if errors.Is(err, ErrNotFound) {
    // handle
}

// Good: errors.As
var notFound *NotFoundError
if errors.As(err, &notFound) {
    // use notFound.Resource
}

// Bad: String comparison
if err.Error() == "not found" { ... }
```

## Interface Design

### Verify Interface Compliance

Use compile-time checks at package level.

```go
// At package level, verify type implements interface
var _ http.Handler = (*Handler)(nil)

// For concrete types implementing multiple interfaces
var (
    _ io.Reader   = (*MyReader)(nil)
    _ io.Writer   = (*MyReader)(nil)
    _ io.Closer   = (*MyReader)(nil)
)
```

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

### Table-Driven Test Pattern

```go
func TestParseUser(t *testing.T) {
    tests := []struct {
        name    string
        give    string      // Input - use give prefix
        want    *User       // Expected output - use want prefix
        wantErr error
    }{
        {
            name: "valid user",
            give: "alice:admin",
            want: &User{Name: "alice", Role: "admin"},
        },
        {
            name:    "invalid format",
            give:    "invalid",
            wantErr: ErrInvalidFormat,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseUser(tt.give)
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

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
        tt := tt  // Capture for goroutine (pre-Go 1.22)
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Safe with captured tt
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

### Functional Options

For extensible APIs with many optional parameters.

```go
type ClientOption func(*clientOptions)

type clientOptions struct {
    timeout time.Duration
    retries int
    logger  *zap.Logger
}

func WithTimeout(d time.Duration) ClientOption {
    return func(o *clientOptions) {
        o.timeout = d
    }
}

func WithLogger(l *zap.Logger) ClientOption {
    return func(o *clientOptions) {
        o.logger = l
    }
}

func NewClient(endpoint string, opts ...ClientOption) (*Client, error) {
    options := clientOptions{
        timeout: defaultTimeout,
        retries: defaultRetries,
    }
    for _, opt := range opts {
        opt(&options)
    }
    // ...
}

// Usage
client, err := NewClient("http://api",
    WithTimeout(5*time.Second),
    WithLogger(logger),
)
```

## References

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
