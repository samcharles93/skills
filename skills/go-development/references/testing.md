# Go Testing Patterns

## Test Pyramid

```
       E2E Tests (~5-30s)
         Complete scenarios
         Real infrastructure
         Slowest, most brittle

    Integration Tests (~1-5s)
       Real external deps
       Docker, databases
       Moderate speed

  Unit Tests (~<100ms)
     Mocked dependencies
     Fast, reliable
     Highest coverage
```

## Build Tags for Test Isolation

### Tag Convention

```go
// Unit tests - no tags, run by default
// File: job_test.go
package core

func TestJobValidation(t *testing.T) {
    // Fast, no external deps
}

// Integration tests - require real Docker
// File: docker_integration_test.go
//go:build integration

package core

func TestDockerExec(t *testing.T) {
    // Requires Docker daemon
}

// E2E tests - complete system
// File: workflow_e2e_test.go
//go:build e2e

package e2e

func TestFullWorkflow(t *testing.T) {
    // Start server, run jobs, verify
}
```

### Running Tests

```bash
# Unit tests only (CI default)
go test ./...

# With integration tests
go test -tags=integration ./...

# Full suite
go test -tags="integration e2e" ./...

# Specific package with tags
go test -tags=integration ./core/...
```

## Table-Driven Tests

### Basic Pattern

```go
func TestParseSchedule(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Schedule
        wantErr bool
    }{
        {name: "every minute", input: "* * * * *", want: Schedule{Minute: "*", Hour: "*", Day: "*", Month: "*", Weekday: "*"}},
        {name: "invalid", input: "invalid", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseSchedule(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSchedule() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseSchedule() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Prefer sentinel errors with `errors.Is` over the `wantErr bool` dance when a
specific error matters. See `uber-style.md` for the give/want naming variant.
### Subtests with Setup/Teardown

```go
func TestJobExecution(t *testing.T) {
    // Shared setup
    logger := logrus.New()
    logger.SetOutput(io.Discard)

    tests := []struct {
        name    string
        job     Job
        setup   func()
        cleanup func()
        wantErr bool
    }{
        {
            name: "successful execution",
            job:  &LocalJob{Command: []string{"echo", "hello"}},
            setup: func() {
                // Optional per-test setup
            },
            wantErr: false,
        },
        {
            name:    "command not found",
            job:     &LocalJob{Command: []string{"nonexistent"}},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setup != nil {
                tt.setup()
            }
            if tt.cleanup != nil {
                defer tt.cleanup()
            }

            err := tt.job.Run(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Job.Run() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Mocking Patterns

### Interface-Based Mocking

```go
// Define interface in consumer package
type DockerClient interface {
    ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, string, error)
    RunContainer(ctx context.Context, image string, cmd []string) (string, error)
}

// Mock implementation for tests
type MockDockerClient struct {
    ExecFunc func(ctx context.Context, containerID string, cmd []string) (string, string, error)
    RunFunc  func(ctx context.Context, image string, cmd []string) (string, error)
}

func (m *MockDockerClient) ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, string, error) {
    if m.ExecFunc != nil {
        return m.ExecFunc(ctx, containerID, cmd)
    }
    return "", "", nil
}

// Usage in tests
func TestExecJob(t *testing.T) {
    mock := &MockDockerClient{
        ExecFunc: func(ctx context.Context, containerID string, cmd []string) (string, string, error) {
            if containerID != "test-container" {
                return "", "", errors.New("container not found")
            }
            return "output", "", nil
        },
    }

    job := &ExecJob{
        Container: "test-container",
        Command:   []string{"ls", "-la"},
        Client:    mock,
    }

    err := job.Run(context.Background())
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Using testify/mock

```go
import "github.com/stretchr/testify/mock"

type MockDockerClient struct {
    mock.Mock
}

func (m *MockDockerClient) ExecInContainer(ctx context.Context, containerID string, cmd []string) (string, string, error) {
    args := m.Called(ctx, containerID, cmd)
    return args.String(0), args.String(1), args.Error(2)
}

func TestWithTestify(t *testing.T) {
    mockClient := new(MockDockerClient)

    // Set expectations
    mockClient.On("ExecInContainer",
        mock.Anything,
        "container-123",
        []string{"ls"},
    ).Return("file1\nfile2", "", nil)

    job := &ExecJob{
        Container: "container-123",
        Command:   []string{"ls"},
        Client:    mockClient,
    }

    err := job.Run(context.Background())

    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}
```

## Time Control in Tests

Prefer `testing/synctest` (Go 1.24+, see `modern-features.md`) for timeouts,
tickers and timers inside the test bubble: fake time advances instantly once
every goroutine is durably blocked.

For code that takes a `Clock` interface (schedulers, caches, rate limiters),
inject a small fake instead of sleeping. Keep the interface minimal and add a
mutex plus timer/ticker registries on the fake:

```go
type Clock interface {
    Now() time.Time
    NewTicker(d time.Duration) Ticker
    NewTimer(d time.Duration) Timer
}

func (c *FakeClock) Advance(d time.Duration) { /* move now, fire due timers and tickers */ }
```

```go
func TestSchedulerExecutesJobsOnTime(t *testing.T) {
    clock := NewFakeClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
    scheduler := NewScheduler(WithClock(clock))
    executed := false
    scheduler.AddJob("test", "*/5 * * * *", func() { executed = true })
    scheduler.Start()
    defer scheduler.Stop()

    clock.Advance(5 * time.Minute) // instant and deterministic
    assert.True(t, executed)
}
```

Never `time.Sleep` in a test to wait for something; that is flaky by
construction. For condition polling see the Eventually pattern below.
## Test Helpers

### Eventually Pattern (Recommended)

Replace brittle `time.Sleep`-based waiting with event-driven polling:

```go
// test/testutil/eventually.go
package testutil

func Eventually(t testing.TB, condition func() bool, timeout, interval time.Duration) bool {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for {
        if condition() {
            return true
        }
        if time.Now().After(deadline) {
            t.Errorf("Eventually timed out after %v", timeout)
            return false
        }
        time.Sleep(interval)
    }
}
```

Prefer `testing/synctest` (see Time Control above) when the code under test
runs inside the bubble; Eventually is for real I/O and goroutines outside it.

```go
func TestServerStartup(t *testing.T) {
    server := startServer()
    defer server.Stop()

    // GOOD: event-driven, fast and reliable. Never time.Sleep(2*time.Second) instead.
    testutil.Eventually(t, server.IsReady, 5*time.Second, 50*time.Millisecond)
}
```

### Temp Directories

Use the built-in `t.TempDir()`: it creates the directory and registers cleanup
with the test automatically. Do not write a `TempDir` helper.
### Test Fixtures

```go
// test/fixtures/fixtures.go
package fixtures

import (
    "embed"
    "testing"
)

//go:embed *.json *.yaml
var testData embed.FS

func LoadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := testData.ReadFile(name)
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", name, err)
    }
    return data
}

// Usage:
// data := fixtures.LoadFixture(t, "valid_config.yaml")
```

## Integration Test Patterns

### Docker-Based Integration Tests

Behind a build tag, against a real daemon; see `docker.md` for the client
patterns. Skip when the daemon is unreachable:

```go
//go:build integration

package core_test

import (
    "context"
    "testing"
    "time"

    docker "github.com/fsouza/go-dockerclient"
)

func TestDockerIntegration(t *testing.T) {
    client, err := docker.NewClientFromEnv()
    if err != nil || client.Ping() != nil {
        t.Skip("Docker not available")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    t.Run("exec in container", func(t *testing.T) {
        container, err := client.CreateContainer(docker.CreateContainerOptions{
            Config:     &docker.Config{Image: "alpine:latest", Cmd: []string{"sleep", "60"}},
            HostConfig: &docker.HostConfig{AutoRemove: true},
            Context:    ctx,
        })
        if err != nil {
            t.Fatalf("create container: %v", err)
        }
        if err := client.StartContainer(container.ID, nil); err != nil {
            t.Fatalf("start container: %v", err)
        }

        exec, err := client.CreateExec(docker.CreateExecOptions{
            Container: container.ID, Cmd: []string{"echo", "hello"},
            AttachStdout: true, Context: ctx,
        })
        if err != nil {
            t.Fatalf("create exec: %v", err)
        }
        var out bytes.Buffer
        if err := client.StartExec(exec.ID, docker.StartExecOptions{OutputStream: &out, Context: ctx}); err != nil {
            t.Fatalf("exec: %v", err)
        }
        if out.String() != "hello\n" {
            t.Errorf("unexpected output: %q", out.String())
        }
    })
}
```

## Coverage and Benchmarks

### Coverage

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Coverage with build tags
go test -tags=integration -coverprofile=coverage.out ./...
```

### Benchmarks

```go
func BenchmarkJobExecution(b *testing.B) {
    job := &LocalJob{Command: []string{"true"}}
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        job.Run(ctx)
    }
}

func BenchmarkParseSchedule(b *testing.B) {
    schedule := "*/5 * * * *"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ParseSchedule(schedule)
    }
}

// Run benchmarks
// go test -bench=. -benchmem ./...
```

## Test Configuration

### testdata Directory

```
package/
├── parser.go
├── parser_test.go
└── testdata/
    ├── valid_config.ini
    ├── invalid_config.ini
    └── complex_schedule.json
```

```go
func TestParseConfig(t *testing.T) {
    data, err := os.ReadFile("testdata/valid_config.ini")
    if err != nil {
        t.Fatalf("failed to read test data: %v", err)
    }

    config, err := ParseConfig(data)
    if err != nil {
        t.Fatalf("parse failed: %v", err)
    }

    // Assertions...
}
```

## Fuzz Testing

See `references/fuzz-testing.md` for patterns, seed corpora and CI setup.
Quick form:

```bash
go test -fuzz=FuzzParseSchedule -fuzztime=30s ./...
```
## Parallel Test Execution

Running tests in parallel dramatically speeds up test suites. Go's testing package supports this natively.

### Basic Parallel Pattern

```go
func TestParseConfig(t *testing.T) {
    t.Parallel() // Mark test as safe to run in parallel

    tests := []struct {
        name  string
        input string
        want  Config
    }{
        {"empty", "", Config{}},
        {"basic", "key=value", Config{Key: "value"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Subtests can also be parallel; Go 1.22+ gives each iteration its own tt

            got := ParseConfig(tt.input)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### When to Use t.Parallel()

| Use Parallel | Avoid Parallel |
|--------------|----------------|
| Pure functions | Shared mutable state |
| Independent tests | Tests that modify global vars |
| Table-driven tests | Tests using shared database |
| Read-only operations | Tests with port conflicts |

### Parallel with Shared Setup

```go
func TestWithSharedSetup(t *testing.T) {
    // Shared setup runs once before parallel tests
    server := startTestServer(t)
    t.Cleanup(func() { server.Stop() })

    tests := []struct {
        name     string
        endpoint string
        want     int
    }{
        {"health", "/health", 200},
        {"metrics", "/metrics", 200},
        {"invalid", "/notfound", 404},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            resp := server.Get(tt.endpoint)
            assert.Equal(t, tt.want, resp.StatusCode)
        })
    }
}
```

### Controlling Parallelism

```bash
# Run with 4 parallel test processes
go test -parallel 4 ./...

# Run with parallelism matching CPU cores (default)
go test ./...

# Disable parallelism
go test -parallel 1 ./...
```

## Race Detection

Go's race detector finds data races at runtime. Essential for concurrent code.

### Running with Race Detection

```bash
# Run tests with race detector
go test -race ./...

# Build binary with race detection
go build -race ./cmd/app

# Run specific package
go test -race -v ./core/...
```

### Common Race Patterns and Fixes

1. **Unsynchronized map access.** Guard the map with `sync.RWMutex`. `RLock`
   is only safe when the entire operation is read-only; any write, including
   timestamps or counters, needs a full `Lock`.
2. **Loop variable capture.** Not a race since Go 1.22: each iteration has its
   own variable, so `for i := range n { go func() { use(i) }() }` needs no
   `i := i` shadow. On pre-1.22 semantics, pass the variable as an argument.
3. **Check-then-act.** Make the check and update one atomic operation under a
   single lock, e.g. a `GetOrSet(key, compute)` method.

Run `go test -race ./...` on every change touching goroutines or shared state.
### CI Integration

```yaml
# GitHub Actions - always run with race detector
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - run: go test -race -v ./...
```

## Mutation Testing

See `references/mutation-testing.md` for tooling (gremlins), CI setup and reading
scores. Mutation testing often exposes missing boundary tests, so include the
edges of every range (0 and 59 for minutes, empty and max for lengths) in
table-driven tests.
## Common Gotchas

### Integer to String Conversion

A common trap in Go: `string(rune(i))` does NOT convert an integer to its string representation:

```go
// BAD - Produces unicode codepoint, not numeric string!
for i := range 10 {
    key := "key" + string(rune(i))  // key + "\x00", "\x01", etc.
}

// GOOD - Correct integer to string conversion
for i := range 10 {
    key := "key" + strconv.Itoa(i)  // "key0", "key1", etc.
}

// Also acceptable
key := fmt.Sprintf("key%d", i)
```

**Why this happens**: `string(rune(i))` interprets `i` as a Unicode code point. `string(rune(65))` produces `"A"`, not `"65"`.

### Test Assertion Precision

Choose the right assertion for nil checks:

```go
// BAD - assert.Empty works but is less precise
assert.Empty(t, err)  // Passes for nil, "", 0, empty slices...

// GOOD - assert.Nil is explicit about intent
assert.Nil(t, err)    // Only passes for nil

// For error checking, even better:
assert.NoError(t, err)
require.NoError(t, err)  // Fails test immediately
```

### Unused Test Parameters

Always name `*testing.T` parameters to enable helper functions:

```go
// BAD - Cannot use require.NotPanics or t.Helper()
func TestSomething(_ *testing.T) {
    // ...
}

// GOOD - Full access to testing helpers
func TestSomething(t *testing.T) {
    require.NotPanics(t, func() {
        // test code
    })
}
```

## Makefile Integration

The standard `make test`, `make test-race`, `make fuzz` and related targets are
defined in `references/makefile.md`; use its template rather than inventing
per-project variants.