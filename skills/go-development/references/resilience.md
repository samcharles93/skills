# Resilience Patterns in Go

## Retry Logic

### Exponential Backoff with Jitter

```go
type RetryConfig struct {
    MaxAttempts  int           // default 3
    InitialDelay time.Duration // default 100ms
    MaxDelay     time.Duration // default 30s
    Factor       float64       // default 2.0
    OnRetry      func(attempt int, err error, nextDelay time.Duration) // optional, for logs/metrics
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func(context.Context) error) (err error) {
    delay := cfg.InitialDelay
    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        if err = fn(ctx); err == nil {
            return nil
        }
        if errors.Is(err, context.Canceled) || !isRetryable(err) {
            return err
        }
        if attempt == cfg.MaxAttempts {
            break
        }
        if cfg.OnRetry != nil {
            cfg.OnRetry(attempt, err, delay)
        }
        select {
        case <-time.After(delay + time.Duration(rand.Int64N(int64(delay/2)))): // jitter
        case <-ctx.Done():
            return ctx.Err()
        }
        delay = min(time.Duration(float64(delay)*cfg.Factor), cfg.MaxDelay)
    }
    return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, err)
}

// isRetryable keeps an explicit list. Defaulting to true is how non-idempotent
// calls get duplicated on unknown errors. net.Error.Temporary is deprecated;
// Timeout is the reliable signal.
func isRetryable(err error) bool {
    var netErr net.Error
    return errors.As(err, &netErr) && netErr.Timeout()
}
```
## Graceful Shutdown

### Shutdown Handler

```go
type ShutdownManager struct {
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    timeout  time.Duration
    mu       sync.Mutex
    cleanups []func(context.Context) error
}

func NewShutdownManager(timeout time.Duration) *ShutdownManager {
    ctx, cancel := context.WithCancel(context.Background())
    return &ShutdownManager{ctx: ctx, cancel: cancel, timeout: timeout}
}

func (sm *ShutdownManager) RegisterCleanup(fn func(context.Context) error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.cleanups = append(sm.cleanups, fn)
}

// Shutdown cancels the context, waits up to timeout for workers, then runs
// cleanups in reverse registration order.
func (sm *ShutdownManager) Shutdown() {
    sm.cancel()
    done := make(chan struct{})
    go func() { sm.wg.Wait(); close(done) }()
    timer := time.NewTimer(sm.timeout)
    defer timer.Stop()
    select {
    case <-done:
    case <-timer.C:
    }
    ctx, cancel := context.WithTimeout(context.Background(), sm.timeout)
    defer cancel()
    sm.mu.Lock()
    cleanups := sm.cleanups
    sm.mu.Unlock()
    for i := len(cleanups) - 1; i >= 0; i-- {
        _ = cleanups[i](ctx) // log the error; shutdown must not block on it
    }
}
```

```go
func main() {
    sm := NewShutdownManager(30 * time.Second)
    sm.RegisterCleanup(func(ctx context.Context) error { return db.Close() })

    server := &http.Server{Addr: ":8080"}
    sm.RegisterCleanup(server.Shutdown)
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    sm.wg.Go(func() { worker(sm.ctx) }) // workers: wg.Go at start, runs until ctx done

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh
    sm.Shutdown()
}
```
## Circuit Breaker

Prefer a maintained library (for example `github.com/sony/gobreaker`) in
production; the state machine is easy to get subtly wrong. The shape, if you
roll it:

```go
type CircuitBreaker struct {
    mu         sync.Mutex
    open       bool
    failures   int
    threshold  int
    resetAfter time.Duration
    lastFail   time.Time
}

var ErrCircuitOpen = errors.New("circuit breaker is open")

func (cb *CircuitBreaker) Execute(fn func() error) error {
    cb.mu.Lock()
    if cb.open && time.Since(cb.lastFail) > cb.resetAfter {
        cb.open = false // half-open probe
    }
    if cb.open {
        cb.mu.Unlock()
        return ErrCircuitOpen
    }
    cb.mu.Unlock()

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()
    if err != nil {
        cb.failures++
        cb.lastFail = time.Now()
        if cb.failures >= cb.threshold {
            cb.open = true
        }
    } else {
        cb.failures = 0
    }
    return err
}
```
## Timeout Patterns

Wrap at the call site and let the context propagate; only spawn a goroutine
plus `select` when the callee cannot take a context:

```go
ctx, cancel := context.WithTimeout(ctx, d)
defer cancel()
err := fn(ctx)
```
## Rate Limiting

Use `golang.org/x/time/rate` directly; a wrapper struct adds nothing:

```go
limiter := rate.NewLimiter(rate.Limit(rps), burst)
if err := limiter.Wait(ctx); err != nil {
    return err
}
// or limiter.Allow() to skip instead of blocking
```
## Health Checks

Register named checks, run them under the request context, report per-check
status. Serve at `/healthz` (liveness) and `/readyz` (readiness) with plain
`net/http`; see `architecture.md` for the handler pattern.

```go
type HealthChecker struct {
    mu     sync.RWMutex
    checks map[string]func(context.Context) error
}

func (hc *HealthChecker) Check(ctx context.Context) (healthy bool, report map[string]string) {
    hc.mu.RLock()
    defer hc.mu.RUnlock()
    report = make(map[string]string, len(hc.checks))
    for name, check := range hc.checks {
        if err := check(ctx); err != nil {
            report[name] = "unhealthy: " + err.Error()
            healthy = false
        } else {
            report[name] = "healthy"
        }
    }
    return healthy, report
}
```